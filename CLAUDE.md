# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

Kanban board (Jira-like) with a full testing cycle — lab work No. 16, team Панов Д. С. and Гостев Д. С., group 9ИС-360. Project documents, UI texts and API error messages are in Russian; code comments and commit messages are in English. `docs/test-report.md` and `README.md` describe the current state.

## Commands

Everything heavy needs Docker (testcontainers, sqlc, k6, ZAP). Go 1.27, Node 22+.

```sh
docker compose up -d --build --wait            # full stack on http://localhost:8000, admin@example.com / admin-password
docker compose --profile monitoring up -d      # + Prometheus :9090, Grafana :3000

# backend (cd backend)
go test ./internal/...                          # unit tests
go test ./tests/integration/ -run TestMoveCard  # one integration test (starts PostgreSQL and Redis containers)
golangci-lint run ./...
go generate ./internal/http/api/                # after editing api/openapi.yaml
go generate ./internal/repository/              # after editing queries/*.sql (runs sqlc in Docker)
sh scripts/branch-coverage.sh                   # branch coverage via gobco, fails below 70 %

# frontend (cd frontend)
npm run dev                                     # :5173, proxies /api to :8080
npx vitest run src/pages/BoardPage.test.tsx -t 'WIP'
npm run lint && npm run format:check && npm run typecheck
npm run test:coverage                           # thresholds 80 % lines / 70 % branches
npm run gen:api                                 # regenerate src/shared/api/schema.d.ts from the contract

# against a running stack — raise the login rate limit first, the tests log in a lot
AUTH_RATE_LIMIT=10000 docker compose up -d --wait
make api-test                                   # Newman
cd e2e && E2E_BROWSERS=chromium npx playwright test tests/board.spec.ts
make load / make load-rps / make security       # k6 and scanners
```

If Go module downloads hang, use `GOPROXY='https://proxy.golang.org|https://goproxy.io|direct'`. The local Docker has no buildx, so Dockerfiles must not use BuildKit-only syntax such as `RUN --mount`.

## Contract first

`api/openapi.yaml` is the single source of the API. From it come the strict server interfaces and models in `backend/internal/http/api/api.gen.go` (oapi-codegen, with `nullable-type`, so PATCH bodies tell an absent field from an explicit `null`) and the client types in `frontend/src/shared/api/schema.d.ts`. CI fails if either generated file is stale. Every request is also checked against the spec at runtime by the kin-openapi middleware (`internal/http/validation.go`); input schemas use `additionalProperties: false`.

## Backend structure

`cmd/server` wires everything. Requests flow `internal/http` (handlers implement the generated `StrictServerInterface`) → `internal/service` (all business rules) → `internal/repository` (sqlc code plus `Store.InTx`).

- Services return `*domain.Error` with a `Kind`; `internal/http/errors.go` maps kinds to status codes and the API error format. Unknown errors become 500 without details.
- Access checks live in `service/access.go`: admins see everything, others only projects they are members of; managing boards, columns and labels needs the admin or manager role; changing a project or its members needs admin or the owning manager.
- Middleware order matters (`http/router.go`): body limit → `rejectNUL` → `authenticate` (an invalid token just leaves the request anonymous) → login rate limit → OpenAPI validator, which enforces each operation's security requirement. Multipart uploads skip the validator and are streamed to storage by the handler. `realIP` trusts `X-Real-IP` only from loopback and private addresses.
- Handlers reach cookies through the request and response stored in the context by the `withExchange` strict middleware.

Invariants found under load testing — keep them when changing card code:

- WIP limit: count-then-insert happens under `SELECT … FOR UPDATE` on the column row, and only when the column has a limit; reordering within a column ignores the limit.
- Card numbers come from a PostgreSQL sequence per project (`repository/card_numbers.go`), created with the project and dropped with it. A counter column serialised all card creation.
- Positions are floats between neighbours; only the two neighbours are read through the `(column_id, position)` index, and the column is renumbered when the gap is exhausted.
- Card search is built by hand in `repository/cards_search.go` instead of sqlc: a catch-all `($n IS NULL OR …)` prepared query gets a generic plan that scans the whole table. The page of ids is chosen before joining `card_views`, whose subqueries would otherwise run for every row skipped by OFFSET.

Migrations are goose files in `backend/migrations`, embedded and applied at server start under an advisory lock. Add a new numbered file; never edit an applied one.

Auth: access JWT for 15 minutes held only in client memory; a rotating refresh token in an httpOnly cookie scoped to `/api/v1/auth`, with reuse detection revoking the whole family; a double-submit CSRF check on refresh and logout. The role is read from the token, so a role change applies at the next refresh.

Real-time: services publish `service.Event` after a commit; `internal/realtime` fans events out through Redis pub/sub to WebSocket clients of the board (`/api/v1/ws?boardId=…&token=…`).

## Frontend structure

React 19 + Redux Toolkit. All server state goes through RTK Query: `shared/api/baseApi.ts` adds the Bearer token and refreshes once on 401 for all parallel failures; features inject endpoints with cache tags. `features/board/useBoardEvents.ts` turns WebSocket events into tag invalidations. Drag-and-drop decisions are pure functions in `features/board/ordering.ts`; display helpers are in `features/board/format.ts` (components files may export only components). Keyboard: Space picks up a card, Enter opens it.

Tests use MSW: `src/test/fakeApi.ts` is an in-memory backend for page tests and records requests; `src/test/fakeSocket.ts` replaces `WebSocket`. `onUnhandledFrame: 'error'` makes any unmocked request fail the test.

## Other tests and artefacts

- `tests/postman/kanban.postman_collection.json` is generated: edit `build_collection.py` and rerun it. The response-time limit is `maxResponseMs` (200 by default).
- E2E helpers in `e2e/tests/api.ts` create users and projects through the API; `zz-screenshots.spec.ts` runs only with `SHOTS=1` and refreshes the presentation images.
- `docs/bug-reports/BUG-NNN.md` follow template 7.1 of the assignment and describe only defects actually found by a test or scan, with the fixing commit; keep `docs/bug-reports/README.md` and `docs/test-report.md` in sync.
- The presentation is generated: `docs/presentation/build.js` reads `data.json` (needs `pptxgenjs`).

## CI/CD and deployment

`.github/workflows/ci.yml`: lint → unit → build images (pushed to GHCR) → integration with coverage gates, E2E in Chromium/Firefox/Edge, Newman, security (govulncheck, gosec, npm audit, Trivy, ZAP) → deploy. Push to `develop` deploys staging (https://staging.37.194.184.235.sslip.io), push to `main` deploys staging and then production (https://kanban.37.194.184.235.sslip.io).

The stands run on a shared home server (`b450`) behind its existing Apache, alongside other services. The CI SSH key is restricted to `deploy/ci-command.sh`, which downloads the `deploy/` files of the commit with the job token and runs `deploy/deploy.sh` (pull with retries, health wait, rollback). Changes under `deploy/` therefore take effect on the next deployment. Do not run heavy load tests against these stands; `load.yml` is manual-only for that reason. Details are in `docs/deployment.md`.
