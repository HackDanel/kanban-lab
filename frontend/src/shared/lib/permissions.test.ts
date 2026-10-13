import { describe, expect, it } from 'vitest'
import { makeProject, makeUser } from '../../test/fixtures'
import { canEditProject, canManage, isAdmin } from './permissions'

describe('права доступа', () => {
  it.each([
    ['admin', true],
    ['manager', true],
    ['developer', false],
  ] as const)('canManage для роли %s: %s', (role, expected) => {
    expect(canManage(makeUser(role))).toBe(expected)
  })

  it('без пользователя ничего не разрешено', () => {
    expect(canManage(null)).toBe(false)
    expect(isAdmin(null)).toBe(false)
    expect(canEditProject(null, makeProject())).toBe(false)
  })

  it('администратор редактирует любой проект', () => {
    expect(canEditProject(makeUser('admin'), makeProject({ ownerId: 'someone' }))).toBe(true)
    expect(isAdmin(makeUser('admin'))).toBe(true)
  })

  it('менеджер редактирует только свой проект', () => {
    const manager = makeUser('manager')
    expect(canEditProject(manager, makeProject({ ownerId: manager.id }))).toBe(true)
    expect(canEditProject(manager, makeProject({ ownerId: 'someone' }))).toBe(false)
  })

  it('разработчик не редактирует проект, даже если значится владельцем', () => {
    const developer = makeUser('developer')
    expect(canEditProject(developer, makeProject({ ownerId: developer.id }))).toBe(false)
  })
})
