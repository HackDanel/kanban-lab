import { describe, expect, it } from 'vitest'
import type { Activity } from '../../shared/api/types'
import { makeCard, makeUser } from '../../test/fixtures'
import {
  contrastText,
  describeActivity,
  diffCard,
  formatSize,
  hasFilters,
  initials,
  isOverdue,
  parseWipLimit,
  toForm,
} from './format'

describe('форматирование', () => {
  it('contrastText выбирает читаемый цвет текста', () => {
    expect(contrastText('#ffffff')).toBe('#000')
    expect(contrastText('#ffeb3b')).toBe('#000')
    expect(contrastText('#000000')).toBe('#fff')
    expect(contrastText('#0c66e4')).toBe('#fff')
  })

  it('initials берёт первые буквы двух слов', () => {
    expect(initials('Иван Петров')).toBe('ИП')
    expect(initials('  anna  ')).toBe('A')
    expect(initials('Мария Анна Ли')).toBe('МА')
  })

  it('isOverdue сравнивает с сегодняшней датой', () => {
    const today = new Date('2026-05-10T12:00:00Z')
    expect(isOverdue('2026-05-09', today)).toBe(true)
    expect(isOverdue('2026-05-10', today)).toBe(false)
    expect(isOverdue(null, today)).toBe(false)
  })

  it('parseWipLimit проверяет диапазон', () => {
    expect(parseWipLimit('')).toEqual({ value: null })
    expect(parseWipLimit(' 3 ')).toEqual({ value: 3 })
    expect(parseWipLimit('1000')).toEqual({ value: 1000 })
    for (const bad of ['0', '1001', '2.5', 'abc', '-1'])
      expect(parseWipLimit(bad).error).toBeDefined()
  })

  it('formatSize выбирает единицы', () => {
    expect(formatSize(512)).toBe('512 Б')
    expect(formatSize(2048)).toBe('2.0 КБ')
    expect(formatSize(5 * 1024 * 1024)).toBe('5.0 МБ')
  })

  it('hasFilters', () => {
    expect(hasFilters({})).toBe(false)
    expect(hasFilters({ q: '  ' })).toBe(false)
    expect(hasFilters({ labelId: [] })).toBe(false)
    expect(hasFilters({ q: 'x' })).toBe(true)
    expect(hasFilters({ priority: 'low' })).toBe(true)
    expect(hasFilters({ assigneeId: 'u' })).toBe(true)
    expect(hasFilters({ labelId: ['l'] })).toBe(true)
  })
})

describe('diffCard', () => {
  const card = makeCard({
    title: 'T',
    labelIds: ['a', 'b'],
    assigneeId: 'u1',
    dueDate: '2026-01-01',
  })

  it('без изменений — пустой патч', () => {
    expect(diffCard(card, toForm(card))).toEqual({})
    expect(diffCard(card, { ...toForm(card), labelIds: ['b', 'a'] })).toEqual({})
  })

  it('отправляет только изменённые поля, пустые значения — как null', () => {
    const form = {
      ...toForm(card),
      title: ' New ',
      description: 'D',
      priority: 'high' as const,
      assigneeId: '',
      dueDate: '',
      labelIds: ['a'],
    }
    expect(diffCard(card, form)).toEqual({
      title: 'New',
      description: 'D',
      priority: 'high',
      assigneeId: null,
      dueDate: null,
      labelIds: ['a'],
    })
  })

  it('назначает исполнителя и дедлайн', () => {
    const empty = makeCard()
    expect(diffCard(empty, { ...toForm(empty), assigneeId: 'u2', dueDate: '2026-02-01' })).toEqual({
      assigneeId: 'u2',
      dueDate: '2026-02-01',
    })
  })
})

describe('describeActivity', () => {
  const dev = makeUser('developer')
  const members = new Map([[dev.id, dev]])
  const entry = (overrides: Partial<Activity>): Activity => ({
    id: 'x',
    cardId: 'c',
    actorId: dev.id,
    action: 'created',
    createdAt: '2026-01-01T00:00:00Z',
    ...overrides,
  })

  it.each([
    [entry({ action: 'created' }), 'создал(а) карточку'],
    [
      entry({ action: 'moved', oldValue: 'To Do', newValue: 'Done' }),
      'переместил(а) из «To Do» в «Done»',
    ],
    [entry({ action: 'assigned', newValue: dev.id }), `назначил(а) исполнителя: ${dev.name}`],
    [entry({ action: 'assigned', newValue: null }), 'назначил(а) исполнителя: никто'],
    [entry({ action: 'assigned', newValue: 'gone' }), 'назначил(а) исполнителя: бывший участник'],
    [entry({ action: 'labels_changed' }), 'изменил(а) метки'],
    [entry({ action: 'commented' }), 'оставил(а) комментарий'],
    [entry({ action: 'attachment_added', newValue: 'a.pdf' }), 'прикрепил(а) файл «a.pdf»'],
    [entry({ action: 'attachment_removed', oldValue: 'a.pdf' }), 'удалил(а) файл «a.pdf»'],
    [entry({ action: 'updated', field: 'description' }), 'изменил(а) описание'],
    [
      entry({ action: 'updated', field: 'title', oldValue: 'A', newValue: 'B' }),
      'изменил(а) название: A → B',
    ],
    [
      entry({ action: 'updated', field: 'priority', oldValue: 'low', newValue: 'high' }),
      'изменил(а) приоритет: Низкий → Высокий',
    ],
    [
      entry({ action: 'updated', field: 'dueDate', oldValue: null, newValue: '2026-01-01' }),
      'изменил(а) дедлайн: — → 2026-01-01',
    ],
    [entry({ action: 'updated', field: 'other' }), 'изменил(а) other: — → —'],
  ])('%#', (e, text) => {
    expect(describeActivity(e, members)).toBe(text)
  })
})
