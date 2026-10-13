import type { AuthResponse, Project, Role, User } from '../shared/api/types'

export function makeUser(role: Role = 'developer', overrides: Partial<User> = {}): User {
  return {
    id: `00000000-0000-4000-8000-00000000000${role.length}`,
    email: `${role}@example.com`,
    name: `Test ${role}`,
    role,
    createdAt: '2026-01-01T00:00:00Z',
    ...overrides,
  }
}

export function makeAuth(user: User = makeUser(), accessToken = 'access-1'): AuthResponse {
  return { accessToken, expiresIn: 900, user }
}

export function makeProject(overrides: Partial<Project> = {}): Project {
  return {
    id: '10000000-0000-4000-8000-000000000001',
    key: 'KAN',
    name: 'Kanban',
    description: 'Учебный проект',
    ownerId: makeUser('manager').id,
    createdAt: '2026-01-01T00:00:00Z',
    updatedAt: '2026-01-01T00:00:00Z',
    ...overrides,
  }
}
