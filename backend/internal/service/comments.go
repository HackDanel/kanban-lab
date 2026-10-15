package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/HackDanel/kanban-lab/backend/internal/domain"
	"github.com/HackDanel/kanban-lab/backend/internal/repository"
)

func validateComment(body string) (string, error) {
	var v validator
	body = v.text("body", body, true, 5000, "Комментарий не может быть пустым")
	return body, v.err()
}

// ListComments returns the comments of a card, oldest first.
func (s *Service) ListComments(ctx context.Context, actor domain.Actor, cardID uuid.UUID) ([]repository.Comment, error) {
	if _, err := loadCard(ctx, s.store.Queries, actor, cardID); err != nil {
		return nil, err
	}
	return s.store.ListComments(ctx, cardID)
}

// CreateComment adds a comment to a card.
func (s *Service) CreateComment(ctx context.Context, actor domain.Actor, cardID uuid.UUID, body string) (repository.Comment, error) {
	var comment repository.Comment
	var card repository.CardView
	err := s.store.InTx(ctx, func(q *repository.Queries) error {
		var err error
		if card, err = loadCard(ctx, q, actor, cardID); err != nil {
			return err
		}
		if body, err = validateComment(body); err != nil {
			return err
		}
		comment, err = q.CreateComment(ctx, repository.CreateCommentParams{CardID: cardID, AuthorID: actor.ID, Body: body})
		if err != nil {
			return fmt.Errorf("create comment: %w", err)
		}
		return q.InsertActivity(ctx, activity(cardID, actor.ID, ActionCommented, "", nil, nil))
	})
	if err != nil {
		return comment, err
	}
	s.publish(ctx, EventCardUpdated, card.BoardID, cardEventPayload{card.ID, card.ColumnID})
	return comment, nil
}

func (s *Service) loadComment(ctx context.Context, actor domain.Actor, id uuid.UUID) (repository.Comment, repository.CardView, error) {
	if err := requireActor(actor); err != nil {
		return repository.Comment{}, repository.CardView{}, err
	}
	comment, err := s.store.GetComment(ctx, id)
	if repository.IsNotFound(err) {
		return comment, repository.CardView{}, domain.NotFound("Комментарий")
	}
	if err != nil {
		return comment, repository.CardView{}, fmt.Errorf("get comment: %w", err)
	}
	card, err := loadCard(ctx, s.store.Queries, actor, comment.CardID)
	return comment, card, err
}

// UpdateComment changes the text of the actor's own comment.
func (s *Service) UpdateComment(ctx context.Context, actor domain.Actor, id uuid.UUID, body string) (repository.Comment, error) {
	comment, _, err := s.loadComment(ctx, actor, id)
	if err != nil {
		return comment, err
	}
	if comment.AuthorID != actor.ID {
		return comment, domain.E(domain.KindForbidden, "Изменять можно только свои комментарии")
	}
	if body, err = validateComment(body); err != nil {
		return comment, err
	}
	return s.store.UpdateComment(ctx, repository.UpdateCommentParams{ID: id, Body: body})
}

// DeleteComment removes a comment: the author's own, or any as a manager or administrator.
func (s *Service) DeleteComment(ctx context.Context, actor domain.Actor, id uuid.UUID) error {
	comment, card, err := s.loadComment(ctx, actor, id)
	if err != nil {
		return err
	}
	if comment.AuthorID != actor.ID && !actor.CanManage() {
		return domain.ErrForbidden
	}
	if _, err := s.store.DeleteComment(ctx, id); err != nil {
		return fmt.Errorf("delete comment: %w", err)
	}
	s.publish(ctx, EventCardUpdated, card.BoardID, cardEventPayload{card.ID, card.ColumnID})
	return nil
}
