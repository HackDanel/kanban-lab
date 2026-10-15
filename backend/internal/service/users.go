package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/HackDanel/kanban-lab/backend/internal/domain"
	"github.com/HackDanel/kanban-lab/backend/internal/repository"
)

// Me returns the actor's own profile.
func (s *Service) Me(ctx context.Context, actor domain.Actor) (repository.User, error) {
	if err := requireActor(actor); err != nil {
		return repository.User{}, err
	}
	user, err := s.store.GetUser(ctx, actor.ID)
	if repository.IsNotFound(err) {
		return user, domain.ErrUnauthorized
	}
	return user, err
}

// ListUsers searches users by name or email.
func (s *Service) ListUsers(ctx context.Context, actor domain.Actor, query string, page Page) ([]repository.User, int64, error) {
	if err := requireActor(actor); err != nil {
		return nil, 0, err
	}
	pattern := likePattern(query)
	limit, offset := page.normalized()
	users, err := s.store.ListUsers(ctx, repository.ListUsersParams{Pattern: pattern, PageLimit: limit, PageOffset: offset})
	if err != nil {
		return nil, 0, fmt.Errorf("list users: %w", err)
	}
	total, err := s.store.CountUsersMatching(ctx, pattern)
	if err != nil {
		return nil, 0, fmt.Errorf("count users: %w", err)
	}
	return users, total, nil
}

// SetRole changes a user's role. Only administrators may do it, and not for themselves,
// so that the system cannot be left without an administrator by accident.
func (s *Service) SetRole(ctx context.Context, actor domain.Actor, userID uuid.UUID, role domain.Role) (repository.User, error) {
	if err := requireActor(actor); err != nil {
		return repository.User{}, err
	}
	if !actor.IsAdmin() {
		return repository.User{}, domain.ErrForbidden
	}
	if !role.Valid() {
		return repository.User{}, domain.Invalid("role", "Неизвестная роль")
	}
	if userID == actor.ID {
		return repository.User{}, domain.Invalid("role", "Нельзя изменить собственную роль")
	}
	user, err := s.store.SetUserRole(ctx, repository.SetUserRoleParams{ID: userID, Role: string(role)})
	if repository.IsNotFound(err) {
		return user, domain.NotFound("Пользователь")
	}
	return user, err
}
