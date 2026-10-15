package integration

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/HackDanel/kanban-lab/backend/internal/domain"
	"github.com/HackDanel/kanban-lab/backend/internal/http/api"
)

func cardPath(c api.Card, suffix string) string { return "/cards/" + c.Id.String() + suffix }

func columnPath(c api.Column, suffix string) string { return "/columns/" + c.Id.String() + suffix }

func activityActions(entries []api.Activity) []string {
	out := make([]string, len(entries))
	for i, e := range entries {
		out[i] = string(e.Action)
		if e.Field.IsSpecified() && !e.Field.IsNull() {
			out[i] += ":" + e.Field.MustGet()
		}
	}
	return out
}

// TC-02
func TestCreateCard(t *testing.T) {
	f := newFixture(t)
	dev := f.member(t, domain.RoleDeveloper)
	label := decode[api.Label](t, f.owner.do(http.MethodPost, projectPath(f.project, "/labels"), api.LabelInput{Name: "bug", Color: "#ff0000"}), http.StatusCreated)

	card := decode[api.Card](t, dev.do(http.MethodPost, columnPath(f.todo, "/cards"), map[string]any{
		"title": "  Login page  ", "description": "Описание", "priority": "high",
		"assigneeId": dev.user.Id, "dueDate": "2026-12-31", "labelIds": []uuid.UUID{label.Id, label.Id},
	}), http.StatusCreated)
	assert.Equal(t, "Login page", card.Title)
	assert.Equal(t, api.High, card.Priority)
	assert.Equal(t, dev.user.Id, card.AssigneeId.MustGet())
	assert.Equal(t, "2026-12-31", card.DueDate.MustGet().String())
	assert.Equal(t, []uuid.UUID{label.Id}, card.LabelIds, "duplicate labels are collapsed")
	assert.Equal(t, dev.user.Id, card.ReporterId)
	assert.Equal(t, f.todo.Id, card.ColumnId)
	assert.Equal(t, 1, card.Number)

	second := f.card(t, dev, f.todo, "Second")
	assert.Equal(t, 2, second.Number, "numbers are sequential within a project")
	assert.Equal(t, api.Medium, second.Priority, "medium is the default priority")
	assert.True(t, second.AssigneeId.IsNull())
	assert.Greater(t, second.Position, card.Position, "new cards go to the end of the column")

	got := decode[api.Card](t, dev.do(http.MethodGet, cardPath(card, ""), nil), http.StatusOK)
	assert.Equal(t, card, got)

	board := decode[api.BoardDetail](t, dev.do(http.MethodGet, boardPath(f.board, ""), nil), http.StatusOK)
	assert.Equal(t, 2, board.Columns[0].CardCount)
}

// TC-05
func TestCreateCardValidation(t *testing.T) {
	f := newFixture(t)
	dev := f.member(t, domain.RoleDeveloper)
	outsider := newUser(t, domain.RoleDeveloper)
	other := newFixture(t)
	foreignLabel := decode[api.Label](t, other.owner.do(http.MethodPost, projectPath(other.project, "/labels"), api.LabelInput{Name: "x", Color: "#000000"}), http.StatusCreated)
	path := columnPath(f.todo, "/cards")

	e := expectError(t, dev.do(http.MethodPost, path, map[string]any{"title": "   "}), http.StatusBadRequest, "VALIDATION_ERROR")
	assert.Equal(t, "Название обязательно", fieldErrors(e)["title"])
	e = expectError(t, dev.do(http.MethodPost, path, map[string]any{"title": ""}), http.StatusBadRequest, "VALIDATION_ERROR")
	assert.Contains(t, fieldErrors(e), "title")
	e = expectError(t, dev.do(http.MethodPost, path, map[string]any{"title": "x", "priority": "urgent"}), http.StatusBadRequest, "VALIDATION_ERROR")
	assert.Contains(t, fieldErrors(e), "priority")
	e = expectError(t, dev.do(http.MethodPost, path, map[string]any{"title": "x", "assigneeId": outsider.user.Id}), http.StatusBadRequest, "VALIDATION_ERROR")
	assert.Equal(t, "Исполнитель должен быть участником проекта", fieldErrors(e)["assigneeId"])
	e = expectError(t, dev.do(http.MethodPost, path, map[string]any{"title": "x", "labelIds": []uuid.UUID{foreignLabel.Id}}), http.StatusBadRequest, "VALIDATION_ERROR")
	assert.Contains(t, fieldErrors(e), "labelIds")
	expectError(t, dev.do(http.MethodPost, path, map[string]any{"title": "x", "dueDate": "31.12.2026"}), http.StatusBadRequest, "VALIDATION_ERROR")
	expectError(t, dev.do(http.MethodPost, path, map[string]any{"title": "x", "extra": true}), http.StatusBadRequest, "VALIDATION_ERROR")

	expectError(t, outsider.do(http.MethodPost, path, map[string]any{"title": "x"}), http.StatusForbidden, "FORBIDDEN")
	expectError(t, dev.do(http.MethodPost, "/columns/"+uuid.NewString()+"/cards", map[string]any{"title": "x"}), http.StatusNotFound, "NOT_FOUND")
	assert.Zero(t, f.listCards(t, dev, "").Total)
}

// TC-20
func TestUpdateCardRecordsActivity(t *testing.T) {
	f := newFixture(t)
	dev := f.member(t, domain.RoleDeveloper)
	label := decode[api.Label](t, f.owner.do(http.MethodPost, projectPath(f.project, "/labels"), api.LabelInput{Name: "bug", Color: "#ff0000"}), http.StatusCreated)
	card := f.card(t, dev, f.todo, "Draft")

	updated := decode[api.Card](t, dev.do(http.MethodPatch, cardPath(card, ""), map[string]any{
		"title": "Final", "description": "Text", "priority": "critical",
		"assigneeId": f.owner.user.Id, "dueDate": "2027-01-15", "labelIds": []uuid.UUID{label.Id},
	}), http.StatusOK)
	assert.Equal(t, "Final", updated.Title)
	assert.Equal(t, "Text", updated.Description)
	assert.Equal(t, api.Critical, updated.Priority)
	assert.Equal(t, f.owner.user.Id, updated.AssigneeId.MustGet())
	assert.Equal(t, "2027-01-15", updated.DueDate.MustGet().String())
	assert.Equal(t, []uuid.UUID{label.Id}, updated.LabelIds)

	// Null clears, absent leaves as is, unchanged values are not logged.
	cleared := decode[api.Card](t, dev.do(http.MethodPatch, cardPath(card, ""), map[string]any{
		"assigneeId": nil, "dueDate": nil, "title": "Final", "labelIds": []uuid.UUID{},
	}), http.StatusOK)
	assert.True(t, cleared.AssigneeId.IsNull())
	assert.True(t, cleared.DueDate.IsNull())
	assert.Empty(t, cleared.LabelIds)
	assert.Equal(t, api.Critical, cleared.Priority)

	same := decode[api.Card](t, dev.do(http.MethodPatch, cardPath(card, ""), map[string]any{"title": "Final"}), http.StatusOK)
	assert.Equal(t, cleared.UpdatedAt, same.UpdatedAt, "a no-op update does not touch the card")

	history := decode[[]api.Activity](t, dev.do(http.MethodGet, cardPath(card, "/activity"), nil), http.StatusOK)
	assert.ElementsMatch(t, []string{
		"created", "updated:title", "updated:description", "updated:priority", "assigned:assigneeId",
		"updated:dueDate", "labels_changed:labelIds",
		"assigned:assigneeId", "updated:dueDate", "labels_changed:labelIds",
	}, activityActions(history))
	assert.Equal(t, "created", string(history[len(history)-1].Action), "newest entries come first")
	for _, entry := range history {
		if entry.Field.IsSpecified() && !entry.Field.IsNull() && entry.Field.MustGet() == "title" {
			assert.Equal(t, "Draft", entry.OldValue.MustGet())
			assert.Equal(t, "Final", entry.NewValue.MustGet())
		}
	}
	page := decode[[]api.Activity](t, dev.do(http.MethodGet, cardPath(card, "/activity?limit=2&offset=1"), nil), http.StatusOK)
	assert.Equal(t, history[1:3], page)

	expectError(t, dev.do(http.MethodPatch, cardPath(card, ""), map[string]any{}), http.StatusBadRequest, "VALIDATION_ERROR")
	expectError(t, dev.do(http.MethodPatch, cardPath(card, ""), map[string]any{"title": " "}), http.StatusBadRequest, "VALIDATION_ERROR")
	expectError(t, dev.do(http.MethodPatch, "/cards/"+uuid.NewString(), map[string]any{"title": "x"}), http.StatusNotFound, "NOT_FOUND")
	expectError(t, newUser(t, domain.RoleDeveloper).do(http.MethodPatch, cardPath(card, ""), map[string]any{"title": "x"}), http.StatusForbidden, "FORBIDDEN")
}

// TC-03, TC-07
func TestMoveCard(t *testing.T) {
	f := newFixture(t)
	dev := f.member(t, domain.RoleDeveloper)
	a, b, c := f.card(t, dev, f.todo, "A"), f.card(t, dev, f.todo, "B"), f.card(t, dev, f.todo, "C")

	moved := decode[api.Card](t, dev.do(http.MethodPost, cardPath(a, "/move"), map[string]any{"columnId": f.doing.Id}), http.StatusOK)
	assert.Equal(t, f.doing.Id, moved.ColumnId)
	persisted := decode[api.Card](t, dev.do(http.MethodGet, cardPath(a, ""), nil), http.StatusOK)
	assert.Equal(t, f.doing.Id, persisted.ColumnId, "the move is stored")

	history := decode[[]api.Activity](t, dev.do(http.MethodGet, cardPath(a, "/activity"), nil), http.StatusOK)
	require.Equal(t, "moved", string(history[0].Action))
	assert.Equal(t, "To Do", history[0].OldValue.MustGet())
	assert.Equal(t, "In Progress", history[0].NewValue.MustGet())

	// Reordering inside a column: C first, then B after C.
	decode[api.Card](t, dev.do(http.MethodPost, cardPath(c, "/move"), map[string]any{"columnId": f.todo.Id, "afterCardId": nil}), http.StatusOK)
	assert.Equal(t, []string{"C", "B"}, titles(f.listCards(t, dev, "columnId="+f.todo.Id.String()).Items))
	decode[api.Card](t, dev.do(http.MethodPost, cardPath(b, "/move"), map[string]any{"columnId": f.todo.Id, "afterCardId": c.Id}), http.StatusOK)
	assert.Equal(t, []string{"C", "B"}, titles(f.listCards(t, dev, "columnId="+f.todo.Id.String()).Items))
	decode[api.Card](t, dev.do(http.MethodPost, cardPath(c, "/move"), map[string]any{"columnId": f.todo.Id, "afterCardId": b.Id}), http.StatusOK)
	assert.Equal(t, []string{"B", "C"}, titles(f.listCards(t, dev, "columnId="+f.todo.Id.String()).Items))
	history = decode[[]api.Activity](t, dev.do(http.MethodGet, cardPath(c, "/activity"), nil), http.StatusOK)
	assert.Equal(t, []string{"created"}, activityActions(history), "reordering within a column is not logged")

	// WIP limit of 1 on In Progress, which already holds A.
	decode[api.Column](t, f.owner.do(http.MethodPatch, columnPath(f.doing, ""), map[string]any{"wipLimit": 1}), http.StatusOK)
	e := expectError(t, dev.do(http.MethodPost, cardPath(b, "/move"), map[string]any{"columnId": f.doing.Id}), http.StatusConflict, "WIP_LIMIT_EXCEEDED")
	assert.Contains(t, e.Message, "In Progress")
	assert.Equal(t, f.todo.Id, decode[api.Card](t, dev.do(http.MethodGet, cardPath(b, ""), nil), http.StatusOK).ColumnId)
	expectError(t, dev.do(http.MethodPost, columnPath(f.doing, "/cards"), map[string]any{"title": "D"}), http.StatusConflict, "WIP_LIMIT_EXCEEDED")
	// Reordering within the full column is still allowed.
	decode[api.Card](t, dev.do(http.MethodPost, cardPath(a, "/move"), map[string]any{"columnId": f.doing.Id, "afterCardId": nil}), http.StatusOK)

	other := newFixture(t)
	e = expectError(t, dev.do(http.MethodPost, cardPath(b, "/move"), map[string]any{"columnId": other.todo.Id}), http.StatusBadRequest, "VALIDATION_ERROR")
	assert.Contains(t, fieldErrors(e), "columnId")
	expectError(t, dev.do(http.MethodPost, cardPath(b, "/move"), map[string]any{"columnId": uuid.New()}), http.StatusBadRequest, "VALIDATION_ERROR")
	e = expectError(t, dev.do(http.MethodPost, cardPath(b, "/move"), map[string]any{"columnId": f.done.Id, "afterCardId": a.Id}), http.StatusBadRequest, "VALIDATION_ERROR")
	assert.Contains(t, fieldErrors(e), "afterCardId")
	expectError(t, dev.do(http.MethodPost, "/cards/"+uuid.NewString()+"/move", map[string]any{"columnId": f.done.Id}), http.StatusNotFound, "NOT_FOUND")
	expectError(t, newUser(t, domain.RoleDeveloper).do(http.MethodPost, cardPath(b, "/move"), map[string]any{"columnId": f.done.Id}), http.StatusForbidden, "FORBIDDEN")
}

// TC-28: concurrent moves cannot overfill a column.
func TestWipLimitUnderConcurrency(t *testing.T) {
	f := newFixture(t)
	dev := f.member(t, domain.RoleDeveloper)
	decode[api.Column](t, f.owner.do(http.MethodPatch, columnPath(f.doing, ""), map[string]any{"wipLimit": 1}), http.StatusOK)

	const n = 10
	cards := make([]api.Card, n)
	for i := range cards {
		cards[i] = f.card(t, dev, f.todo, fmt.Sprintf("card %d", i))
	}
	statuses := make([]int, n)
	var wg sync.WaitGroup
	for i := range cards {
		wg.Go(func() {
			statuses[i] = dev.do(http.MethodPost, cardPath(cards[i], "/move"), map[string]any{"columnId": f.doing.Id}).status
		})
	}
	wg.Wait()

	ok, conflicts := 0, 0
	for _, s := range statuses {
		switch s {
		case http.StatusOK:
			ok++
		case http.StatusConflict:
			conflicts++
		}
	}
	assert.Equal(t, 1, ok)
	assert.Equal(t, n-1, conflicts)
	assert.Equal(t, 1, f.listCards(t, dev, "columnId="+f.doing.Id.String()).Total)
}

func TestCardRenumbering(t *testing.T) {
	f := newFixture(t)
	anchor := f.card(t, f.owner, f.doing, "anchor")
	tail := f.card(t, f.owner, f.doing, "tail")
	var want []string
	for i := range 60 {
		c := f.card(t, f.owner, f.todo, fmt.Sprintf("c%02d", i))
		decode[api.Card](t, f.owner.do(http.MethodPost, cardPath(c, "/move"), map[string]any{"columnId": f.doing.Id, "afterCardId": anchor.Id}), http.StatusOK)
		want = append([]string{c.Title}, want...)
	}
	want = append(append([]string{anchor.Title}, want...), tail.Title)
	got := f.listCards(t, f.owner, "columnId="+f.doing.Id.String()+"&limit=500")
	assert.Equal(t, want, titles(got.Items))
}

// TC-06, TC-24, TC-26
func TestFilterAndSearchCards(t *testing.T) {
	f := newFixture(t)
	dev := f.member(t, domain.RoleDeveloper)
	bug := decode[api.Label](t, f.owner.do(http.MethodPost, projectPath(f.project, "/labels"), api.LabelInput{Name: "bug", Color: "#ff0000"}), http.StatusCreated)
	ui := decode[api.Label](t, f.owner.do(http.MethodPost, projectPath(f.project, "/labels"), api.LabelInput{Name: "ui", Color: "#00ff00"}), http.StatusCreated)

	create := func(column api.Column, body map[string]any) {
		decode[api.Card](t, dev.do(http.MethodPost, columnPath(column, "/cards"), body), http.StatusCreated)
	}
	create(f.todo, map[string]any{"title": "Login form", "description": "email field", "labelIds": []uuid.UUID{bug.Id, ui.Id}, "priority": "high", "assigneeId": dev.user.Id})
	create(f.todo, map[string]any{"title": "Logout", "labelIds": []uuid.UUID{bug.Id}})
	create(f.doing, map[string]any{"title": "Reports", "description": "100% coverage", "priority": "high"})
	create(f.done, map[string]any{"title": "Setup_CI", "assigneeId": dev.user.Id})

	q := func(params url.Values) []string { return titles(f.listCards(t, dev, params.Encode()).Items) }

	assert.Len(t, q(nil), 4)
	assert.ElementsMatch(t, []string{"Login form", "Logout"}, q(url.Values{"q": {"LOG"}}), "search is case-insensitive")
	assert.Equal(t, []string{"Login form"}, q(url.Values{"q": {"email"}}), "description is searched too")
	assert.Equal(t, []string{"Reports"}, q(url.Values{"q": {"%"}}), "% is matched literally")
	assert.Equal(t, []string{"Setup_CI"}, q(url.Values{"q": {"p_c"}}), "_ is matched literally")
	assert.Empty(t, q(url.Values{"q": {"' OR 1=1 --"}}))
	assert.Len(t, q(url.Values{"q": {"   "}}), 4, "a blank query does not filter")

	assert.ElementsMatch(t, []string{"Login form", "Setup_CI"}, q(url.Values{"assigneeId": {dev.user.Id.String()}}))
	assert.ElementsMatch(t, []string{"Login form", "Reports"}, q(url.Values{"priority": {"high"}}))
	assert.ElementsMatch(t, []string{"Login form", "Logout"}, q(url.Values{"labelId": {bug.Id.String()}}))
	assert.Equal(t, []string{"Login form"}, q(url.Values{"labelId": {bug.Id.String(), ui.Id.String()}}), "all labels must match")
	assert.Equal(t, []string{"Login form"}, q(url.Values{"labelId": {bug.Id.String()}, "priority": {"high"}, "q": {"form"}}))
	assert.Equal(t, []string{"Reports"}, q(url.Values{"columnId": {f.doing.Id.String()}}))

	page := f.listCards(t, dev, "limit=2&offset=1")
	assert.Len(t, page.Items, 2)
	assert.Equal(t, 4, page.Total)

	expectError(t, dev.do(http.MethodGet, boardPath(f.board, "/cards?priority=urgent"), nil), http.StatusBadRequest, "VALIDATION_ERROR")
	expectError(t, dev.do(http.MethodGet, boardPath(f.board, "/cards?limit=0"), nil), http.StatusBadRequest, "VALIDATION_ERROR")
	expectError(t, dev.do(http.MethodGet, boardPath(f.board, "/cards?assigneeId=me"), nil), http.StatusBadRequest, "VALIDATION_ERROR")
	expectError(t, newUser(t, domain.RoleDeveloper).do(http.MethodGet, boardPath(f.board, "/cards"), nil), http.StatusForbidden, "FORBIDDEN")
}

// TC-25: markup is stored and returned as plain text, never interpreted by the API.
func TestCardTextIsStoredVerbatim(t *testing.T) {
	f := newFixture(t)
	payload := `<img src=x onerror=alert(1)>`
	card := decode[api.Card](t, f.owner.do(http.MethodPost, columnPath(f.todo, "/cards"), map[string]any{"title": payload, "description": payload}), http.StatusCreated)
	r := f.owner.do(http.MethodGet, cardPath(card, ""), nil)
	assert.Equal(t, "application/json", r.header.Get("Content-Type"))
	assert.Equal(t, "nosniff", r.header.Get("X-Content-Type-Options"))
	got := decode[api.Card](t, r, http.StatusOK)
	assert.Equal(t, payload, got.Title)
	assert.Equal(t, payload, got.Description)
}

func TestDeleteCard(t *testing.T) {
	f := newFixture(t)
	dev := f.member(t, domain.RoleDeveloper)
	card := f.card(t, dev, f.todo, "temp")
	expectError(t, newUser(t, domain.RoleDeveloper).do(http.MethodDelete, cardPath(card, ""), nil), http.StatusForbidden, "FORBIDDEN")
	require.Equal(t, http.StatusNoContent, dev.do(http.MethodDelete, cardPath(card, ""), nil).status)
	expectError(t, dev.do(http.MethodGet, cardPath(card, ""), nil), http.StatusNotFound, "NOT_FOUND")
	expectError(t, dev.do(http.MethodDelete, cardPath(card, ""), nil), http.StatusNotFound, "NOT_FOUND")
	expectError(t, dev.do(http.MethodGet, cardPath(card, "/activity"), nil), http.StatusNotFound, "NOT_FOUND")
}

func TestCardNumbersUnderConcurrency(t *testing.T) {
	f := newFixture(t)
	const n = 20
	numbers := make([]int, n)
	var wg sync.WaitGroup
	for i := range n {
		wg.Go(func() {
			numbers[i] = decode[api.Card](t, f.owner.do(http.MethodPost, columnPath(f.todo, "/cards"), map[string]any{"title": fmt.Sprintf("c%d", i)}), http.StatusCreated).Number
		})
	}
	wg.Wait()
	seen := map[int]bool{}
	for _, num := range numbers {
		assert.False(t, seen[num], "number %d issued twice", num)
		seen[num] = true
		assert.True(t, num >= 1 && num <= n, "number %d out of range", num)
	}

	var exists bool
	seq := "card_seq_" + strings.ReplaceAll(f.project.Id.String(), "-", "")
	require.NoError(t, pool.QueryRow(context.Background(), "SELECT to_regclass($1) IS NOT NULL", seq).Scan(&exists))
	assert.True(t, exists)
	require.Equal(t, http.StatusNoContent, f.owner.do(http.MethodDelete, projectPath(f.project, ""), nil).status)
	require.NoError(t, pool.QueryRow(context.Background(), "SELECT to_regclass($1) IS NOT NULL", seq).Scan(&exists))
	assert.False(t, exists, "the sequence is dropped with the project")
}
