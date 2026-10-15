package httpserver

import (
	"github.com/google/uuid"
	"github.com/oapi-codegen/nullable"
	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/HackDanel/kanban-lab/backend/internal/http/api"
	"github.com/HackDanel/kanban-lab/backend/internal/repository"
	"github.com/HackDanel/kanban-lab/backend/internal/service"
)

func nullableOf[T any](v *T) nullable.Nullable[T] {
	if v == nil {
		return nullable.NewNullNullable[T]()
	}
	return nullable.NewNullableWithValue(*v)
}

// optional reads a nullable request field: set reports whether it was sent
// at all, and value is nil when it was sent as null.
func optional[T any](n nullable.Nullable[T]) (value *T, set bool) {
	if !n.IsSpecified() {
		return nil, false
	}
	if n.IsNull() {
		return nil, true
	}
	v := n.MustGet()
	return &v, true
}

func mapSlice[T, R any](items []T, f func(T) R) []R {
	out := make([]R, len(items))
	for i, item := range items {
		out[i] = f(item)
	}
	return out
}

func toUser(u repository.User) api.User {
	return api.User{
		Id: u.ID, Email: openapi_types.Email(u.Email), Name: u.Name,
		Role: api.Role(u.Role), CreatedAt: u.CreatedAt,
	}
}

func toAuth(s service.Session) api.AuthResponse {
	return api.AuthResponse{AccessToken: s.AccessToken, ExpiresIn: int(s.ExpiresIn.Seconds()), User: toUser(s.User)}
}

func toProject(p repository.Project) api.Project {
	return api.Project{
		Id: p.ID, Key: p.Key, Name: p.Name, Description: p.Description,
		OwnerId: p.OwnerID, CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt,
	}
}

func toLabel(l repository.Label) api.Label {
	return api.Label{Id: l.ID, ProjectId: l.ProjectID, Name: l.Name, Color: l.Color}
}

func toBoard(b repository.Board) api.Board {
	return api.Board{Id: b.ID, ProjectId: b.ProjectID, Name: b.Name, CreatedAt: b.CreatedAt, UpdatedAt: b.UpdatedAt}
}

func toColumn(c repository.ListColumnsRow) api.Column {
	var limit *int
	if c.WipLimit != nil {
		n := int(*c.WipLimit)
		limit = &n
	}
	return api.Column{
		Id: c.ID, BoardId: c.BoardID, Name: c.Name, Position: c.Position,
		WipLimit: nullableOf(limit), CardCount: int(c.CardCount),
	}
}

func toBoardDetail(d service.BoardDetail) api.BoardDetail {
	return api.BoardDetail{
		Id: d.Board.ID, ProjectId: d.Board.ProjectID, Name: d.Board.Name,
		CreatedAt: d.Board.CreatedAt, UpdatedAt: d.Board.UpdatedAt,
		Columns: mapSlice(d.Columns, toColumn),
	}
}

func toCard(c repository.CardView) api.Card {
	var assignee *uuid.UUID
	if c.AssigneeID.Valid {
		assignee = &c.AssigneeID.UUID
	}
	var due *openapi_types.Date
	if c.DueDate != nil {
		due = &openapi_types.Date{Time: *c.DueDate}
	}
	return api.Card{
		Id: c.ID, BoardId: c.BoardID, ColumnId: c.ColumnID, Number: int(c.Number),
		Title: c.Title, Description: c.Description, Priority: api.Priority(c.Priority),
		Position: c.Position, AssigneeId: nullableOf(assignee), ReporterId: c.ReporterID,
		DueDate: nullableOf(due), LabelIds: c.LabelIds,
		CommentCount: int(c.CommentCount), AttachmentCount: int(c.AttachmentCount),
		CreatedAt: c.CreatedAt, UpdatedAt: c.UpdatedAt,
	}
}

func toComment(c repository.Comment) api.Comment {
	return api.Comment{Id: c.ID, CardId: c.CardID, AuthorId: c.AuthorID, Body: c.Body, CreatedAt: c.CreatedAt, UpdatedAt: c.UpdatedAt}
}

func toAttachment(a repository.Attachment) api.Attachment {
	return api.Attachment{
		Id: a.ID, CardId: a.CardID, UploaderId: a.UploaderID, Filename: a.Filename,
		ContentType: a.ContentType, Size: a.Size, CreatedAt: a.CreatedAt,
	}
}

func toActivity(a repository.CardActivity) api.Activity {
	return api.Activity{
		Id: a.ID, CardId: a.CardID, ActorId: a.ActorID, Action: api.ActivityAction(a.Action),
		Field: nullableOf(a.Field), OldValue: nullableOf(a.OldValue), NewValue: nullableOf(a.NewValue),
		CreatedAt: a.CreatedAt,
	}
}
