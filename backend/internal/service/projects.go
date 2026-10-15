package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/HackDanel/kanban-lab/backend/internal/domain"
	"github.com/HackDanel/kanban-lab/backend/internal/repository"
)

// ProjectInput holds the fields of a new project.
type ProjectInput struct {
	Key         string
	Name        string
	Description string
}

// ProjectPatch holds the fields to change; nil means "leave as is".
type ProjectPatch struct {
	Name        *string
	Description *string
}

// ListProjects returns all projects to administrators and the member projects to everyone else.
func (s *Service) ListProjects(ctx context.Context, actor domain.Actor) ([]repository.Project, error) {
	if err := requireActor(actor); err != nil {
		return nil, err
	}
	if actor.IsAdmin() {
		return s.store.ListAllProjects(ctx)
	}
	return s.store.ListProjectsForMember(ctx, actor.ID)
}

// CreateProject creates a project owned by the actor, who becomes its first member.
func (s *Service) CreateProject(ctx context.Context, actor domain.Actor, in ProjectInput) (repository.Project, error) {
	if err := requireActor(actor); err != nil {
		return repository.Project{}, err
	}
	if !actor.CanManage() {
		return repository.Project{}, domain.ErrForbidden
	}
	var v validator
	if !projectKeyRe.MatchString(in.Key) {
		v.add("key", "От 2 до 10 латинских заглавных букв и цифр, первая — буква")
	}
	name := v.text("name", in.Name, true, 100, "Название обязательно")
	description := v.text("description", in.Description, false, 2000, "")
	if err := v.err(); err != nil {
		return repository.Project{}, err
	}

	var project repository.Project
	err := s.store.InTx(ctx, func(q *repository.Queries) error {
		var err error
		project, err = q.CreateProject(ctx, repository.CreateProjectParams{
			Key: in.Key, Name: name, Description: description, OwnerID: actor.ID,
		})
		if repository.IsUniqueViolation(err) {
			return domain.E(domain.KindConflict, "Проект с таким ключом уже существует")
		}
		if err != nil {
			return fmt.Errorf("create project: %w", err)
		}
		return q.AddProjectMember(ctx, repository.AddProjectMemberParams{ProjectID: project.ID, UserID: actor.ID})
	})
	return project, err
}

// GetProject returns a project visible to the actor.
func (s *Service) GetProject(ctx context.Context, actor domain.Actor, id uuid.UUID) (repository.Project, error) {
	return loadProject(ctx, s.store.Queries, actor, id)
}

// UpdateProject changes the name or description of a project.
func (s *Service) UpdateProject(ctx context.Context, actor domain.Actor, id uuid.UUID, patch ProjectPatch) (repository.Project, error) {
	if _, err := ownProject(ctx, s.store.Queries, actor, id); err != nil {
		return repository.Project{}, err
	}
	var v validator
	params := repository.UpdateProjectParams{ID: id}
	if patch.Name != nil {
		name := v.text("name", *patch.Name, true, 100, "Название обязательно")
		params.Name = &name
	}
	if patch.Description != nil {
		description := v.text("description", *patch.Description, false, 2000, "")
		params.Description = &description
	}
	if err := v.err(); err != nil {
		return repository.Project{}, err
	}
	project, err := s.store.UpdateProject(ctx, params)
	if repository.IsNotFound(err) {
		return project, domain.NotFound("Проект")
	}
	return project, err
}

// DeleteProject removes a project with its boards, cards and attachments.
func (s *Service) DeleteProject(ctx context.Context, actor domain.Actor, id uuid.UUID) error {
	var keys []string
	err := s.store.InTx(ctx, func(q *repository.Queries) error {
		if _, err := ownProject(ctx, q, actor, id); err != nil {
			return err
		}
		var err error
		if keys, err = q.ListAttachmentKeysForProject(ctx, id); err != nil {
			return fmt.Errorf("list attachments: %w", err)
		}
		if _, err := q.DeleteProject(ctx, id); err != nil {
			return fmt.Errorf("delete project: %w", err)
		}
		return nil
	})
	if err != nil {
		return err
	}
	s.removeFiles(keys)
	return nil
}

// ListMembers returns the members of a project.
func (s *Service) ListMembers(ctx context.Context, actor domain.Actor, projectID uuid.UUID) ([]repository.User, error) {
	if _, err := loadProject(ctx, s.store.Queries, actor, projectID); err != nil {
		return nil, err
	}
	return s.store.ListProjectMembers(ctx, projectID)
}

// AddMember adds a user to a project.
func (s *Service) AddMember(ctx context.Context, actor domain.Actor, projectID, userID uuid.UUID) error {
	if _, err := ownProject(ctx, s.store.Queries, actor, projectID); err != nil {
		return err
	}
	if _, err := s.store.GetUser(ctx, userID); err != nil {
		if repository.IsNotFound(err) {
			return domain.NotFound("Пользователь")
		}
		return fmt.Errorf("get user: %w", err)
	}
	return s.store.AddProjectMember(ctx, repository.AddProjectMemberParams{ProjectID: projectID, UserID: userID})
}

// RemoveMember removes a user from a project and unassigns their cards in it.
// The owner cannot be removed.
func (s *Service) RemoveMember(ctx context.Context, actor domain.Actor, projectID, userID uuid.UUID) error {
	return s.store.InTx(ctx, func(q *repository.Queries) error {
		project, err := ownProject(ctx, q, actor, projectID)
		if err != nil {
			return err
		}
		if project.OwnerID == userID {
			return domain.Invalid("userId", "Нельзя исключить владельца проекта")
		}
		n, err := q.RemoveProjectMember(ctx, repository.RemoveProjectMemberParams{ProjectID: projectID, UserID: userID})
		if err != nil {
			return fmt.Errorf("remove member: %w", err)
		}
		if n == 0 {
			return domain.NotFound("Участник проекта")
		}
		return q.UnassignMemberCards(ctx, repository.UnassignMemberCardsParams{ProjectID: projectID, UserID: uuid.NullUUID{UUID: userID, Valid: true}})
	})
}
