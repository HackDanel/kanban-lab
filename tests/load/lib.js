import http from 'k6/http'
import { check, fail } from 'k6'

export const BASE = `${__ENV.BASE_URL || 'http://localhost:8000'}/api/v1`
const ADMIN_EMAIL = __ENV.ADMIN_EMAIL || 'admin@example.com'
const ADMIN_PASSWORD = __ENV.ADMIN_PASSWORD || 'admin-password'

export function json(token) {
  const headers = { 'Content-Type': 'application/json' }
  if (token) headers.Authorization = `Bearer ${token}`
  return { headers }
}

export function expectOk(res, what) {
  if (res.status < 200 || res.status >= 300) fail(`${what}: ${res.status} ${res.body}`)
  return res.json()
}

function login(email, password) {
  return expectOk(http.post(`${BASE}/auth/login`, JSON.stringify({ email, password }), json()), 'login').accessToken
}

/**
 * Готовит данные: менеджера, проект, доску и `cards` карточек, распределённых по
 * колонкам. Возвращает токен и идентификаторы для сценариев.
 */
export function seed(cards) {
  const suffix = `${Date.now()}`
  const email = `load-${suffix}@example.com`
  const user = expectOk(
    http.post(`${BASE}/auth/register`, JSON.stringify({ email, name: 'Load test', password: 'password123' }), json()),
    'register',
  )
  const admin = login(ADMIN_EMAIL, ADMIN_PASSWORD)
  expectOk(http.put(`${BASE}/users/${user.id}/role`, JSON.stringify({ role: 'manager' }), json(admin)), 'set role')
  const token = login(email, 'password123')

  const key = 'L' + suffix.slice(-8).replace(/\d/g, (d) => 'ABCDEFGHIJ'[d])
  const project = expectOk(http.post(`${BASE}/projects`, JSON.stringify({ key, name: 'Load test' }), json(token)), 'project')
  const board = expectOk(
    http.post(`${BASE}/projects/${project.id}/boards`, JSON.stringify({ name: 'Load' }), json(token)),
    'board',
  )
  const columns = board.columns.map((c) => c.id)

  const words = ['login', 'report', 'export', 'payment', 'search', 'profile', 'api', 'mobile', 'bug', 'refactor']
  const priorities = ['low', 'medium', 'high', 'critical']
  const BATCH = 100
  for (let start = 0; start < cards; start += BATCH) {
    const requests = []
    for (let i = start; i < Math.min(cards, start + BATCH); i++) {
      requests.push([
        'POST',
        `${BASE}/columns/${columns[i % columns.length]}/cards`,
        JSON.stringify({
          title: `${words[i % words.length]} task ${i}`,
          description: `Seeded card ${i} about ${words[(i * 7) % words.length]}`,
          priority: priorities[i % priorities.length],
        }),
        json(token),
      ])
    }
    for (const res of http.batch(requests)) {
      check(res, { 'seed card created': (r) => r.status === 201 })
    }
  }
  return { token, boardId: board.id, columns, userId: user.id }
}
