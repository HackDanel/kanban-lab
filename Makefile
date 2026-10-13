SHELL := /bin/sh
COMPOSE := docker compose

.DEFAULT_GOAL := help

.PHONY: help
help: ## Список команд
	@grep -E '^[a-zA-Z_-]+:.*## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*## "}; {printf "  %-14s %s\n", $$1, $$2}'

.PHONY: up
up: ## Поднять стек в Docker Compose
	$(COMPOSE) up -d --build --wait

.PHONY: down
down: ## Остановить стек
	$(COMPOSE) down

.PHONY: gen
gen: ## Сгенерировать код из OpenAPI и SQL-запросов
	cd backend && go generate ./...
	cd frontend && npm run gen:api

.PHONY: lint
lint: ## Линтеры backend и frontend
	cd backend && golangci-lint run ./...
	cd frontend && npm run lint && npm run typecheck

.PHONY: test
test: test-backend test-frontend ## Модульные и интеграционные тесты с покрытием

.PHONY: test-backend
test-backend:
	cd backend && go test -race -coverprofile=coverage.out ./...

.PHONY: test-frontend
test-frontend:
	cd frontend && npm run test:coverage

.PHONY: api-test
api-test: ## API-тесты: Postman-коллекция через Newman
	npx --yes newman run tests/postman/kanban.postman_collection.json \
		-e tests/postman/local.postman_environment.json \
		-r cli,junit --reporter-junit-export tests/postman/reports/newman.xml

.PHONY: e2e
e2e: ## E2E-тесты Playwright в трёх браузерах
	cd e2e && npx playwright test

.PHONY: load
load: ## Нагрузочный тест k6
	docker run --rm --network host -v $(CURDIR)/tests/load:/scripts grafana/k6 run /scripts/crud.js

.PHONY: security
security: ## Проверка зависимостей и ZAP baseline
	cd backend && go run golang.org/x/vuln/cmd/govulncheck@latest ./...
	cd frontend && npm audit --audit-level=high
	sh tests/security/zap-baseline.sh
