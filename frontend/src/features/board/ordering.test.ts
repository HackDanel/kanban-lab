import { describe, expect, it } from 'vitest'
import { makeCard, makeColumn } from '../../test/fixtures'
import {
  computeMove,
  groupByColumn,
  isColumnFull,
  positionBetween,
  POSITION_STEP,
  sortColumns,
} from './ordering'

const todo = makeColumn({ id: 'todo', position: 1024 })
const doing = makeColumn({ id: 'doing', position: 2048, wipLimit: 2 })
const a = makeCard({ id: 'a', columnId: 'todo', position: 1024 })
const b = makeCard({ id: 'b', columnId: 'todo', position: 2048 })
const c = makeCard({ id: 'c', columnId: 'todo', position: 3072 })
const d = makeCard({ id: 'd', columnId: 'doing', position: 1024 })
const groups = groupByColumn([c, d, a, b], [doing, todo])

describe('groupByColumn', () => {
  it('раскладывает карточки по колонкам в порядке позиции', () => {
    expect(groups.todo.map((x) => x.id)).toEqual(['a', 'b', 'c'])
    expect(groups.doing.map((x) => x.id)).toEqual(['d'])
  })

  it('игнорирует карточки неизвестных колонок и создаёт пустые списки', () => {
    const g = groupByColumn([makeCard({ columnId: 'gone' })], [todo])
    expect(g).toEqual({ todo: [] })
  })

  it('при равных позициях упорядочивает по id', () => {
    const g = groupByColumn(
      [
        makeCard({ id: 'y', columnId: 'todo', position: 1 }),
        makeCard({ id: 'x', columnId: 'todo', position: 1 }),
      ],
      [todo],
    )
    expect(g.todo.map((x) => x.id)).toEqual(['x', 'y'])
  })
})

describe('sortColumns', () => {
  it('сортирует колонки по позиции, не меняя исходный массив', () => {
    const input = [doing, todo]
    expect(sortColumns(input).map((x) => x.id)).toEqual(['todo', 'doing'])
    expect(input[0]).toBe(doing)
  })
})

describe('positionBetween', () => {
  it('считает середину, начало, конец и пустой список', () => {
    expect(positionBetween(a, b)).toBe(1536)
    expect(positionBetween(c)).toBe(3072 + POSITION_STEP)
    expect(positionBetween(undefined, a)).toBe(0)
    expect(positionBetween()).toBe(POSITION_STEP)
  })
})

describe('computeMove', () => {
  it('в другую колонку на пустое место — в конец', () => {
    expect(computeMove(groups, 'a', 'doing')).toEqual({
      columnId: 'doing',
      afterCardId: 'd',
      position: 2048,
    })
  })

  it('на место другой карточки в чужой колонке', () => {
    expect(computeMove(groups, 'b', 'd')).toEqual({
      columnId: 'doing',
      afterCardId: null,
      position: 0,
    })
  })

  it('вниз внутри колонки — после карточки, над которой отпустили (как arrayMove)', () => {
    expect(computeMove(groups, 'a', 'c')).toEqual({
      columnId: 'todo',
      afterCardId: 'c',
      position: 4096,
    })
    expect(computeMove(groups, 'a', 'b')).toEqual({
      columnId: 'todo',
      afterCardId: 'b',
      position: 2560,
    })
  })

  it('вверх внутри колонки', () => {
    expect(computeMove(groups, 'c', 'a')).toEqual({
      columnId: 'todo',
      afterCardId: null,
      position: 0,
    })
  })

  it('в конец своей колонки при отпускании над колонкой', () => {
    expect(computeMove(groups, 'a', 'todo')).toEqual({
      columnId: 'todo',
      afterCardId: 'c',
      position: 4096,
    })
  })

  it('ничего не делает, если положение не меняется', () => {
    expect(computeMove(groups, 'b', 'b')).toBeNull()
    expect(computeMove(groups, 'd', 'doing')).toBeNull()
  })

  it('ничего не делает для неизвестных карточек и целей', () => {
    expect(computeMove(groups, 'nope', 'doing')).toBeNull()
    expect(computeMove(groups, 'a', 'nowhere')).toBeNull()
  })
})

describe('isColumnFull', () => {
  it('учитывает WIP-лимит', () => {
    expect(isColumnFull(todo, 100)).toBe(false)
    expect(isColumnFull(doing, 1)).toBe(false)
    expect(isColumnFull(doing, 2)).toBe(true)
  })
})
