package repository

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// cardSequence names the sequence that numbers the cards of a project. The
// name is built from the UUID's hex digits only.
func cardSequence(projectID uuid.UUID) string {
	return "card_seq_" + strings.ReplaceAll(projectID.String(), "-", "")
}

// CreateCardSequence creates the card number sequence of a new project.
func (q *Queries) CreateCardSequence(ctx context.Context, projectID uuid.UUID) error {
	_, err := q.db.Exec(ctx, "CREATE SEQUENCE "+pgx.Identifier{cardSequence(projectID)}.Sanitize())
	return err
}

// DropCardSequence removes the sequence of a deleted project.
func (q *Queries) DropCardSequence(ctx context.Context, projectID uuid.UUID) error {
	_, err := q.db.Exec(ctx, "DROP SEQUENCE IF EXISTS "+pgx.Identifier{cardSequence(projectID)}.Sanitize())
	return err
}

// NextCardNumber takes the next card number of a project. Sequences are not
// transactional: a rolled-back creation leaves a gap, and concurrent callers
// never wait for each other.
func (q *Queries) NextCardNumber(ctx context.Context, projectID uuid.UUID) (int32, error) {
	var n int32
	err := q.db.QueryRow(ctx, "SELECT nextval($1::regclass)::integer", cardSequence(projectID)).Scan(&n)
	return n, err
}
