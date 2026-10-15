package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/HackDanel/kanban-lab/backend/internal/domain"
	"github.com/HackDanel/kanban-lab/backend/internal/repository"
)

func requireActor(actor domain.Actor) error {
	if actor.ID == uuid.Nil {
		return domain.ErrUnauthorized
	}
	return nil
}

// loadProject returns the project if the actor may see it: administrators see
// every project, everybody else only the projects they are members of.
func loadProject(ctx context.Context, q *repository.Queries, actor domain.Actor, projectID uuid.UUID) (repository.Project, error) {
	if err := requireActor(actor); err != nil {
		return repository.Project{}, err
	}
	project, err := q.GetProject(ctx, projectID)
	if repository.IsNotFound(err) {
		return project, domain.NotFound("Проект")
	}
	if err != nil {
		return project, fmt.Errorf("get project: %w", err)
	}
	if actor.IsAdmin() {
		return project, nil
	}
	member, err := q.IsProjectMember(ctx, repository.IsProjectMemberParams{ProjectID: projectID, UserID: actor.ID})
	if err != nil {
		return project, fmt.Errorf("check membership: %w", err)
	}
	if !member {
		return project, domain.ErrForbidden
	}
	return project, nil
}

// manageProject is loadProject plus the requirement that the actor's role
// allows managing boards, columns and labels.
func manageProject(ctx context.Context, q *repository.Queries, actor domain.Actor, projectID uuid.UUID) (repository.Project, error) {
	project, err := loadProject(ctx, q, actor, projectID)
	if err != nil {
		return project, err
	}
	if !actor.CanManage() {
		return project, domain.ErrForbidden
	}
	return project, nil
}

// ownProject allows changing the project itself and its membership: to an
// administrator or to the manager who owns the project.
func ownProject(ctx context.Context, q *repository.Queries, actor domain.Actor, projectID uuid.UUID) (repository.Project, error) {
	project, err := loadProject(ctx, q, actor, projectID)
	if err != nil {
		return project, err
	}
	if !canOwn(actor, project) {
		return project, domain.ErrForbidden
	}
	return project, nil
}

func canOwn(actor domain.Actor, project repository.Project) bool {
	return actor.IsAdmin() || (actor.Role == domain.RoleManager && project.OwnerID == actor.ID)
}

func loadBoard(ctx context.Context, q *repository.Queries, actor domain.Actor, boardID uuid.UUID, manage bool) (repository.Board, error) {
	if err := requireActor(actor); err != nil {
		return repository.Board{}, err
	}
	board, err := q.GetBoard(ctx, boardID)
	if repository.IsNotFound(err) {
		return board, domain.NotFound("Доска")
	}
	if err != nil {
		return board, fmt.Errorf("get board: %w", err)
	}
	if manage {
		_, err = manageProject(ctx, q, actor, board.ProjectID)
	} else {
		_, err = loadProject(ctx, q, actor, board.ProjectID)
	}
	return board, err
}

func loadColumn(ctx context.Context, q *repository.Queries, actor domain.Actor, columnID uuid.UUID, manage bool) (repository.BoardColumn, repository.Board, error) {
	if err := requireActor(actor); err != nil {
		return repository.BoardColumn{}, repository.Board{}, err
	}
	column, err := q.GetColumn(ctx, columnID)
	if repository.IsNotFound(err) {
		return column, repository.Board{}, domain.NotFound("Колонка")
	}
	if err != nil {
		return column, repository.Board{}, fmt.Errorf("get column: %w", err)
	}
	board, err := loadBoard(ctx, q, actor, column.BoardID, manage)
	return column, board, err
}

func loadCard(ctx context.Context, q *repository.Queries, actor domain.Actor, cardID uuid.UUID) (repository.CardView, error) {
	if err := requireActor(actor); err != nil {
		return repository.CardView{}, err
	}
	card, err := q.GetCard(ctx, cardID)
	if repository.IsNotFound(err) {
		return card, domain.NotFound("Карточка")
	}
	if err != nil {
		return card, fmt.Errorf("get card: %w", err)
	}
	_, err = loadProject(ctx, q, actor, card.ProjectID)
	return card, err
}
