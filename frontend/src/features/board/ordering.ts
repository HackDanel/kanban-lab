import type { Card, Column } from '../../shared/api/types'

/** Шаг между соседями — такой же, как на сервере. */
export const POSITION_STEP = 1024

export type CardsByColumn = Record<string, Card[]>

const byPosition = (a: { position: number; id: string }, b: { position: number; id: string }) =>
  a.position - b.position || a.id.localeCompare(b.id)

export function sortColumns(columns: Column[]): Column[] {
  return [...columns].sort(byPosition)
}

/** Раскладывает карточки по колонкам в порядке позиции. */
export function groupByColumn(cards: Card[], columns: Column[]): CardsByColumn {
  const groups: CardsByColumn = Object.fromEntries(columns.map((c) => [c.id, []]))
  for (const card of cards) groups[card.columnId]?.push(card)
  for (const list of Object.values(groups)) list.sort(byPosition)
  return groups
}

/** Позиция между соседями; приблизительная — точную назначит сервер. */
export function positionBetween(prev?: Card, next?: Card): number {
  if (prev && next) return (prev.position + next.position) / 2
  if (prev) return prev.position + POSITION_STEP
  if (next) return next.position - POSITION_STEP
  return POSITION_STEP
}

export interface MoveTarget {
  columnId: string
  afterCardId: string | null
  position: number
}

/**
 * Куда переместить карточку, отпущенную над `overId` — колонкой или другой
 * карточкой. Над колонкой карточка встаёт в конец, над карточкой — на её место.
 * Возвращает null, если положение не меняется.
 */
export function computeMove(
  groups: CardsByColumn,
  cardId: string,
  overId: string,
): MoveTarget | null {
  const source = Object.entries(groups).find(([, list]) => list.some((c) => c.id === cardId))
  if (!source) return null
  const [sourceColumnId, sourceList] = source
  const sourceIndex = sourceList.findIndex((c) => c.id === cardId)

  let targetColumnId: string
  let targetIndex: number
  if (overId in groups) {
    targetColumnId = overId
    targetIndex = groups[overId].filter((c) => c.id !== cardId).length
  } else {
    const over = Object.entries(groups).find(([, list]) => list.some((c) => c.id === overId))
    if (!over) return null
    targetColumnId = over[0]
    targetIndex = over[1].findIndex((c) => c.id === overId)
  }

  if (targetColumnId === sourceColumnId && targetIndex === sourceIndex) return null

  const without = groups[targetColumnId].filter((c) => c.id !== cardId)
  const prev = without[targetIndex - 1]
  const next = without[targetIndex]
  return {
    columnId: targetColumnId,
    afterCardId: prev?.id ?? null,
    position: positionBetween(prev, next),
  }
}

/** Колонка заполнена до WIP-лимита (перемещение внутри неё не учитывается). */
export function isColumnFull(column: Column, count: number): boolean {
  return column.wipLimit != null && count >= column.wipLimit
}
