package service

import (
	"github.com/google/uuid"

	"github.com/HackDanel/kanban-lab/backend/internal/domain"
)

// positionStep is the gap left between neighbours when items are appended or renumbered.
const positionStep = 1024.0

// minPositionGap is the smallest gap between neighbours that can still be
// halved reliably; below it the list has to be renumbered.
const minPositionGap = 1e-6

// placed is an item of an ordered list: a card in a column or a column on a board.
type placed struct {
	ID       uuid.UUID
	Position float64
}

// positionBetween returns a position strictly between two neighbours, either
// of which may be absent. renumber is true when they are too close to fit a
// value between them.
func positionBetween(prev, next *float64) (pos float64, renumber bool) {
	switch {
	case prev == nil && next == nil:
		return positionStep, false
	case prev == nil:
		return *next - positionStep, false
	case next == nil:
		return *prev + positionStep, false
	case *next-*prev < minPositionGap:
		return 0, true
	default:
		return *prev + (*next-*prev)/2, false
	}
}

// positionAfter returns the position for an item inserted into siblings — the
// ordered list without the item itself — right after the item with id after,
// or at the head when after is nil. renumber is true when neighbours are too
// close to fit a value between them; the caller must then renumber the list
// and call positionAfter again.
func positionAfter(siblings []placed, after *uuid.UUID, field string) (pos float64, renumber bool, err error) {
	if after == nil {
		var next *float64
		if len(siblings) > 0 {
			next = &siblings[0].Position
		}
		pos, renumber = positionBetween(nil, next)
		return pos, renumber, nil
	}
	for i, s := range siblings {
		if s.ID != *after {
			continue
		}
		var next *float64
		if i+1 < len(siblings) {
			next = &siblings[i+1].Position
		}
		pos, renumber = positionBetween(&s.Position, next)
		return pos, renumber, nil
	}
	return 0, false, errAnchorNotFound(field)
}

func errAnchorNotFound(field string) error {
	return domain.Invalid(field, "Элемент, после которого нужно вставить, не найден в целевом списке")
}
