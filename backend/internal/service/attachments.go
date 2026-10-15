package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"path"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/HackDanel/kanban-lab/backend/internal/domain"
	"github.com/HackDanel/kanban-lab/backend/internal/repository"
)

// ErrFileTooLarge is returned by FileStore.Save when the upload exceeds the limit.
var ErrFileTooLarge = errors.New("file too large")

// allowedContentTypes lists the attachment types accepted for upload.
var allowedContentTypes = []string{
	"image/png", "image/jpeg", "image/gif", "image/webp",
	"application/pdf", "text/plain", "text/csv", "application/json", "application/zip",
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document",
	"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	"application/vnd.openxmlformats-officedocument.presentationml.presentation",
}

// Download is an attachment opened for reading.
type Download struct {
	Attachment repository.Attachment
	Content    io.ReadSeekCloser
}

// sanitizeFilename keeps the base name of an uploaded file, without
// directories or control characters, and at most 255 bytes long.
func sanitizeFilename(name string) string {
	name = path.Base(strings.ReplaceAll(name, `\`, "/"))
	name = strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f || r == '"' {
			return -1
		}
		return r
	}, name)
	name = strings.TrimSpace(name)
	for len(name) > 255 {
		_, size := utf8.DecodeLastRuneInString(name)
		name = name[:len(name)-size]
	}
	if name == "" || name == "." || name == "/" {
		return "file"
	}
	return name
}

func normalizeContentType(contentType string) (string, bool) {
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return "", false
	}
	return mediaType, slices.Contains(allowedContentTypes, mediaType)
}

// ListAttachments returns the attachments of a card.
func (s *Service) ListAttachments(ctx context.Context, actor domain.Actor, cardID uuid.UUID) ([]repository.Attachment, error) {
	if _, err := loadCard(ctx, s.store.Queries, actor, cardID); err != nil {
		return nil, err
	}
	return s.store.ListAttachments(ctx, cardID)
}

// UploadAttachment stores a file and attaches it to a card.
func (s *Service) UploadAttachment(ctx context.Context, actor domain.Actor, cardID uuid.UUID, filename, contentType string, content io.Reader) (repository.Attachment, error) {
	card, err := loadCard(ctx, s.store.Queries, actor, cardID)
	if err != nil {
		return repository.Attachment{}, err
	}
	mediaType, ok := normalizeContentType(contentType)
	if !ok {
		return repository.Attachment{}, domain.E(domain.KindUnsupportedMedia,
			"Недопустимый тип файла. Разрешены изображения, PDF, текст, CSV, JSON, ZIP и документы Office")
	}

	key := uuid.NewString()
	size, err := s.files.Save(ctx, key, content, s.maxUpload)
	if errors.Is(err, ErrFileTooLarge) {
		return repository.Attachment{}, domain.E(domain.KindFileTooLarge,
			fmt.Sprintf("Файл больше %d МБ", s.maxUpload>>20))
	}
	if err != nil {
		return repository.Attachment{}, fmt.Errorf("save file: %w", err)
	}

	var attachment repository.Attachment
	err = s.store.InTx(ctx, func(q *repository.Queries) error {
		var err error
		attachment, err = q.CreateAttachment(ctx, repository.CreateAttachmentParams{
			CardID: cardID, UploaderID: actor.ID, Filename: sanitizeFilename(filename),
			ContentType: mediaType, Size: size, StorageKey: key,
		})
		if err != nil {
			return fmt.Errorf("create attachment: %w", err)
		}
		return q.InsertActivity(ctx, activity(cardID, actor.ID, ActionAttachmentAdded, "", nil, strPtr(attachment.Filename)))
	})
	if err != nil {
		s.removeFiles([]string{key})
		return attachment, err
	}
	s.publish(ctx, EventCardUpdated, card.BoardID, cardEventPayload{card.ID, card.ColumnID})
	return attachment, nil
}

func (s *Service) loadAttachment(ctx context.Context, actor domain.Actor, id uuid.UUID) (repository.Attachment, repository.CardView, error) {
	if err := requireActor(actor); err != nil {
		return repository.Attachment{}, repository.CardView{}, err
	}
	attachment, err := s.store.GetAttachment(ctx, id)
	if repository.IsNotFound(err) {
		return attachment, repository.CardView{}, domain.NotFound("Вложение")
	}
	if err != nil {
		return attachment, repository.CardView{}, fmt.Errorf("get attachment: %w", err)
	}
	card, err := loadCard(ctx, s.store.Queries, actor, attachment.CardID)
	return attachment, card, err
}

// OpenAttachment returns the contents of an attachment. The caller closes Content.
func (s *Service) OpenAttachment(ctx context.Context, actor domain.Actor, id uuid.UUID) (Download, error) {
	attachment, _, err := s.loadAttachment(ctx, actor, id)
	if err != nil {
		return Download{}, err
	}
	content, err := s.files.Open(attachment.StorageKey)
	if err != nil {
		return Download{}, fmt.Errorf("open file: %w", err)
	}
	return Download{Attachment: attachment, Content: content}, nil
}

// DeleteAttachment removes an attachment: the uploader's own, or any as a manager or administrator.
func (s *Service) DeleteAttachment(ctx context.Context, actor domain.Actor, id uuid.UUID) error {
	attachment, card, err := s.loadAttachment(ctx, actor, id)
	if err != nil {
		return err
	}
	if attachment.UploaderID != actor.ID && !actor.CanManage() {
		return domain.ErrForbidden
	}
	err = s.store.InTx(ctx, func(q *repository.Queries) error {
		if _, err := q.DeleteAttachment(ctx, id); err != nil {
			return fmt.Errorf("delete attachment: %w", err)
		}
		return q.InsertActivity(ctx, activity(attachment.CardID, actor.ID, ActionAttachmentRemoved, "", strPtr(attachment.Filename), nil))
	})
	if err != nil {
		return err
	}
	s.removeFiles([]string{attachment.StorageKey})
	s.publish(ctx, EventCardUpdated, card.BoardID, cardEventPayload{card.ID, card.ColumnID})
	return nil
}
