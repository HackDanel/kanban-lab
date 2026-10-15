// Чек-лист 7.2: «Нагрузочный тест выдерживает 1000 RPS» — постоянный поток чтения доски.
import http from 'k6/http'
import { check } from 'k6'
import { BASE, json, seed } from './lib.js'

const RATE = Number(__ENV.RATE || 1000)

export const options = {
  setupTimeout: '15m',
  scenarios: {
    rps: {
      executor: 'constant-arrival-rate',
      rate: RATE,
      timeUnit: '1s',
      duration: __ENV.DURATION || '2m',
      preAllocatedVUs: 200,
      maxVUs: 2000,
    },
  },
  thresholds: {
    http_req_failed: ['rate<0.01'],
    http_req_duration: ['p(95)<200'],
    dropped_iterations: ['count<' + Math.ceil(RATE * 0.01 * 120)],
  },
  summaryTrendStats: ['avg', 'med', 'p(90)', 'p(95)', 'p(99)', 'max'],
}

export function setup() {
  return seed(Number(__ENV.CARDS || 10000))
}

export default function (data) {
  const column = data.columns[__ITER % data.columns.length]
  const res = http.get(`${BASE}/boards/${data.boardId}/cards?limit=50&columnId=${column}&offset=${(__ITER * 50) % 3000}`, json(data.token))
  check(res, { 'status 200': (r) => r.status === 200 })
}

export function handleSummary(data) {
  return { '/scripts/results/rps-summary.json': JSON.stringify(data, null, 2), stdout: JSON.stringify({
    rps: data.metrics.http_reqs.values.rate,
    p95: data.metrics.http_req_duration.values['p(95)'],
    failed: data.metrics.http_req_failed.values.rate,
    dropped: data.metrics.dropped_iterations?.values.count ?? 0,
  }, null, 2) + '\n' }
}
