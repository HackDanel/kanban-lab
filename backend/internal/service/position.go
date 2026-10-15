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

// positionAfter returns the position for an item inserted into siblings — the
// ordered list without the item itself — right after the item with id after,
// or at the head when after is nil. renumber is true when neighbours are too
// close to fit a value between them; the caller must then renumber the list
// and call positionAfter again.
func positionAfter(siblings []placed, after *uuid.UUID, field string) (pos float64, renumber bool, err error) {
	if after == nil {
		if len(siblings) == 0 {
			return positionStep, false, nil
		}
		return siblings[0].Position - positionStep, false, nil
	}

	for i, s := range siblings {
		if s.ID != *after {
			continue
		}
		if i == len(siblings)-1 {
			return s.Position + positionStep, false, nil
		}
		next := siblings[i+1].Position
		if next-s.Position < minPositionGap {
			return 0, true, nil
		}
		return s.Position + (next-s.Position)/2, false, nil
	}
	return 0, false, domain.Invalid(field, "Элемент, после которого нужно вставить, не найден в целевом списке")
}
