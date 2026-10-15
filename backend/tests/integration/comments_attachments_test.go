package integration

import (
	"bytes"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/HackDanel/kanban-lab/backend/internal/domain"
	"github.com/HackDanel/kanban-lab/backend/internal/http/api"
)

// TC-21
func TestComments(t *testing.T) {
	f := newFixture(t)
	author := f.member(t, domain.RoleDeveloper)
	colleague := f.member(t, domain.RoleDeveloper)
	card := f.card(t, author, f.todo, "discuss")

	first := decode[api.Comment](t, author.do(http.MethodPost, cardPath(card, "/comments"), api.CommentInput{Body: "  first  "}), http.StatusCreated)
	assert.Equal(t, "first", first.Body)
	second := decode[api.Comment](t, colleague.do(http.MethodPost, cardPath(card, "/comments"), api.CommentInput{Body: "second"}), http.StatusCreated)

	list := decode[[]api.Comment](t, colleague.do(http.MethodGet, cardPath(card, "/comments"), nil), http.StatusOK)
	require.Len(t, list, 2)
	assert.Equal(t, []uuid.UUID{first.Id, second.Id}, []uuid.UUID{list[0].Id, list[1].Id}, "oldest first")
	assert.Equal(t, 2, decode[api.Card](t, author.do(http.MethodGet, cardPath(card, ""), nil), http.StatusOK).CommentCount)

	expectError(t, author.do(http.MethodPost, cardPath(card, "/comments"), api.CommentInput{Body: "   "}), http.StatusBadRequest, "VALIDATION_ERROR")
	expectError(t, author.do(http.MethodPost, cardPath(card, "/comments"), api.CommentInput{Body: strings.Repeat("x", 5001)}), http.StatusBadRequest, "VALIDATION_ERROR")

	edited := decode[api.Comment](t, author.do(http.MethodPatch, "/comments/"+first.Id.String(), api.CommentInput{Body: "edited"}), http.StatusOK)
	assert.Equal(t, "edited", edited.Body)
	expectError(t, colleague.do(http.MethodPatch, "/comments/"+first.Id.String(), api.CommentInput{Body: "hijack"}), http.StatusForbidden, "FORBIDDEN")
	expectError(t, f.owner.do(http.MethodPatch, "/comments/"+first.Id.String(), api.CommentInput{Body: "hijack"}), http.StatusForbidden, "FORBIDDEN")
	expectError(t, author.do(http.MethodPatch, "/comments/"+first.Id.String(), api.CommentInput{Body: ""}), http.StatusBadRequest, "VALIDATION_ERROR")

	// Developers delete only their own comments; managers delete any.
	expectError(t, colleague.do(http.MethodDelete, "/comments/"+first.Id.String(), nil), http.StatusForbidden, "FORBIDDEN")
	require.Equal(t, http.StatusNoContent, colleague.do(http.MethodDelete, "/comments/"+second.Id.String(), nil).status)
	require.Equal(t, http.StatusNoContent, f.owner.do(http.MethodDelete, "/comments/"+first.Id.String(), nil).status)
	expectError(t, author.do(http.MethodDelete, "/comments/"+first.Id.String(), nil), http.StatusNotFound, "NOT_FOUND")
	expectError(t, newUser(t, domain.RoleDeveloper).do(http.MethodGet, cardPath(card, "/comments"), nil), http.StatusForbidden, "FORBIDDEN")

	history := decode[[]api.Activity](t, author.do(http.MethodGet, cardPath(card, "/activity"), nil), http.StatusOK)
	assert.Equal(t, []string{"commented", "commented", "created"}, activityActions(history))
}

// TC-22, TC-23
func TestAttachments(t *testing.T) {
	f := newFixture(t)
	uploader := f.member(t, domain.RoleDeveloper)
	colleague := f.member(t, domain.RoleDeveloper)
	card := f.card(t, uploader, f.todo, "with files")
	content := bytes.Repeat([]byte("pdf-bytes "), 100)

	att := decode[api.Attachment](t, uploader.upload(card.Id, `C:\Users\me\отчёт "final".pdf`, "application/pdf", content), http.StatusCreated)
	assert.Equal(t, "отчёт final.pdf", att.Filename, "directories and quotes are stripped")
	assert.Equal(t, "application/pdf", att.ContentType)
	assert.Equal(t, int64(len(content)), att.Size)

	list := decode[[]api.Attachment](t, colleague.do(http.MethodGet, cardPath(card, "/attachments"), nil), http.StatusOK)
	require.Len(t, list, 1)
	assert.Equal(t, 1, decode[api.Card](t, uploader.do(http.MethodGet, cardPath(card, ""), nil), http.StatusOK).AttachmentCount)

	r := colleague.do(http.MethodGet, "/attachments/"+att.Id.String()+"/download", nil)
	require.Equal(t, http.StatusOK, r.status)
	assert.Equal(t, content, r.body, "the file is returned unchanged")
	assert.Equal(t, "application/octet-stream", r.header.Get("Content-Type"))
	assert.Contains(t, r.header.Get("Content-Disposition"), "attachment")
	assert.Equal(t, "nosniff", r.header.Get("X-Content-Type-Options"))

	expectError(t, uploader.upload(card.Id, "big.txt", "text/plain", make([]byte, uploadLimit+1)), http.StatusRequestEntityTooLarge, "FILE_TOO_LARGE")
	decode[api.Attachment](t, uploader.upload(card.Id, "exact.txt", "text/plain", make([]byte, uploadLimit)), http.StatusCreated)
	expectError(t, uploader.upload(card.Id, "page.html", "text/html", []byte("<script>")), http.StatusUnsupportedMediaType, "UNSUPPORTED_MEDIA_TYPE")
	expectError(t, uploader.upload(card.Id, "x", "not a type", []byte("x")), http.StatusUnsupportedMediaType, "UNSUPPORTED_MEDIA_TYPE")
	expectError(t, newUser(t, domain.RoleDeveloper).upload(card.Id, "a.txt", "text/plain", []byte("x")), http.StatusForbidden, "FORBIDDEN")
	expectError(t, uploader.upload(uuid.New(), "a.txt", "text/plain", []byte("x")), http.StatusNotFound, "NOT_FOUND")
	anon := newClient(t)
	expectError(t, anon.upload(card.Id, "a.txt", "text/plain", []byte("x")), http.StatusUnauthorized, "UNAUTHORIZED")

	noFile := uploader.send(http.MethodPost, cardPath(card, "/attachments"), strings.NewReader("--b--\r\n"), "multipart/form-data; boundary=b", nil)
	expectError(t, noFile, http.StatusBadRequest, "VALIDATION_ERROR")

	expectError(t, colleague.do(http.MethodDelete, "/attachments/"+att.Id.String(), nil), http.StatusForbidden, "FORBIDDEN")
	require.Equal(t, http.StatusNoContent, uploader.do(http.MethodDelete, "/attachments/"+att.Id.String(), nil).status)
	expectError(t, uploader.do(http.MethodGet, "/attachments/"+att.Id.String()+"/download", nil), http.StatusNotFound, "NOT_FOUND")
	expectError(t, uploader.do(http.MethodDelete, "/attachments/"+att.Id.String(), nil), http.StatusNotFound, "NOT_FOUND")

	history := decode[[]api.Activity](t, uploader.do(http.MethodGet, cardPath(card, "/activity"), nil), http.StatusOK)
	assert.Equal(t, []string{"attachment_removed", "attachment_added", "attachment_added", "created"}, activityActions(history))
}
