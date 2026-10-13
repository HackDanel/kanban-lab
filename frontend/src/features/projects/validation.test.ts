import { describe, expect, it } from 'vitest'
import { validateProject } from './validation'

describe('validateProject', () => {
  it('принимает корректные данные', () => {
    expect(validateProject({ key: 'KAN', name: 'Kanban' })).toEqual({})
    expect(validateProject({ key: 'A1', name: 'x' })).toEqual({})
    expect(validateProject({ key: 'ABCDEFGHIJ', name: 'x' })).toEqual({})
  })

  it.each(['', 'K', 'kan', '1AB', 'ABCDEFGHIJK', 'КАН', 'A-B'])('отклоняет ключ «%s»', (key) => {
    expect(validateProject({ key, name: 'Kanban' }).key).toBeDefined()
  })

  it('требует название', () => {
    expect(validateProject({ key: 'KAN', name: ' ' })).toEqual({ name: 'Название обязательно' })
  })
})
