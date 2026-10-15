package service

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/HackDanel/kanban-lab/backend/internal/domain"
	"github.com/HackDanel/kanban-lab/backend/internal/repository"
)

// Activity actions.
const (
	ActionCreated           = "created"
	ActionUpdated           = "updated"
	ActionMoved             = "moved"
	ActionAssigned          = "assigned"
	ActionLabelsChanged     = "labels_changed"
	ActionCommented         = "commented"
	ActionAttachmentAdded   = "attachment_added"
	ActionAttachmentRemoved = "attachment_removed"
)

var priorities = []string{"low", "medium", "high", "critical"}

const dateLayout = "2006-01-02"

// CardInput holds the fields of a new card.
type CardInput struct {
	Title       string
	Description string
	Priority    string
	AssigneeID  *uuid.UUID
	DueDate     *time.Time
	LabelIDs    []uuid.UUID
}

// CardPatch holds the card fields to change. Fields with a Set flag are
// applied when the flag is true, so that they can also be cleared.
type CardPatch struct {
	Title       *string
	Description *string
	Priority    *string
	SetAssignee bool
	AssigneeID  *uuid.UUID
	SetDueDate  bool
	DueDate     *time.Time
	LabelIDs    *[]uuid.UUID
}

// CardFilter narrows the card list of a board.
type CardFilter struct {
	Query      string
	ColumnID   *uuid.UUID
	AssigneeID *uuid.UUID
	LabelIDs   []uuid.UUID
	Priority   *string
	Page       Page
}

type cardEventPayload struct {
	CardID   uuid.UUID `json:"cardId"`
	ColumnID uuid.UUID `json:"columnId"`
}

func validatePriority(v *validator, priority string) {
	if !slices.Contains(priorities, priority) {
		v.add("priority", "Приоритет: low, medium, high или critical")
	}
}

func validateTitle(v *validator, title string) string {
	return v.text("title", title, true, 200, "Название обязательно")
}

func validateDescription(v *validator, description string) string {
	if utf8.RuneCountInString(description) > 10000 {
		v.add("description", "Слишком длинное описание")
	}
	return description
}

func dedupe(ids []uuid.UUID) []uuid.UUID {
	out := slices.Clone(ids)
	slices.SortFunc(out, func(a, b uuid.UUID) int { return strings.Compare(a.String(), b.String()) })
	return slices.Compact(out)
}

// checkCardRefs verifies that the assignee is a project member and the labels belong to the project.
func checkCardRefs(ctx context.Context, q *repository.Queries, projectID uuid.UUID, assignee *uuid.UUID, labels []uuid.UUID) error {
	var v validator
	if assignee != nil {
		member, err := q.IsProjectMember(ctx, repository.IsProjectMemberParams{ProjectID: projectID, UserID: *assignee})
		if err != nil {
			return fmt.Errorf("check assignee: %w", err)
		}
		if !member {
			v.add("assigneeId", "Исполнитель должен быть участником проекта")
		}
	}
	if len(labels) > 20 {
		v.add("labelIds", "Не больше 20 меток")
	} else if len(labels) > 0 {
		n, err := q.CountLabelsInProject(ctx, repository.CountLabelsInProjectParams{ProjectID: projectID, Ids: labels})
		if err != nil {
			return fmt.Errorf("check labels: %w", err)
		}
		if int(n) != len(labels) {
			v.add("labelIds", "Метки должны принадлежать проекту")
		}
	}
	return v.err()
}

// checkWipLimit fails when the locked column has no room for one more card.
func checkWipLimit(ctx context.Context, q *repository.Queries, column repository.BoardColumn) error {
	if column.WipLimit == nil {
		return nil
	}
	count, err := q.CountCardsInColumn(ctx, column.ID)
	if err != nil {
		return fmt.Errorf("count cards: %w", err)
	}
	if count >= int64(*column.WipLimit) {
		return domain.E(domain.KindWipLimit,
			fmt.Sprintf("Превышен WIP-лимит колонки «%s»: не больше %d карточек", column.Name, *column.WipLimit))
	}
	return nil
}

func nullUUID(id *uuid.UUID) uuid.NullUUID {
	if id == nil {
		return uuid.NullUUID{}
	}
	return uuid.NullUUID{UUID: *id, Valid: true}
}

func activity(cardID, actorID uuid.UUID, action string, field string, oldValue, newValue *string) repository.InsertActivityParams {
	p := repository.InsertActivityParams{CardID: cardID, ActorID: actorID, Action: action, OldValue: oldValue, NewValue: newValue}
	if field != "" {
		p.Field = &field
	}
	return p
}

func strPtr(s string) *string { return &s }

func uuidText(id uuid.NullUUID) *string {
	if !id.Valid {
		return nil
	}
	return strPtr(id.UUID.String())
}

func dateText(d *time.Time) *string {
	if d == nil {
		return nil
	}
	return strPtr(d.Format(dateLayout))
}

func idsText(ids []uuid.UUID) *string {
	parts := make([]string, len(ids))
	for i, id := range ids {
		parts[i] = id.String()
	}
	return strPtr(strings.Join(parts, ","))
}

// ListCards returns a page of the cards of a board that match the filter.
func (s *Service) ListCards(ctx context.Context, actor domain.Actor, boardID uuid.UUID, f CardFilter) ([]repository.CardView, int64, error) {
	if _, err := loadBoard(ctx, s.store.Queries, actor, boardID, false); err != nil {
		return nil, 0, err
	}
	if f.Priority != nil {
		var v validator
		validatePriority(&v, *f.Priority)
		if err := v.err(); err != nil {
			return nil, 0, err
		}
	}
	limit, offset := f.Page.normalized()
	labels := dedupe(f.LabelIDs)
	pattern := likePattern(f.Query)

	cards, err := s.store.ListCards(ctx, repository.ListCardsParams{
		BoardID: boardID, ColumnID: nullUUID(f.ColumnID), AssigneeID: nullUUID(f.AssigneeID),
		Priority: f.Priority, Pattern: pattern, LabelIds: labels, PageLimit: limit, PageOffset: offset,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list cards: %w", err)
	}
	total, err := s.store.CountCards(ctx, repository.CountCardsParams{
		BoardID: boardID, ColumnID: nullUUID(f.ColumnID), AssigneeID: nullUUID(f.AssigneeID),
		Priority: f.Priority, Pattern: pattern, LabelIds: labels,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("count cards: %w", err)
	}
	return cards, total, nil
}

// CreateCard appends a card to a column, respecting its WIP limit.
func (s *Service) CreateCard(ctx context.Context, actor domain.Actor, columnID uuid.UUID, in CardInput) (repository.CardView, error) {
	var v validator
	title := validateTitle(&v, in.Title)
	description := validateDescription(&v, in.Description)
	if in.Priority == "" {
		in.Priority = "medium"
	}
	validatePriority(&v, in.Priority)
	labels := dedupe(in.LabelIDs)

	var card repository.CardView
	err := s.store.InTx(ctx, func(q *repository.Queries) error {
		_, board, err := loadColumn(ctx, q, actor, columnID, false)
		if err != nil {
			return err
		}
		if err := v.err(); err != nil {
			return err
		}
		if err := checkCardRefs(ctx, q, board.ProjectID, in.AssigneeID, labels); err != nil {
			return err
		}
		column, err := q.LockColumn(ctx, columnID)
		if err != nil {
			return fmt.Errorf("lock column: %w", err)
		}
		if err := checkWipLimit(ctx, q, column); err != nil {
			return err
		}
		number, err := q.NextCardNumber(ctx, board.ProjectID)
		if err != nil {
			return fmt.Errorf("next card number: %w", err)
		}
		last, err := q.MaxCardPosition(ctx, columnID)
		if err != nil {
			return fmt.Errorf("max card position: %w", err)
		}
		id, err := q.CreateCard(ctx, repository.CreateCardParams{
			ProjectID: board.ProjectID, BoardID: board.ID, ColumnID: columnID, Number: number,
			Title: title, Description: description, Priority: in.Priority, Position: last + positionStep,
			AssigneeID: nullUUID(in.AssigneeID), ReporterID: actor.ID, DueDate: in.DueDate,
		})
		if err != nil {
			return fmt.Errorf("create card: %w", err)
		}
		if len(labels) > 0 {
			if err := q.AddCardLabels(ctx, repository.AddCardLabelsParams{CardID: id, LabelIds: labels}); err != nil {
				return fmt.Errorf("add labels: %w", err)
			}
		}
		if err := q.InsertActivity(ctx, activity(id, actor.ID, ActionCreated, "", nil, strPtr(title))); err != nil {
			return fmt.Errorf("log activity: %w", err)
		}
		card, err = q.GetCard(ctx, id)
		return err
	})
	if err != nil {
		return card, err
	}
	s.publish(ctx, EventCardCreated, card.BoardID, cardEventPayload{card.ID, card.ColumnID})
	return card, nil
}

// GetCard returns a card.
func (s *Service) GetCard(ctx context.Context, actor domain.Actor, id uuid.UUID) (repository.CardView, error) {
	return loadCard(ctx, s.store.Queries, actor, id)
}

// UpdateCard changes card fields and records each change in the activity log.
func (s *Service) UpdateCard(ctx context.Context, actor domain.Actor, id uuid.UUID, patch CardPatch) (repository.CardView, error) {
	var v validator
	if patch.Title != nil {
		*patch.Title = validateTitle(&v, *patch.Title)
	}
	if patch.Description != nil {
		validateDescription(&v, *patch.Description)
	}
	if patch.Priority != nil {
		validatePriority(&v, *patch.Priority)
	}
	var labels []uuid.UUID
	if patch.LabelIDs != nil {
		labels = dedupe(*patch.LabelIDs)
	}

	var card repository.CardView
	err := s.store.InTx(ctx, func(q *repository.Queries) error {
		if err := requireActor(actor); err != nil {
			return err
		}
		if _, err := q.LockCard(ctx, id); err != nil {
			if repository.IsNotFound(err) {
				return domain.NotFound("Карточка")
			}
			return fmt.Errorf("lock card: %w", err)
		}
		current, err := loadCard(ctx, q, actor, id)
		if err != nil {
			return err
		}
		if err := v.err(); err != nil {
			return err
		}
		var assignee *uuid.UUID
		if patch.SetAssignee {
			assignee = patch.AssigneeID
		}
		if err := checkCardRefs(ctx, q, current.ProjectID, assignee, labels); err != nil {
			return err
		}

		next := repository.UpdateCardFieldsParams{
			ID: id, Title: current.Title, Description: current.Description, Priority: current.Priority,
			AssigneeID: current.AssigneeID, DueDate: current.DueDate,
		}
		var log []repository.InsertActivityParams
		if patch.Title != nil && *patch.Title != current.Title {
			log = append(log, activity(id, actor.ID, ActionUpdated, "title", strPtr(current.Title), patch.Title))
			next.Title = *patch.Title
		}
		if patch.Description != nil && *patch.Description != current.Description {
			log = append(log, activity(id, actor.ID, ActionUpdated, "description", nil, nil))
			next.Description = *patch.Description
		}
		if patch.Priority != nil && *patch.Priority != current.Priority {
			log = append(log, activity(id, actor.ID, ActionUpdated, "priority", strPtr(current.Priority), patch.Priority))
			next.Priority = *patch.Priority
		}
		if patch.SetAssignee && nullUUID(patch.AssigneeID) != current.AssigneeID {
			next.AssigneeID = nullUUID(patch.AssigneeID)
			log = append(log, activity(id, actor.ID, ActionAssigned, "assigneeId", uuidText(current.AssigneeID), uuidText(next.AssigneeID)))
		}
		if patch.SetDueDate && !sameDate(patch.DueDate, current.DueDate) {
			log = append(log, activity(id, actor.ID, ActionUpdated, "dueDate", dateText(current.DueDate), dateText(patch.DueDate)))
			next.DueDate = patch.DueDate
		}
		if patch.LabelIDs != nil && !slices.Equal(labels, dedupe(current.LabelIds)) {
			if err := q.ClearCardLabels(ctx, id); err != nil {
				return fmt.Errorf("clear labels: %w", err)
			}
			if len(labels) > 0 {
				if err := q.AddCardLabels(ctx, repository.AddCardLabelsParams{CardID: id, LabelIds: labels}); err != nil {
					return fmt.Errorf("add labels: %w", err)
				}
			}
			log = append(log, activity(id, actor.ID, ActionLabelsChanged, "labelIds", idsText(dedupe(current.LabelIds)), idsText(labels)))
		}

		if len(log) == 0 {
			card = current
			return nil
		}
		if err := q.UpdateCardFields(ctx, next); err != nil {
			return fmt.Errorf("update card: %w", err)
		}
		for _, entry := range log {
			if err := q.InsertActivity(ctx, entry); err != nil {
				return fmt.Errorf("log activity: %w", err)
			}
		}
		card, err = q.GetCard(ctx, id)
		return err
	})
	if err != nil {
		return card, err
	}
	s.publish(ctx, EventCardUpdated, card.BoardID, cardEventPayload{card.ID, card.ColumnID})
	return card, nil
}

func sameDate(a, b *time.Time) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return a.Format(dateLayout) == b.Format(dateLayout)
}

// MoveCard places a card into a column of the same board after another card,
// or first when after is nil. Entering a different column respects its WIP limit;
// reordering within a column does not change the number of cards and is always allowed.
func (s *Service) MoveCard(ctx context.Context, actor domain.Actor, id, columnID uuid.UUID, after *uuid.UUID) (repository.CardView, error) {
	var card repository.CardView
	err := s.store.InTx(ctx, func(q *repository.Queries) error {
		if err := requireActor(actor); err != nil {
			return err
		}
		if _, err := q.LockCard(ctx, id); err != nil {
			if repository.IsNotFound(err) {
				return domain.NotFound("Карточка")
			}
			return fmt.Errorf("lock card: %w", err)
		}
		current, err := loadCard(ctx, q, actor, id)
		if err != nil {
			return err
		}
		target, err := q.LockColumn(ctx, columnID)
		if repository.IsNotFound(err) || (err == nil && target.BoardID != current.BoardID) {
			return domain.Invalid("columnId", "Колонка не найдена на доске карточки")
		}
		if err != nil {
			return fmt.Errorf("lock column: %w", err)
		}
		changesColumn := target.ID != current.ColumnID
		if changesColumn {
			if err := checkWipLimit(ctx, q, target); err != nil {
				return err
			}
		}
		pos, err := placeInList(ctx, after, "afterCardId",
			func() ([]placed, error) {
				rows, err := q.CardPositions(ctx, repository.CardPositionsParams{ColumnID: columnID, ExcludeID: id})
				out := make([]placed, len(rows))
				for i, r := range rows {
					out[i] = placed{r.ID, r.Position}
				}
				return out, err
			},
			func() error { return q.RenumberCards(ctx, columnID) },
		)
		if err != nil {
			return err
		}
		if err := q.SetCardPlace(ctx, repository.SetCardPlaceParams{ID: id, ColumnID: columnID, Position: pos}); err != nil {
			return fmt.Errorf("move card: %w", err)
		}
		if changesColumn {
			from, err := q.GetColumn(ctx, current.ColumnID)
			if err != nil {
				return fmt.Errorf("get column: %w", err)
			}
			entry := activity(id, actor.ID, ActionMoved, "columnId", strPtr(from.Name), strPtr(target.Name))
			if err := q.InsertActivity(ctx, entry); err != nil {
				return fmt.Errorf("log activity: %w", err)
			}
		}
		card, err = q.GetCard(ctx, id)
		return err
	})
	if err != nil {
		return card, err
	}
	s.publish(ctx, EventCardMoved, card.BoardID, cardEventPayload{card.ID, card.ColumnID})
	return card, nil
}

// DeleteCard removes a card with its comments and attachments.
func (s *Service) DeleteCard(ctx context.Context, actor domain.Actor, id uuid.UUID) error {
	var card repository.CardView
	var keys []string
	err := s.store.InTx(ctx, func(q *repository.Queries) error {
		var err error
		if card, err = loadCard(ctx, q, actor, id); err != nil {
			return err
		}
		if keys, err = q.ListAttachmentKeysForCard(ctx, id); err != nil {
			return fmt.Errorf("list attachments: %w", err)
		}
		_, err = q.DeleteCard(ctx, id)
		return err
	})
	if err != nil {
		return err
	}
	s.removeFiles(keys)
	s.publish(ctx, EventCardDeleted, card.BoardID, cardEventPayload{card.ID, card.ColumnID})
	return nil
}

// ListActivity returns the history of a card, newest first.
func (s *Service) ListActivity(ctx context.Context, actor domain.Actor, cardID uuid.UUID, page Page) ([]repository.CardActivity, error) {
	if _, err := loadCard(ctx, s.store.Queries, actor, cardID); err != nil {
		return nil, err
	}
	limit, offset := page.normalized()
	return s.store.ListActivity(ctx, repository.ListActivityParams{CardID: cardID, PageLimit: limit, PageOffset: offset})
}
