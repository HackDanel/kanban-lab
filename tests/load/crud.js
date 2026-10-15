// NFR-01, NFR-02: 1000 одновременных пользователей выполняют CRUD на доске с 10 000 карточек.
// Запуск: docker run --rm --network host -v $PWD/tests/load:/scripts grafana/k6 run /scripts/crud.js
import http from 'k6/http'
import { check, group, sleep } from 'k6'
import { BASE, json, seed } from './lib.js'

const VUS = Number(__ENV.VUS || 1000)
const CARDS = Number(__ENV.CARDS || 10000)

export const options = {
  setupTimeout: '15m',
  scenarios: {
    crud: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: __ENV.RAMP || '1m', target: VUS },
        { duration: __ENV.HOLD || '3m', target: VUS },
        { duration: '30s', target: 0 },
      ],
      gracefulRampDown: '30s',
    },
  },
  thresholds: {
    http_req_failed: ['rate<0.01'],
    http_req_duration: ['p(95)<200'],
    'http_req_duration{op:list}': ['p(95)<200'],
    'http_req_duration{op:search}': ['p(95)<200'],
    'http_req_duration{op:create}': ['p(95)<200'],
    'http_req_duration{op:update}': ['p(95)<200'],
    'http_req_duration{op:move}': ['p(95)<200'],
    'http_req_duration{op:delete}': ['p(95)<200'],
    checks: ['rate>0.99'],
  },
  summaryTrendStats: ['avg', 'min', 'med', 'p(90)', 'p(95)', 'p(99)', 'max'],
}

export function setup() {
  return seed(CARDS)
}

const queries = ['login', 'report', 'export', 'payment', 'task 42', 'mobile']

export default function (data) {
  const params = (op) => ({ ...json(data.token), tags: { op } })
  const column = data.columns[__VU % data.columns.length]

  group('read board', () => {
    const list = http.get(`${BASE}/boards/${data.boardId}/cards?limit=50&columnId=${column}`, params('list'))
    check(list, { 'list 200': (r) => r.status === 200 })
    const q = queries[__ITER % queries.length]
    const search = http.get(`${BASE}/boards/${data.boardId}/cards?limit=50&q=${encodeURIComponent(q)}`, params('search'))
    check(search, { 'search 200': (r) => r.status === 200 })
  })

  group('card lifecycle', () => {
    const created = http.post(
      `${BASE}/columns/${column}/cards`,
      JSON.stringify({ title: `vu ${__VU} iter ${__ITER}`, priority: 'medium' }),
      params('create'),
    )
    if (!check(created, { 'create 201': (r) => r.status === 201 })) return
    const id = created.json('id')

    const updated = http.patch(`${BASE}/cards/${id}`, JSON.stringify({ priority: 'high', description: 'updated by load test' }), params('update'))
    check(updated, { 'update 200': (r) => r.status === 200 })

    const target = data.columns[(__VU + 1) % data.columns.length]
    const moved = http.post(`${BASE}/cards/${id}/move`, JSON.stringify({ columnId: target, afterCardId: null }), params('move'))
    check(moved, { 'move 200': (r) => r.status === 200 })

    const removed = http.del(`${BASE}/cards/${id}`, null, params('delete'))
    check(removed, { 'delete 204': (r) => r.status === 204 })
  })

  sleep(1)
}

export function handleSummary(data) {
  return {
    stdout: textSummary(data),
    '/scripts/results/crud-summary.json': JSON.stringify(data, null, 2),
  }
}

function textSummary(data) {
  const m = data.metrics
  const line = (name, metric) =>
    metric ? `${name.padEnd(40)} p95=${metric.values['p(95)']?.toFixed(1)}ms p99=${metric.values['p(99)']?.toFixed(1)}ms avg=${metric.values.avg?.toFixed(1)}ms\n` : ''
  let out = '\n=== Kanban CRUD load test ===\n'
  out += `VUs max: ${m.vus_max?.values.max}, requests: ${m.http_reqs?.values.count}, rps: ${m.http_reqs?.values.rate.toFixed(1)}\n`
  out += `failed: ${(m.http_req_failed?.values.rate * 100).toFixed(2)}%\n`
  out += line('all requests', m.http_req_duration)
  for (const op of ['list', 'search', 'create', 'update', 'move', 'delete']) {
    out += line(op, m[`http_req_duration{op:${op}}`])
  }
  for (const [name, t] of Object.entries(data.metrics).filter(([, v]) => v.thresholds)) {
    for (const [expr, res] of Object.entries(t.thresholds)) out += `${res.ok ? 'PASS' : 'FAIL'} ${name} ${expr}\n`
  }
  return out
}
