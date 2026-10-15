// Чистые функции доски: форматирование, преобразования форм, описания истории.
import type { Activity, Card, CardPatch, Priority, User } from '../../shared/api/types'
import type { CardFilters } from './boardApi'

export const PRIORITY_LABELS: Record<Priority, string> = {
  low: 'Низкий',
  medium: 'Средний',
  high: 'Высокий',
  critical: 'Критический',
}

export const PRIORITY_COLORS: Record<Priority, 'default' | 'info' | 'warning' | 'error'> = {
  low: 'default',
  medium: 'info',
  high: 'warning',
  critical: 'error',
}

/** Чёрный или белый текст в зависимости от яркости фона метки. */
export function contrastText(hex: string): string {
  const n = Number.parseInt(hex.slice(1), 16)
  const r = (n >> 16) & 255
  const g = (n >> 8) & 255
  const b = n & 255
  return 0.299 * r + 0.587 * g + 0.114 * b > 150 ? '#000' : '#fff'
}

export function initials(name: string): string {
  return name
    .split(/\s+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((part) => part[0]!.toUpperCase())
    .join('')
}

export function isOverdue(dueDate: string | null | undefined, today = new Date()): boolean {
  if (!dueDate) return false
  return dueDate < today.toISOString().slice(0, 10)
}

export function parseWipLimit(raw: string): { value: number | null; error?: string } {
  const trimmed = raw.trim()
  if (trimmed === '') return { value: null }
  const n = Number(trimmed)
  if (!Number.isInteger(n) || n < 1 || n > 1000) {
    return { value: null, error: 'Целое число от 1 до 1000 или пусто' }
  }
  return { value: n }
}

export function formatDateTime(iso: string): string {
  return new Date(iso).toLocaleString('ru-RU', { dateStyle: 'short', timeStyle: 'short' })
}

export const MAX_UPLOAD_BYTES = 10 * 1024 * 1024

export function formatSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} Б`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} КБ`
  return `${(bytes / 1024 / 1024).toFixed(1)} МБ`
}

export function hasFilters(f: CardFilters): boolean {
  return Boolean(f.q?.trim() || f.assigneeId || f.priority || f.labelId?.length)
}

export interface CardForm {
  title: string
  description: string
  priority: Priority
  assigneeId: string
  dueDate: string
  labelIds: string[]
}

export function toForm(card: Card): CardForm {
  return {
    title: card.title,
    description: card.description,
    priority: card.priority,
    assigneeId: card.assigneeId ?? '',
    dueDate: card.dueDate ?? '',
    labelIds: card.labelIds,
  }
}

/** Только изменённые поля; пустые исполнитель и дедлайн отправляются как null. */
export function diffCard(card: Card, form: CardForm): CardPatch {
  const patch: CardPatch = {}
  if (form.title.trim() !== card.title) patch.title = form.title.trim()
  if (form.description !== card.description) patch.description = form.description
  if (form.priority !== card.priority) patch.priority = form.priority
  if ((form.assigneeId || null) !== (card.assigneeId ?? null))
    patch.assigneeId = form.assigneeId || null
  if ((form.dueDate || null) !== (card.dueDate ?? null)) patch.dueDate = form.dueDate || null
  const sorted = (ids: string[]) => [...ids].sort().join()
  if (sorted(form.labelIds) !== sorted(card.labelIds)) patch.labelIds = form.labelIds
  return patch
}

const FIELD_NAMES: Record<string, string> = {
  title: 'название',
  description: 'описание',
  priority: 'приоритет',
  dueDate: 'дедлайн',
}

const show = (value: string | null | undefined) => value || '—'

/** Человекочитаемое описание записи истории. */
export function describeActivity(entry: Activity, members: Map<string, User>): string {
  const name = (id: string | null | undefined) =>
    id ? (members.get(id)?.name ?? 'бывший участник') : 'никто'
  switch (entry.action) {
    case 'created':
      return 'создал(а) карточку'
    case 'moved':
      return `переместил(а) из «${show(entry.oldValue)}» в «${show(entry.newValue)}»`
    case 'assigned':
      return `назначил(а) исполнителя: ${name(entry.newValue)}`
    case 'labels_changed':
      return 'изменил(а) метки'
    case 'commented':
      return 'оставил(а) комментарий'
    case 'attachment_added':
      return `прикрепил(а) файл «${show(entry.newValue)}»`
    case 'attachment_removed':
      return `удалил(а) файл «${show(entry.oldValue)}»`
    case 'updated': {
      const field = FIELD_NAMES[entry.field ?? ''] ?? entry.field ?? 'поле'
      if (entry.field === 'description') return 'изменил(а) описание'
      const fmt = (v: string | null | undefined) =>
        entry.field === 'priority' && v
          ? (PRIORITY_LABELS[v as keyof typeof PRIORITY_LABELS] ?? v)
          : show(v)
      return `изменил(а) ${field}: ${fmt(entry.oldValue)} → ${fmt(entry.newValue)}`
    }
    default:
      return entry.action
  }
}
