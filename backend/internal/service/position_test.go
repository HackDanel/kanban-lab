package service

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/HackDanel/kanban-lab/backend/internal/domain"
)

func TestPositionAfter(t *testing.T) {
	a, b, c := uuid.New(), uuid.New(), uuid.New()
	list := []placed{{a, 1024}, {b, 2048}, {c, 3072}}

	t.Run("empty list", func(t *testing.T) {
		pos, renumber, err := positionAfter(nil, nil, "afterCardId")
		require.NoError(t, err)
		assert.False(t, renumber)
		assert.Equal(t, 1024.0, pos)
	})

	t.Run("head of the list", func(t *testing.T) {
		pos, renumber, err := positionAfter(list, nil, "afterCardId")
		require.NoError(t, err)
		assert.False(t, renumber)
		assert.Less(t, pos, list[0].Position)
	})

	t.Run("between neighbours", func(t *testing.T) {
		pos, renumber, err := positionAfter(list, &a, "afterCardId")
		require.NoError(t, err)
		assert.False(t, renumber)
		assert.Equal(t, 1536.0, pos)
	})

	t.Run("tail of the list", func(t *testing.T) {
		pos, renumber, err := positionAfter(list, &c, "afterCardId")
		require.NoError(t, err)
		assert.False(t, renumber)
		assert.Equal(t, 4096.0, pos)
	})

	t.Run("unknown anchor", func(t *testing.T) {
		stranger := uuid.New()
		_, _, err := positionAfter(list, &stranger, "afterCardId")
		require.Error(t, err)
		assert.Equal(t, domain.KindValidation, domain.KindOf(err))
		var de *domain.Error
		require.ErrorAs(t, err, &de)
		assert.Equal(t, "afterCardId", de.Fields[0].Field)
	})

	t.Run("anchor in an empty list", func(t *testing.T) {
		_, _, err := positionAfter(nil, &a, "afterCardId")
		assert.Equal(t, domain.KindValidation, domain.KindOf(err))
	})

	t.Run("exhausted gap asks for renumbering", func(t *testing.T) {
		tight := []placed{{a, 1}, {b, 1 + minPositionGap/2}}
		_, renumber, err := positionAfter(tight, &a, "afterCardId")
		require.NoError(t, err)
		assert.True(t, renumber)
	})

	t.Run("equal positions ask for renumbering", func(t *testing.T) {
		_, renumber, err := positionAfter([]placed{{a, 5}, {b, 5}}, &a, "afterCardId")
		require.NoError(t, err)
		assert.True(t, renumber)
	})
}

// Repeatedly inserting into the same gap must never yield an out-of-order
// position: it either stays strictly between the neighbours or asks to renumber.
func TestPositionAfterRepeatedInsertsStayOrdered(t *testing.T) {
	first, last := uuid.New(), uuid.New()
	list := []placed{{first, 1024}, {last, 2048}}

	for i := 0; i < 200; i++ {
		pos, renumber, err := positionAfter(list, &first, "afterCardId")
		require.NoError(t, err)
		if renumber {
			assert.Greater(t, i, 20, "a fresh gap must survive many inserts")
			return
		}
		require.Greater(t, pos, list[0].Position)
		require.Less(t, pos, list[1].Position)
		list = []placed{list[0], {uuid.New(), pos}}
	}
	t.Fatal("renumbering was never requested")
}
