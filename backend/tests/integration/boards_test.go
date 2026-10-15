package integration

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/HackDanel/kanban-lab/backend/internal/domain"
	"github.com/HackDanel/kanban-lab/backend/internal/http/api"
)

func columnNames(cs []api.Column) []string {
	out := make([]string, len(cs))
	for i, c := range cs {
		out[i] = c.Name
	}
	return out
}

func boardPath(b api.BoardDetail, suffix string) string { return "/boards/" + b.Id.String() + suffix }

func TestBoards(t *testing.T) {
	f := newFixture(t)
	dev := f.member(t, domain.RoleDeveloper)

	// TC-17
	assert.Equal(t, []string{"To Do", "In Progress", "Done"}, columnNames(f.board.Columns))
	for _, c := range f.board.Columns {
		assert.True(t, c.WipLimit.IsNull())
		assert.Zero(t, c.CardCount)
	}

	boards := decode[[]api.Board](t, dev.do(http.MethodGet, projectPath(f.project, "/boards"), nil), http.StatusOK)
	require.Len(t, boards, 1)
	got := decode[api.BoardDetail](t, dev.do(http.MethodGet, boardPath(f.board, ""), nil), http.StatusOK)
	assert.Equal(t, f.board.Columns, got.Columns)

	expectError(t, dev.do(http.MethodPost, projectPath(f.project, "/boards"), api.BoardInput{Name: "x"}), http.StatusForbidden, "FORBIDDEN")
	expectError(t, f.owner.do(http.MethodPost, projectPath(f.project, "/boards"), api.BoardInput{Name: "  "}), http.StatusBadRequest, "VALIDATION_ERROR")
	expectError(t, f.owner.do(http.MethodPost, "/projects/"+uuid.NewString()+"/boards", api.BoardInput{Name: "x"}), http.StatusNotFound, "NOT_FOUND")

	renamed := decode[api.Board](t, f.owner.do(http.MethodPatch, boardPath(f.board, ""), api.BoardInput{Name: "Sprint 1"}), http.StatusOK)
	assert.Equal(t, "Sprint 1", renamed.Name)
	expectError(t, dev.do(http.MethodPatch, boardPath(f.board, ""), api.BoardInput{Name: "x"}), http.StatusForbidden, "FORBIDDEN")
	expectError(t, f.owner.do(http.MethodPatch, boardPath(f.board, ""), api.BoardInput{Name: ""}), http.StatusBadRequest, "VALIDATION_ERROR")

	card := f.card(t, dev, f.todo, "goes away with the board")
	expectError(t, dev.do(http.MethodDelete, boardPath(f.board, ""), nil), http.StatusForbidden, "FORBIDDEN")
	require.Equal(t, http.StatusNoContent, f.owner.do(http.MethodDelete, boardPath(f.board, ""), nil).status)
	expectError(t, dev.do(http.MethodGet, boardPath(f.board, ""), nil), http.StatusNotFound, "NOT_FOUND")
	expectError(t, dev.do(http.MethodGet, "/cards/"+card.Id.String(), nil), http.StatusNotFound, "NOT_FOUND")
}

// TC-18, TC-19
func TestColumns(t *testing.T) {
	f := newFixture(t)
	dev := f.member(t, domain.RoleDeveloper)
	path := boardPath(f.board, "/columns")

	review := decode[api.Column](t, f.owner.do(http.MethodPost, path, map[string]any{"name": "Review", "wipLimit": 2}), http.StatusCreated)
	assert.Equal(t, 2, review.WipLimit.MustGet())
	assert.Greater(t, review.Position, f.done.Position, "new columns go last")

	expectError(t, dev.do(http.MethodPost, path, map[string]any{"name": "x"}), http.StatusForbidden, "FORBIDDEN")
	expectError(t, f.owner.do(http.MethodPost, path, map[string]any{"name": " "}), http.StatusBadRequest, "VALIDATION_ERROR")
	expectError(t, f.owner.do(http.MethodPost, path, map[string]any{"name": "x", "wipLimit": 0}), http.StatusBadRequest, "VALIDATION_ERROR")

	colPath := "/columns/" + review.Id.String()
	renamed := decode[api.Column](t, f.owner.do(http.MethodPatch, colPath, map[string]any{"name": "Code review"}), http.StatusOK)
	assert.Equal(t, "Code review", renamed.Name)
	assert.Equal(t, 2, renamed.WipLimit.MustGet(), "an absent wipLimit is left unchanged")

	cleared := decode[api.Column](t, f.owner.do(http.MethodPatch, colPath, map[string]any{"wipLimit": nil}), http.StatusOK)
	assert.True(t, cleared.WipLimit.IsNull(), "an explicit null removes the limit")
	expectError(t, dev.do(http.MethodPatch, colPath, map[string]any{"name": "x"}), http.StatusForbidden, "FORBIDDEN")
	expectError(t, f.owner.do(http.MethodPatch, colPath, map[string]any{}), http.StatusBadRequest, "VALIDATION_ERROR")
	expectError(t, f.owner.do(http.MethodPatch, "/columns/"+uuid.NewString(), map[string]any{"name": "x"}), http.StatusNotFound, "NOT_FOUND")

	// Reordering: Review first, then after To Do.
	moved := decode[api.Column](t, f.owner.do(http.MethodPost, colPath+"/move", map[string]any{"afterColumnId": nil}), http.StatusOK)
	assert.Less(t, moved.Position, f.todo.Position)
	board := decode[api.BoardDetail](t, dev.do(http.MethodGet, boardPath(f.board, ""), nil), http.StatusOK)
	assert.Equal(t, []string{"Code review", "To Do", "In Progress", "Done"}, columnNames(board.Columns))

	decode[api.Column](t, f.owner.do(http.MethodPost, colPath+"/move", map[string]any{"afterColumnId": f.todo.Id}), http.StatusOK)
	board = decode[api.BoardDetail](t, dev.do(http.MethodGet, boardPath(f.board, ""), nil), http.StatusOK)
	assert.Equal(t, []string{"To Do", "Code review", "In Progress", "Done"}, columnNames(board.Columns))

	expectError(t, f.owner.do(http.MethodPost, colPath+"/move", map[string]any{"afterColumnId": uuid.New()}), http.StatusBadRequest, "VALIDATION_ERROR")
	expectError(t, dev.do(http.MethodPost, colPath+"/move", map[string]any{"afterColumnId": nil}), http.StatusForbidden, "FORBIDDEN")

	// Deleting: only empty columns.
	card := f.card(t, dev, f.todo, "blocks deletion")
	expectError(t, f.owner.do(http.MethodDelete, "/columns/"+f.todo.Id.String(), nil), http.StatusConflict, "COLUMN_NOT_EMPTY")
	require.Equal(t, http.StatusNoContent, dev.do(http.MethodDelete, "/cards/"+card.Id.String(), nil).status)
	expectError(t, dev.do(http.MethodDelete, colPath, nil), http.StatusForbidden, "FORBIDDEN")
	require.Equal(t, http.StatusNoContent, f.owner.do(http.MethodDelete, colPath, nil).status)
	expectError(t, f.owner.do(http.MethodDelete, colPath, nil), http.StatusNotFound, "NOT_FOUND")
}

func TestColumnRenumbering(t *testing.T) {
	f := newFixture(t)
	path := boardPath(f.board, "/columns")
	// Each new column is moved right after To Do, halving the same gap until it is exhausted.
	var names []string
	for i := range 60 {
		c := decode[api.Column](t, f.owner.do(http.MethodPost, path, map[string]any{"name": "C" + string(rune('A'+i%26)) + string(rune('a'+i/26))}), http.StatusCreated)
		decode[api.Column](t, f.owner.do(http.MethodPost, "/columns/"+c.Id.String()+"/move", map[string]any{"afterColumnId": f.todo.Id}), http.StatusOK)
		names = append([]string{c.Name}, names...)
	}
	board := decode[api.BoardDetail](t, f.owner.do(http.MethodGet, boardPath(f.board, ""), nil), http.StatusOK)
	want := append(append([]string{"To Do"}, names...), "In Progress", "Done")
	assert.Equal(t, want, columnNames(board.Columns))
}
