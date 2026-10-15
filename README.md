# Kanban Lab

Kanban-доска (аналог Jira) с полным циклом тестирования. Лабораторная работа № 16 по дисциплине «Поддержка и тестирование программных модулей».

- Frontend: React, TypeScript, Redux Toolkit, dnd-kit, Material UI
- Backend: Go, PostgreSQL, Redis, WebSocket
- Тесты: `go test` + testcontainers, Vitest, Newman, Playwright, k6, OWASP ZAP
- CI/CD: GitHub Actions, образы в GHCR, деплой на сервер

Стенды: staging — https://staging.37.194.184.235.sslip.io, production — https://kanban.37.194.184.235.sslip.io.

| Документ | Содержание |
|---|---|
| [STACK.md](STACK.md) | обоснование выбора стека |
| [docs/requirements.md](docs/requirements.md) | требования и их трассировка до API и тест-кейсов |
| [docs/architecture.md](docs/architecture.md) | архитектура, модель данных, ключевые решения |
| [api/openapi.yaml](api/openapi.yaml) | контракт API; Swagger UI — `/docs` на запущенном стенде |
| [docs/test-plan.md](docs/test-plan.md) | план тестирования |
| [docs/test-cases.md](docs/test-cases.md) | тест-кейсы и их автотесты |
| [docs/bug-reports/](docs/bug-reports/) | баг-репорты |
| [docs/usability-checklist.md](docs/usability-checklist.md) | чек-лист юзабилити |
| [docs/test-report.md](docs/test-report.md) | отчёт о тестировании |
| [docs/deployment.md](docs/deployment.md) | стенды, подготовка сервера и деплой |
| [docs/presentation/kanban-lab.pptx](docs/presentation/kanban-lab.pptx) | презентация к защите (собирается `node docs/presentation/build.js`, снимки экрана — `SHOTS=1 npx playwright test zz-screenshots` в `e2e/`) |

## Возможности

- регистрация, вход, выход; JWT, обновление сессии, защита от CSRF; роли администратора, менеджера и разработчика;
- проекты и их участники, доски, колонки с порядком и WIP-лимитами;
- карточки: исполнитель, приоритет, дедлайн, метки, описание, комментарии, вложения, перемещение drag-and-drop;
- поиск по тексту и фильтры по исполнителю, меткам, приоритету;
- история изменений карточки;
- обновление доски у всех пользователей в реальном времени;
- метрики Prometheus и дашборд Grafana.

## Запуск

Нужен Docker с Docker Compose.

```sh
docker compose up -d --build --wait
```

- приложение: http://localhost:8000 (вход `admin@example.com` / `admin-password`, задаются переменными `ADMIN_EMAIL`, `ADMIN_PASSWORD`);
- документация API: http://localhost:8000/docs;
- мониторинг: `docker compose --profile monitoring up -d`, затем Grafana на http://localhost:3000 (`admin` / `admin`), Prometheus на http://localhost:9090.

Новые пользователи регистрируются разработчиками; роль менеджера выдаёт администратор на странице «Пользователи».

### Разработка

Нужны Go 1.27 и Node.js 22 или новее.

```sh
docker compose up -d postgres redis
cp .env.example .env    # и заполнить JWT_SECRET
cd backend && set -a && . ../.env && set +a && go run ./cmd/server
cd frontend && npm ci && npm run dev    # http://localhost:5173
```

Код из контракта и SQL генерируется командой `make gen` (для sqlc нужен Docker).

## Тесты

```sh
make test          # модульные и интеграционные тесты, покрытие (нужен Docker для testcontainers)
make coverage-branches   # покрытие ветвей бэкенда (gobco)
make lint
```

Для API-, E2E-, нагрузочных тестов и сканирования нужен запущенный стек. Тесты создают много входов подряд, поэтому лимит попыток входа на время прогона поднимается:

```sh
AUTH_RATE_LIMIT=10000 docker compose up -d --build --wait
make api-test      # Newman
cd e2e && npm ci && npx playwright install chromium firefox msedge && cd ..
make e2e           # Playwright: Chromium, Firefox, Edge (E2E_BROWSERS=chromium,firefox — без Edge)
make load          # k6: 1000 пользователей, 10 000 карточек
make load-rps      # k6: постоянные 1000 запросов в секунду
make security      # govulncheck, gosec, npm audit, OWASP ZAP
```

## CI/CD

`.github/workflows/ci.yml`: lint → модульные тесты → сборка образов → интеграционные тесты → E2E в трёх браузерах и Newman → сканирование безопасности → деплой на staging → деплой на production (только из `main`). Пуш в `develop` выкатывает staging, пуш в `main` — staging и затем production. Отчёты (покрытие, трассировки Playwright, Newman, ZAP) сохраняются как артефакты сборки.

`.github/workflows/load.yml` — нагрузочный тест staging вручную и раз в неделю.

## Структура

```
api/        контракт OpenAPI
backend/    сервер на Go: cmd/, internal/{http,service,repository,domain,realtime,platform}, migrations/, queries/, tests/
frontend/   клиент: src/{app,features,pages,shared}
e2e/        сценарии Playwright
tests/      postman/, load/ (k6), security/ (ZAP и сканеры)
deploy/     стек для VPS, скрипты деплоя и бэкапа, nginx, Prometheus, Grafana
docs/       требования, архитектура, тест-план, тест-кейсы, баг-репорты, отчёт
```
