package repository

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestCardSearchWhere(t *testing.T) {
	board := uuid.New()

	where, args := CardSearch{BoardID: board}.where()
	assert.Equal(t, "c.board_id = $1", where)
	assert.Equal(t, []any{board}, args)

	column, assignee := uuid.New(), uuid.New()
	priority, pattern := "high", "%'; DROP TABLE cards; --%"
	labels := []uuid.UUID{uuid.New(), uuid.New()}
	where, args = CardSearch{
		BoardID: board, ColumnID: &column, AssigneeID: &assignee, Priority: &priority,
		Pattern: &pattern, LabelIDs: labels,
	}.where()
	assert.Equal(t, "c.board_id = $1 AND c.column_id = $2 AND c.assignee_id = $3 AND c.priority = $4"+
		" AND (c.title ILIKE $5 OR c.description ILIKE $5)"+
		" AND c.id IN (SELECT card_id FROM card_labels WHERE label_id = ANY ($6) GROUP BY card_id HAVING count(*) = $7)", where)
	assert.Equal(t, []any{board, column, assignee, priority, pattern, labels, 2}, args)
	assert.NotContains(t, where, "DROP", "values travel as parameters only")

	where, args = CardSearch{BoardID: board, Pattern: &pattern}.where()
	assert.Equal(t, "c.board_id = $1 AND (c.title ILIKE $2 OR c.description ILIKE $2)", where)
	assert.Len(t, args, 2)
}
