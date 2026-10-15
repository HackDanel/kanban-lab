package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/HackDanel/kanban-lab/backend/internal/domain"
	"github.com/HackDanel/kanban-lab/backend/internal/repository"
)

// defaultColumns are created on every new board.
var defaultColumns = []string{"To Do", "In Progress", "Done"}

// BoardDetail is a board with its ordered columns.
type BoardDetail struct {
	Board   repository.Board
	Columns []repository.ListColumnsRow
}

// ColumnPatch holds the column fields to change. WipLimit is applied when
// SetWipLimit is true; a nil WipLimit then removes the limit.
type ColumnPatch struct {
	Name        *string
	SetWipLimit bool
	WipLimit    *int
}

func validateWipLimit(v *validator, limit *int) *int32 {
	if limit == nil {
		return nil
	}
	if *limit < 1 || *limit > 1000 {
		v.add("wipLimit", "WIP-лимит — целое число от 1 до 1000")
		return nil
	}
	n := int32(*limit)
	return &n
}

// ListBoards returns the boards of a project.
func (s *Service) ListBoards(ctx context.Context, actor domain.Actor, projectID uuid.UUID) ([]repository.Board, error) {
	if _, err := loadProject(ctx, s.store.Queries, actor, projectID); err != nil {
		return nil, err
	}
	return s.store.ListBoards(ctx, projectID)
}

// CreateBoard creates a board with the default columns.
func (s *Service) CreateBoard(ctx context.Context, actor domain.Actor, projectID uuid.UUID, name string) (BoardDetail, error) {
	if _, err := manageProject(ctx, s.store.Queries, actor, projectID); err != nil {
		return BoardDetail{}, err
	}
	var v validator
	name = v.text("name", name, true, 100, "Название обязательно")
	if err := v.err(); err != nil {
		return BoardDetail{}, err
	}

	var detail BoardDetail
	err := s.store.InTx(ctx, func(q *repository.Queries) error {
		board, err := q.CreateBoard(ctx, repository.CreateBoardParams{ProjectID: projectID, Name: name})
		if err != nil {
			return fmt.Errorf("create board: %w", err)
		}
		for i, column := range defaultColumns {
			_, err := q.CreateColumn(ctx, repository.CreateColumnParams{
				BoardID: board.ID, Name: column, Position: float64(i+1) * positionStep,
			})
			if err != nil {
				return fmt.Errorf("create column: %w", err)
			}
		}
		columns, err := q.ListColumns(ctx, board.ID)
		detail = BoardDetail{Board: board, Columns: columns}
		return err
	})
	return detail, err
}

// GetBoard returns a board with its columns.
func (s *Service) GetBoard(ctx context.Context, actor domain.Actor, id uuid.UUID) (BoardDetail, error) {
	board, err := loadBoard(ctx, s.store.Queries, actor, id, false)
	if err != nil {
		return BoardDetail{}, err
	}
	columns, err := s.store.ListColumns(ctx, id)
	if err != nil {
		return BoardDetail{}, fmt.Errorf("list columns: %w", err)
	}
	return BoardDetail{Board: board, Columns: columns}, nil
}

// RenameBoard changes the name of a board.
func (s *Service) RenameBoard(ctx context.Context, actor domain.Actor, id uuid.UUID, name string) (repository.Board, error) {
	if _, err := loadBoard(ctx, s.store.Queries, actor, id, true); err != nil {
		return repository.Board{}, err
	}
	var v validator
	name = v.text("name", name, true, 100, "Название обязательно")
	if err := v.err(); err != nil {
		return repository.Board{}, err
	}
	board, err := s.store.RenameBoard(ctx, repository.RenameBoardParams{ID: id, Name: name})
	if err != nil {
		return board, fmt.Errorf("rename board: %w", err)
	}
	s.publish(ctx, EventBoardUpdated, id, nil)
	return board, nil
}

// DeleteBoard removes a board with its columns, cards and attachments.
func (s *Service) DeleteBoard(ctx context.Context, actor domain.Actor, id uuid.UUID) error {
	var keys []string
	err := s.store.InTx(ctx, func(q *repository.Queries) error {
		if _, err := loadBoard(ctx, q, actor, id, true); err != nil {
			return err
		}
		var err error
		if keys, err = q.ListAttachmentKeysForBoard(ctx, id); err != nil {
			return fmt.Errorf("list attachments: %w", err)
		}
		_, err = q.DeleteBoard(ctx, id)
		return err
	})
	if err != nil {
		return err
	}
	s.removeFiles(keys)
	s.publish(ctx, EventBoardDeleted, id, nil)
	return nil
}

// CreateColumn appends a column to a board.
func (s *Service) CreateColumn(ctx context.Context, actor domain.Actor, boardID uuid.UUID, name string, wipLimit *int) (repository.ListColumnsRow, error) {
	var v validator
	name = v.text("name", name, true, 60, "Название обязательно")
	limit := validateWipLimit(&v, wipLimit)

	var column repository.BoardColumn
	err := s.store.InTx(ctx, func(q *repository.Queries) error {
		if _, err := loadBoard(ctx, q, actor, boardID, true); err != nil {
			return err
		}
		if err := v.err(); err != nil {
			return err
		}
		if _, err := q.LockBoard(ctx, boardID); err != nil {
			return fmt.Errorf("lock board: %w", err)
		}
		last, err := q.MaxColumnPosition(ctx, boardID)
		if err != nil {
			return fmt.Errorf("max column position: %w", err)
		}
		column, err = q.CreateColumn(ctx, repository.CreateColumnParams{
			BoardID: boardID, Name: name, Position: last + positionStep, WipLimit: limit,
		})
		return err
	})
	if err != nil {
		return repository.ListColumnsRow{}, err
	}
	s.publish(ctx, EventColumnsChanged, boardID, nil)
	return columnRow(column, 0), nil
}

func columnRow(c repository.BoardColumn, cardCount int32) repository.ListColumnsRow {
	return repository.ListColumnsRow{
		ID: c.ID, BoardID: c.BoardID, Name: c.Name, Position: c.Position,
		WipLimit: c.WipLimit, CreatedAt: c.CreatedAt, CardCount: cardCount,
	}
}

func (s *Service) columnWithCount(ctx context.Context, q *repository.Queries, id uuid.UUID) (repository.ListColumnsRow, error) {
	column, err := q.GetColumn(ctx, id)
	if err != nil {
		return repository.ListColumnsRow{}, fmt.Errorf("get column: %w", err)
	}
	count, err := q.CountCardsInColumn(ctx, id)
	if err != nil {
		return repository.ListColumnsRow{}, fmt.Errorf("count cards: %w", err)
	}
	return columnRow(column, int32(count)), nil
}

// UpdateColumn renames a column or changes its WIP limit. Lowering the limit
// below the current number of cards is allowed: it only blocks further cards.
func (s *Service) UpdateColumn(ctx context.Context, actor domain.Actor, id uuid.UUID, patch ColumnPatch) (repository.ListColumnsRow, error) {
	var v validator
	var name string
	if patch.Name != nil {
		name = v.text("name", *patch.Name, true, 60, "Название обязательно")
	}
	limit := validateWipLimit(&v, patch.WipLimit)

	var result repository.ListColumnsRow
	var boardID uuid.UUID
	err := s.store.InTx(ctx, func(q *repository.Queries) error {
		column, _, err := loadColumn(ctx, q, actor, id, true)
		if err != nil {
			return err
		}
		if err := v.err(); err != nil {
			return err
		}
		boardID = column.BoardID
		if patch.Name != nil {
			if err := q.RenameColumn(ctx, repository.RenameColumnParams{ID: id, Name: name}); err != nil {
				return fmt.Errorf("rename column: %w", err)
			}
		}
		if patch.SetWipLimit {
			if err := q.SetColumnWipLimit(ctx, repository.SetColumnWipLimitParams{ID: id, WipLimit: limit}); err != nil {
				return fmt.Errorf("set WIP limit: %w", err)
			}
		}
		result, err = s.columnWithCount(ctx, q, id)
		return err
	})
	if err != nil {
		return result, err
	}
	s.publish(ctx, EventColumnsChanged, boardID, nil)
	return result, nil
}

// DeleteColumn removes an empty column.
func (s *Service) DeleteColumn(ctx context.Context, actor domain.Actor, id uuid.UUID) error {
	var boardID uuid.UUID
	err := s.store.InTx(ctx, func(q *repository.Queries) error {
		column, _, err := loadColumn(ctx, q, actor, id, true)
		if err != nil {
			return err
		}
		boardID = column.BoardID
		// The lock keeps a card from being created or moved here meanwhile.
		if _, err := q.LockColumn(ctx, id); err != nil {
			return fmt.Errorf("lock column: %w", err)
		}
		count, err := q.CountCardsInColumn(ctx, id)
		if err != nil {
			return fmt.Errorf("count cards: %w", err)
		}
		if count > 0 {
			return domain.E(domain.KindColumnNotEmpty, "В колонке есть карточки. Перенесите или удалите их, чтобы удалить колонку")
		}
		_, err = q.DeleteColumn(ctx, id)
		return err
	})
	if err != nil {
		return err
	}
	s.publish(ctx, EventColumnsChanged, boardID, nil)
	return nil
}

// MoveColumn places a column after another one on the same board, or first when after is nil.
func (s *Service) MoveColumn(ctx context.Context, actor domain.Actor, id uuid.UUID, after *uuid.UUID) (repository.ListColumnsRow, error) {
	var result repository.ListColumnsRow
	var boardID uuid.UUID
	err := s.store.InTx(ctx, func(q *repository.Queries) error {
		column, _, err := loadColumn(ctx, q, actor, id, true)
		if err != nil {
			return err
		}
		boardID = column.BoardID
		if _, err := q.LockBoard(ctx, boardID); err != nil {
			return fmt.Errorf("lock board: %w", err)
		}
		pos, err := placeInList(ctx, after, "afterColumnId",
			func() ([]placed, error) {
				rows, err := q.ColumnPositions(ctx, repository.ColumnPositionsParams{BoardID: boardID, ExcludeID: id})
				out := make([]placed, len(rows))
				for i, r := range rows {
					out[i] = placed{r.ID, r.Position}
				}
				return out, err
			},
			func() error { return q.RenumberColumns(ctx, boardID) },
		)
		if err != nil {
			return err
		}
		if err := q.SetColumnPosition(ctx, repository.SetColumnPositionParams{ID: id, Position: pos}); err != nil {
			return fmt.Errorf("set column position: %w", err)
		}
		result, err = s.columnWithCount(ctx, q, id)
		return err
	})
	if err != nil {
		return result, err
	}
	s.publish(ctx, EventColumnsChanged, boardID, nil)
	return result, nil
}

// placeInList computes the position for an item inserted after another one,
// renumbering the list once when the gap between neighbours is exhausted.
func placeInList(_ context.Context, after *uuid.UUID, field string, siblings func() ([]placed, error), renumber func() error) (float64, error) {
	for attempt := 0; ; attempt++ {
		list, err := siblings()
		if err != nil {
			return 0, fmt.Errorf("load positions: %w", err)
		}
		pos, needRenumber, err := positionAfter(list, after, field)
		if err != nil || !needRenumber {
			return pos, err
		}
		if attempt > 0 {
			return 0, fmt.Errorf("positions still too close after renumbering")
		}
		if err := renumber(); err != nil {
			return 0, fmt.Errorf("renumber: %w", err)
		}
	}
}
