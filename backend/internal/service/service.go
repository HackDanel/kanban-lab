// Package service implements the business rules of the Kanban board: access
// control, validation, WIP limits, card ordering and the activity log.
package service

import (
	"context"
	"io"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/HackDanel/kanban-lab/backend/internal/auth"
	"github.com/HackDanel/kanban-lab/backend/internal/repository"
)

// Event is a board change broadcast to connected clients.
type Event struct {
	Type    string    `json:"type"`
	BoardID uuid.UUID `json:"boardId"`
	Payload any       `json:"payload,omitempty"`
}

// Event types.
const (
	EventBoardUpdated   = "board.updated"
	EventBoardDeleted   = "board.deleted"
	EventColumnsChanged = "columns.changed"
	EventCardCreated    = "card.created"
	EventCardUpdated    = "card.updated"
	EventCardMoved      = "card.moved"
	EventCardDeleted    = "card.deleted"
)

// Publisher delivers events to clients watching a board.
type Publisher interface {
	Publish(ctx context.Context, event Event)
}

// FileStore keeps attachment contents.
type FileStore interface {
	// Save writes r under key and fails with ErrFileTooLarge when more than max bytes arrive.
	Save(ctx context.Context, key string, r io.Reader, max int64) (int64, error)
	Open(key string) (io.ReadSeekCloser, error)
	Delete(key string) error
}

// Deps are the collaborators of Service.
type Deps struct {
	Store          *repository.Store
	Hasher         *auth.Hasher
	Tokens         *auth.Tokens
	RefreshTTL     time.Duration
	Now            func() time.Time
	Events         Publisher
	Files          FileStore
	UploadMaxBytes int64
	Log            *slog.Logger
}

// Service is the application layer used by the HTTP handlers.
type Service struct {
	store      *repository.Store
	hasher     *auth.Hasher
	tokens     *auth.Tokens
	refreshTTL time.Duration
	now        func() time.Time
	events     Publisher
	files      FileStore
	maxUpload  int64
	log        *slog.Logger
}

// New builds a Service.
func New(d Deps) *Service {
	return &Service{
		store:      d.Store,
		hasher:     d.Hasher,
		tokens:     d.Tokens,
		refreshTTL: d.RefreshTTL,
		now:        d.Now,
		events:     d.Events,
		files:      d.Files,
		maxUpload:  d.UploadMaxBytes,
		log:        d.Log,
	}
}

func (s *Service) publish(ctx context.Context, eventType string, boardID uuid.UUID, payload any) {
	s.events.Publish(context.WithoutCancel(ctx), Event{Type: eventType, BoardID: boardID, Payload: payload})
}

// removeFiles deletes stored attachment contents after their rows are gone.
// Failures only leave orphaned files behind, so they are logged, not returned.
func (s *Service) removeFiles(keys []string) {
	for _, key := range keys {
		if err := s.files.Delete(key); err != nil {
			s.log.Warn("delete attachment file", "key", key, "err", err)
		}
	}
}
