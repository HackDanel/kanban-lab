package integration

import (
	"net/http"
	"slices"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/HackDanel/kanban-lab/backend/internal/domain"
	"github.com/HackDanel/kanban-lab/backend/internal/http/api"
)

func projectPath(p api.Project, suffix string) string { return "/projects/" + p.Id.String() + suffix }

func projectIDs(ps []api.Project) []uuid.UUID {
	out := make([]uuid.UUID, len(ps))
	for i, p := range ps {
		out[i] = p.Id
	}
	return out
}

func TestUsers(t *testing.T) {
	root := admin(t)
	dev := newUser(t, domain.RoleDeveloper)

	page := decode[api.UserPage](t, dev.do(http.MethodGet, "/users?q="+string(dev.user.Email), nil), http.StatusOK)
	require.Equal(t, 1, page.Total)
	assert.Equal(t, dev.user.Id, page.Items[0].Id)

	page = decode[api.UserPage](t, dev.do(http.MethodGet, "/users?limit=1", nil), http.StatusOK)
	assert.Len(t, page.Items, 1)
	assert.Greater(t, page.Total, 1)

	expectError(t, dev.do(http.MethodGet, "/users?limit=501", nil), http.StatusBadRequest, "VALIDATION_ERROR")

	// Only administrators change roles.
	expectError(t, dev.do(http.MethodPut, "/users/"+dev.user.Id.String()+"/role", api.SetRoleRequest{Role: api.Admin}), http.StatusForbidden, "FORBIDDEN")
	promoted := decode[api.User](t, root.do(http.MethodPut, "/users/"+dev.user.Id.String()+"/role", api.SetRoleRequest{Role: api.Manager}), http.StatusOK)
	assert.Equal(t, api.Manager, promoted.Role)

	expectError(t, root.do(http.MethodPut, "/users/"+root.user.Id.String()+"/role", api.SetRoleRequest{Role: api.Developer}), http.StatusBadRequest, "VALIDATION_ERROR")
	expectError(t, root.do(http.MethodPut, "/users/"+uuid.NewString()+"/role", api.SetRoleRequest{Role: api.Manager}), http.StatusNotFound, "NOT_FOUND")
	expectError(t, root.do(http.MethodPut, "/users/"+dev.user.Id.String()+"/role", map[string]string{"role": "owner"}), http.StatusBadRequest, "VALIDATION_ERROR")
}

func TestCreateProjectPermissions(t *testing.T) {
	dev := newUser(t, domain.RoleDeveloper)
	expectError(t, dev.do(http.MethodPost, "/projects", api.ProjectInput{Key: uniqueKey(), Name: "X"}), http.StatusForbidden, "FORBIDDEN")

	// TC-14
	manager := newUser(t, domain.RoleManager)
	key := uniqueKey()
	p := decode[api.Project](t, manager.do(http.MethodPost, "/projects", api.ProjectInput{Key: key, Name: "  Сайт  ", Description: ptr("Описание")}), http.StatusCreated)
	assert.Equal(t, key, p.Key)
	assert.Equal(t, "Сайт", p.Name)
	assert.Equal(t, manager.user.Id, p.OwnerId)

	members := decode[[]api.User](t, manager.do(http.MethodGet, projectPath(p, "/members"), nil), http.StatusOK)
	require.Len(t, members, 1, "the creator becomes a member")
	assert.Equal(t, manager.user.Id, members[0].Id)

	expectError(t, manager.do(http.MethodPost, "/projects", api.ProjectInput{Key: key, Name: "Dup"}), http.StatusConflict, "CONFLICT")
	e := expectError(t, manager.do(http.MethodPost, "/projects", map[string]string{"key": "bad key", "name": " "}), http.StatusBadRequest, "VALIDATION_ERROR")
	assert.Contains(t, fieldErrors(e), "key")
	e = expectError(t, manager.do(http.MethodPost, "/projects", api.ProjectInput{Key: uniqueKey(), Name: "   "}), http.StatusBadRequest, "VALIDATION_ERROR")
	assert.Equal(t, "Название обязательно", fieldErrors(e)["name"])
}

func TestProjectVisibility(t *testing.T) {
	f := newFixture(t)
	member := f.member(t, domain.RoleDeveloper)
	outsider := newUser(t, domain.RoleManager)
	root := admin(t)

	assert.Contains(t, projectIDs(decode[[]api.Project](t, member.do(http.MethodGet, "/projects", nil), http.StatusOK)), f.project.Id)
	assert.NotContains(t, projectIDs(decode[[]api.Project](t, outsider.do(http.MethodGet, "/projects", nil), http.StatusOK)), f.project.Id)
	assert.Contains(t, projectIDs(decode[[]api.Project](t, root.do(http.MethodGet, "/projects", nil), http.StatusOK)), f.project.Id)

	decode[api.Project](t, member.do(http.MethodGet, projectPath(f.project, ""), nil), http.StatusOK)
	decode[api.Project](t, root.do(http.MethodGet, projectPath(f.project, ""), nil), http.StatusOK)
	// TC-16
	expectError(t, outsider.do(http.MethodGet, projectPath(f.project, ""), nil), http.StatusForbidden, "FORBIDDEN")
	expectError(t, outsider.do(http.MethodGet, "/boards/"+f.board.Id.String(), nil), http.StatusForbidden, "FORBIDDEN")
	expectError(t, member.do(http.MethodGet, "/projects/"+uuid.NewString(), nil), http.StatusNotFound, "NOT_FOUND")
	expectError(t, member.do(http.MethodGet, "/projects/not-a-uuid", nil), http.StatusBadRequest, "VALIDATION_ERROR")
}

func TestUpdateAndDeleteProject(t *testing.T) {
	f := newFixture(t)
	otherManager := f.member(t, domain.RoleManager)
	dev := f.member(t, domain.RoleDeveloper)
	root := admin(t)

	updated := decode[api.Project](t, f.owner.do(http.MethodPatch, projectPath(f.project, ""), api.ProjectPatch{Name: ptr("Новое имя")}), http.StatusOK)
	assert.Equal(t, "Новое имя", updated.Name)
	assert.Equal(t, f.project.Description, updated.Description)

	updated = decode[api.Project](t, root.do(http.MethodPatch, projectPath(f.project, ""), api.ProjectPatch{Description: ptr("by admin")}), http.StatusOK)
	assert.Equal(t, "by admin", updated.Description)

	expectError(t, f.owner.do(http.MethodPatch, projectPath(f.project, ""), map[string]any{}), http.StatusBadRequest, "VALIDATION_ERROR")
	expectError(t, f.owner.do(http.MethodPatch, projectPath(f.project, ""), api.ProjectPatch{Name: ptr(" ")}), http.StatusBadRequest, "VALIDATION_ERROR")

	// TC-15: a manager who is a member but not the owner, and a developer, may not change or delete it.
	expectError(t, otherManager.do(http.MethodPatch, projectPath(f.project, ""), api.ProjectPatch{Name: ptr("x")}), http.StatusForbidden, "FORBIDDEN")
	expectError(t, otherManager.do(http.MethodDelete, projectPath(f.project, ""), nil), http.StatusForbidden, "FORBIDDEN")
	// TC-04
	expectError(t, dev.do(http.MethodDelete, projectPath(f.project, ""), nil), http.StatusForbidden, "FORBIDDEN")

	card := f.card(t, dev, f.todo, "will be removed with the project")
	att := decode[api.Attachment](t, dev.upload(card.Id, "a.txt", "text/plain", []byte("x")), http.StatusCreated)

	require.Equal(t, http.StatusNoContent, f.owner.do(http.MethodDelete, projectPath(f.project, ""), nil).status)
	expectError(t, root.do(http.MethodGet, projectPath(f.project, ""), nil), http.StatusNotFound, "NOT_FOUND")
	expectError(t, root.do(http.MethodGet, "/cards/"+card.Id.String(), nil), http.StatusNotFound, "NOT_FOUND")
	expectError(t, root.do(http.MethodGet, "/attachments/"+att.Id.String()+"/download", nil), http.StatusNotFound, "NOT_FOUND")
	expectError(t, root.do(http.MethodDelete, projectPath(f.project, ""), nil), http.StatusNotFound, "NOT_FOUND")
}

func TestProjectMembers(t *testing.T) {
	f := newFixture(t)
	dev := f.member(t, domain.RoleDeveloper)
	stranger := newUser(t, domain.RoleDeveloper)

	members := decode[[]api.User](t, dev.do(http.MethodGet, projectPath(f.project, "/members"), nil), http.StatusOK)
	assert.Len(t, members, 2)

	// Adding twice is harmless; developers cannot add members; unknown users are reported.
	require.Equal(t, http.StatusNoContent, f.owner.do(http.MethodPost, projectPath(f.project, "/members"), api.AddMemberRequest{UserId: dev.user.Id}).status)
	expectError(t, dev.do(http.MethodPost, projectPath(f.project, "/members"), api.AddMemberRequest{UserId: stranger.user.Id}), http.StatusForbidden, "FORBIDDEN")
	expectError(t, f.owner.do(http.MethodPost, projectPath(f.project, "/members"), api.AddMemberRequest{UserId: uuid.New()}), http.StatusNotFound, "NOT_FOUND")

	// Removing a member unassigns their cards.
	card := f.card(t, f.owner, f.todo, "assigned")
	card = decode[api.Card](t, f.owner.do(http.MethodPatch, "/cards/"+card.Id.String(), map[string]any{"assigneeId": dev.user.Id}), http.StatusOK)
	require.Equal(t, dev.user.Id, card.AssigneeId.MustGet())

	require.Equal(t, http.StatusNoContent, f.owner.do(http.MethodDelete, projectPath(f.project, "/members/"+dev.user.Id.String()), nil).status)
	card = decode[api.Card](t, f.owner.do(http.MethodGet, "/cards/"+card.Id.String(), nil), http.StatusOK)
	assert.True(t, card.AssigneeId.IsNull())
	expectError(t, dev.do(http.MethodGet, projectPath(f.project, ""), nil), http.StatusForbidden, "FORBIDDEN")

	expectError(t, f.owner.do(http.MethodDelete, projectPath(f.project, "/members/"+dev.user.Id.String()), nil), http.StatusNotFound, "NOT_FOUND")
	e := expectError(t, f.owner.do(http.MethodDelete, projectPath(f.project, "/members/"+f.owner.user.Id.String()), nil), http.StatusBadRequest, "VALIDATION_ERROR")
	assert.Contains(t, fieldErrors(e), "userId")
}

func TestLabels(t *testing.T) {
	f := newFixture(t)
	dev := f.member(t, domain.RoleDeveloper)

	label := decode[api.Label](t, f.owner.do(http.MethodPost, projectPath(f.project, "/labels"), api.LabelInput{Name: "bug", Color: "#FF0000"}), http.StatusCreated)
	assert.Equal(t, "#ff0000", label.Color, "colours are normalised to lower case")

	expectError(t, f.owner.do(http.MethodPost, projectPath(f.project, "/labels"), api.LabelInput{Name: "bug", Color: "#00ff00"}), http.StatusConflict, "CONFLICT")
	expectError(t, f.owner.do(http.MethodPost, projectPath(f.project, "/labels"), api.LabelInput{Name: "x", Color: "red"}), http.StatusBadRequest, "VALIDATION_ERROR")
	expectError(t, dev.do(http.MethodPost, projectPath(f.project, "/labels"), api.LabelInput{Name: "feature", Color: "#00ff00"}), http.StatusForbidden, "FORBIDDEN")

	labels := decode[[]api.Label](t, dev.do(http.MethodGet, projectPath(f.project, "/labels"), nil), http.StatusOK)
	require.Len(t, labels, 1)

	other := decode[api.Label](t, f.owner.do(http.MethodPost, projectPath(f.project, "/labels"), api.LabelInput{Name: "feature", Color: "#00ff00"}), http.StatusCreated)
	renamed := decode[api.Label](t, f.owner.do(http.MethodPatch, "/labels/"+other.Id.String(), api.LabelPatch{Name: ptr("enhancement"), Color: ptr("#0000FF")}), http.StatusOK)
	assert.Equal(t, "enhancement", renamed.Name)
	assert.Equal(t, "#0000ff", renamed.Color)
	expectError(t, f.owner.do(http.MethodPatch, "/labels/"+other.Id.String(), api.LabelPatch{Name: ptr("bug")}), http.StatusConflict, "CONFLICT")
	expectError(t, dev.do(http.MethodPatch, "/labels/"+other.Id.String(), api.LabelPatch{Name: ptr("x")}), http.StatusForbidden, "FORBIDDEN")
	expectError(t, f.owner.do(http.MethodPatch, "/labels/"+uuid.NewString(), api.LabelPatch{Name: ptr("x")}), http.StatusNotFound, "NOT_FOUND")

	// Deleting a label removes it from cards.
	card := decode[api.Card](t, dev.do(http.MethodPost, "/columns/"+f.todo.Id.String()+"/cards", map[string]any{"title": "t", "labelIds": []uuid.UUID{label.Id}}), http.StatusCreated)
	require.Equal(t, []uuid.UUID{label.Id}, card.LabelIds)
	require.Equal(t, http.StatusNoContent, f.owner.do(http.MethodDelete, "/labels/"+label.Id.String(), nil).status)
	card = decode[api.Card](t, dev.do(http.MethodGet, "/cards/"+card.Id.String(), nil), http.StatusOK)
	assert.Empty(t, card.LabelIds)
	assert.False(t, slices.Contains(titles(nil), "x"))
	expectError(t, f.owner.do(http.MethodDelete, "/labels/"+label.Id.String(), nil), http.StatusNotFound, "NOT_FOUND")
}

func ptr[T any](v T) *T { return &v }
