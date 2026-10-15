package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/HackDanel/kanban-lab/backend/internal/domain"
	"github.com/HackDanel/kanban-lab/backend/internal/repository"
)

// LabelPatch holds the label fields to change; nil means "leave as is".
type LabelPatch struct {
	Name  *string
	Color *string
}

var errLabelExists = domain.E(domain.KindConflict, "Метка с таким названием уже есть в проекте")

func validateColor(v *validator, color string) string {
	if !colorRe.MatchString(color) {
		v.add("color", "Цвет в формате #RRGGBB")
	}
	return strings.ToLower(color)
}

// ListLabels returns the labels of a project.
func (s *Service) ListLabels(ctx context.Context, actor domain.Actor, projectID uuid.UUID) ([]repository.Label, error) {
	if _, err := loadProject(ctx, s.store.Queries, actor, projectID); err != nil {
		return nil, err
	}
	return s.store.ListLabels(ctx, projectID)
}

// CreateLabel adds a label to a project.
func (s *Service) CreateLabel(ctx context.Context, actor domain.Actor, projectID uuid.UUID, name, color string) (repository.Label, error) {
	if _, err := manageProject(ctx, s.store.Queries, actor, projectID); err != nil {
		return repository.Label{}, err
	}
	var v validator
	name = v.text("name", name, true, 40, "Название обязательно")
	color = validateColor(&v, color)
	if err := v.err(); err != nil {
		return repository.Label{}, err
	}
	label, err := s.store.CreateLabel(ctx, repository.CreateLabelParams{ProjectID: projectID, Name: name, Color: color})
	if repository.IsUniqueViolation(err) {
		return label, errLabelExists
	}
	return label, err
}

func (s *Service) manageLabel(ctx context.Context, actor domain.Actor, id uuid.UUID) error {
	if err := requireActor(actor); err != nil {
		return err
	}
	label, err := s.store.GetLabel(ctx, id)
	if repository.IsNotFound(err) {
		return domain.NotFound("Метка")
	}
	if err != nil {
		return fmt.Errorf("get label: %w", err)
	}
	_, err = manageProject(ctx, s.store.Queries, actor, label.ProjectID)
	return err
}

// UpdateLabel renames or recolours a label.
func (s *Service) UpdateLabel(ctx context.Context, actor domain.Actor, id uuid.UUID, patch LabelPatch) (repository.Label, error) {
	if err := s.manageLabel(ctx, actor, id); err != nil {
		return repository.Label{}, err
	}
	var v validator
	params := repository.UpdateLabelParams{ID: id}
	if patch.Name != nil {
		name := v.text("name", *patch.Name, true, 40, "Название обязательно")
		params.Name = &name
	}
	if patch.Color != nil {
		color := validateColor(&v, *patch.Color)
		params.Color = &color
	}
	if err := v.err(); err != nil {
		return repository.Label{}, err
	}
	label, err := s.store.UpdateLabel(ctx, params)
	if repository.IsUniqueViolation(err) {
		return label, errLabelExists
	}
	return label, err
}

// DeleteLabel removes a label from the project and from every card.
func (s *Service) DeleteLabel(ctx context.Context, actor domain.Actor, id uuid.UUID) error {
	if err := s.manageLabel(ctx, actor, id); err != nil {
		return err
	}
	_, err := s.store.DeleteLabel(ctx, id)
	return err
}
