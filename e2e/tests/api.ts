import { randomUUID } from 'node:crypto'
import type { APIRequestContext } from '@playwright/test'

/** Учётная запись администратора из docker-compose.yml (переопределяется переменными окружения). */
export const ADMIN = {
  email: process.env.ADMIN_EMAIL ?? 'admin@example.com',
  password: process.env.ADMIN_PASSWORD ?? 'admin-password',
}

export type Role = 'admin' | 'manager' | 'developer'

export interface Account {
  id: string
  email: string
  password: string
  name: string
  token: string
}

const API = '/api/v1'

async function json<T>(response: Awaited<ReturnType<APIRequestContext['get']>>): Promise<T> {
  if (!response.ok()) {
    throw new Error(`${response.url()} → ${response.status()}: ${await response.text()}`)
  }
  return (await response.json()) as T
}

export function unique(prefix: string): string {
  return `${prefix}-${randomUUID().slice(0, 8)}`
}

/** Ключ проекта: латинские заглавные буквы. */
export function uniqueKey(): string {
  return 'E' + randomUUID().replace(/[^a-f]/g, '').toUpperCase().slice(0, 8).padEnd(4, 'A')
}

export async function login(request: APIRequestContext, email: string, password: string): Promise<string> {
  const body = await json<{ accessToken: string }>(
    await request.post(`${API}/auth/login`, { data: { email, password } }),
  )
  return body.accessToken
}

/** Регистрирует пользователя и при необходимости выдаёт роль от имени администратора. */
export async function createAccount(request: APIRequestContext, role: Role, name = unique('Пользователь')): Promise<Account> {
  const email = `${unique('e2e')}@example.com`
  const password = 'password123'
  const user = await json<{ id: string }>(
    await request.post(`${API}/auth/register`, { data: { email, password, name } }),
  )
  if (role !== 'developer') await setRole(request, user.id, role)
  return { id: user.id, email, password, name, token: await login(request, email, password) }
}

export async function setRole(request: APIRequestContext, userId: string, role: Role) {
  const admin = await login(request, ADMIN.email, ADMIN.password)
  await json(
    await request.put(`${API}/users/${userId}/role`, {
      data: { role },
      headers: { Authorization: `Bearer ${admin}` },
    }),
  )
}

export class Client {
  constructor(
    private readonly request: APIRequestContext,
    private readonly token: string,
  ) {}

  private headers() {
    return { Authorization: `Bearer ${this.token}` }
  }

  get<T>(path: string) {
    return this.request.get(API + path, { headers: this.headers() }).then(json<T>)
  }

  post<T>(path: string, data?: unknown) {
    return this.request.post(API + path, { headers: this.headers(), data }).then(json<T>)
  }

  patch<T>(path: string, data: unknown) {
    return this.request.patch(API + path, { headers: this.headers(), data }).then(json<T>)
  }

  raw(method: 'DELETE' | 'GET', path: string) {
    return this.request.fetch(API + path, { method, headers: this.headers() })
  }
}

export interface Column {
  id: string
  name: string
}

export interface Board {
  id: string
  columns: Column[]
}

/** Проект менеджера с доской и колонками по умолчанию; участники добавляются сразу. */
export async function createProjectWithBoard(request: APIRequestContext, owner: Account, members: Account[] = []) {
  const client = new Client(request, owner.token)
  const project = await client.post<{ id: string; key: string; name: string }>('/projects', {
    key: uniqueKey(),
    name: unique('Проект'),
  })
  for (const m of members) {
    const r = await request.post(`${API}/projects/${project.id}/members`, {
      data: { userId: m.id },
      headers: { Authorization: `Bearer ${owner.token}` },
    })
    if (!r.ok()) throw new Error(await r.text())
  }
  const board = await client.post<Board>(`/projects/${project.id}/boards`, { name: 'Спринт' })
  return { client, project, board, column: (name: string) => board.columns.find((c) => c.name === name)! }
}
