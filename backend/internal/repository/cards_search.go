package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// CardSearch is a filtered, paged query over the cards of a board. Nil and
// empty fields do not filter.
type CardSearch struct {
	BoardID    uuid.UUID
	ColumnID   *uuid.UUID
	AssigneeID *uuid.UUID
	Priority   *string
	Pattern    *string // ILIKE pattern, already escaped
	LabelIDs   []uuid.UUID
	Limit      int32
	Offset     int32
}

// where builds the WHERE clause from the filters that are set.
//
// The query is assembled per request instead of being one sqlc query with
// "($n IS NULL OR column = $n)" conditions: PostgreSQL plans such a prepared
// statement once for all parameter values and ends up scanning the whole
// table. With only the active conditions present, the planner uses the
// (column_id, position) and trigram indexes. Values are still passed as
// parameters, never spliced into the SQL.
func (f CardSearch) where() (string, []any) {
	conds := []string{"c.board_id = $1"}
	args := []any{f.BoardID}
	add := func(cond string, arg any) {
		args = append(args, arg)
		conds = append(conds, strings.ReplaceAll(cond, "$?", fmt.Sprintf("$%d", len(args))))
	}
	if f.ColumnID != nil {
		add("c.column_id = $?", *f.ColumnID)
	}
	if f.AssigneeID != nil {
		add("c.assignee_id = $?", *f.AssigneeID)
	}
	if f.Priority != nil {
		add("c.priority = $?", *f.Priority)
	}
	if f.Pattern != nil {
		add("(c.title ILIKE $? OR c.description ILIKE $?)", *f.Pattern)
	}
	if len(f.LabelIDs) > 0 {
		args = append(args, f.LabelIDs, len(f.LabelIDs))
		conds = append(conds, fmt.Sprintf(
			"c.id IN (SELECT card_id FROM card_labels WHERE label_id = ANY ($%d) GROUP BY card_id HAVING count(*) = $%d)",
			len(args)-1, len(args)))
	}
	return strings.Join(conds, " AND "), args
}

const cardViewColumns = `c.id, c.project_id, c.board_id, c.column_id, c.number, c.title, c.description, c.priority,
	c.position, c.assignee_id, c.reporter_id, c.due_date, c.created_at, c.updated_at,
	c.label_ids, c.comment_count, c.attachment_count`

// SearchCards returns one page of matching cards ordered by column and position.
//
// The page is chosen from the cards table first and only its rows are joined
// with card_views: the view computes labels and counters with subqueries, and
// selecting from it directly would evaluate them for every row skipped by
// OFFSET as well.
func (q *Queries) SearchCards(ctx context.Context, f CardSearch) ([]CardView, error) {
	where, args := f.where()
	order := "c.column_id, c.position, c.id"
	if f.ColumnID != nil {
		order = "c.position, c.id"
	}
	args = append(args, f.Limit, f.Offset)
	// #nosec G201 -- only fixed fragments are formatted in; values are parameters
	sql := fmt.Sprintf(`SELECT %s FROM (
		SELECT c.id, c.column_id, c.position FROM cards c WHERE %s ORDER BY %s LIMIT $%d OFFSET $%d
	) page JOIN card_views c ON c.id = page.id
	ORDER BY page.column_id, page.position, page.id`,
		cardViewColumns, where, order, len(args)-1, len(args))

	rows, err := q.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []CardView{}
	for rows.Next() {
		var c CardView
		if err := rows.Scan(&c.ID, &c.ProjectID, &c.BoardID, &c.ColumnID, &c.Number, &c.Title, &c.Description,
			&c.Priority, &c.Position, &c.AssigneeID, &c.ReporterID, &c.DueDate, &c.CreatedAt, &c.UpdatedAt,
			&c.LabelIds, &c.CommentCount, &c.AttachmentCount); err != nil {
			return nil, err
		}
		items = append(items, c)
	}
	return items, rows.Err()
}

// CountSearchCards counts all cards matching the filters, ignoring paging.
func (q *Queries) CountSearchCards(ctx context.Context, f CardSearch) (int64, error) {
	where, args := f.where()
	var n int64
	err := q.db.QueryRow(ctx, "SELECT count(*) FROM cards c WHERE "+where, args...).Scan(&n) // #nosec G202 -- see where()
	return n, err
}
