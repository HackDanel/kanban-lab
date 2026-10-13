# Kanban Lab

Kanban-доска (аналог Jira) с полным циклом тестирования. Лабораторная работа № 16 по дисциплине «Поддержка и тестирование программных модулей».

- Frontend: React, TypeScript, Redux Toolkit, dnd-kit, Material UI
- Backend: Go, PostgreSQL, Redis
- Тесты: `go test`, Vitest, Newman, Playwright, k6, OWASP ZAP

Обоснование стека — в [STACK.md](STACK.md), требования и архитектура — в [docs/](docs/), контракт API — в [api/openapi.yaml](api/openapi.yaml).

## Состояние

| Этап | Статус |
|---|---|
| Каркас репозитория, контракт OpenAPI, требования, архитектура | готово |
| Frontend: аутентификация, список проектов, модульные тесты | готово |
| Backend | не начат |
| Frontend: доска, карточки, drag-and-drop | не начат |
| Интеграционные, API, E2E, нагрузочные, security-тесты | не начаты |
| CI/CD | проверки frontend и контракта; сборка образов и деплой не настроены |

## Требования к окружению

- Node.js 22 или новее
- Go (версия — в `backend/go.mod`, появится вместе с сервером)
- Docker с Docker Compose

## Запуск

```sh
cp .env.example .env
docker compose up -d      # PostgreSQL и Redis
cd frontend
npm ci
npm run dev               # http://localhost:5173, запросы /api проксируются на :8080
```

Сервера пока нет, поэтому интерфейс открывается, но вход и регистрация вернут ошибку.

## Тесты

```sh
cd frontend
npm run lint
npm run typecheck
npm run test:coverage     # пороги: 80 % строк, 70 % ветвей
```

Типы клиента генерируются из контракта: `npm run gen:api`.

## Структура

```
api/        контракт OpenAPI
backend/    сервер на Go
frontend/   клиент
e2e/        сценарии Playwright
tests/      Postman, k6, ZAP
deploy/     Nginx, Compose для staging и production, Prometheus, Grafana
docs/       требования, архитектура, тест-план, отчёты
```
