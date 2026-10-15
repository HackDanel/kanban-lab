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
	cd backend && go test -race -coverpkg=./internal/... -coverprofile=coverage.raw ./...
	cd backend && grep -v -E '/internal/http/api/api.gen.go|/internal/repository/.*\.sql\.go|/internal/repository/(db|models|querier)\.go' coverage.raw > coverage.out
	cd backend && go tool cover -func=coverage.out | tail -1

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
	docker run --rm --network host -u $$(id -u):$$(id -g) -v $(CURDIR)/tests/load:/scripts grafana/k6 run /scripts/crud.js

.PHONY: security
security: ## Сканеры зависимостей и OWASP ZAP по запущенному стеку
	sh tests/security/deps.sh
	sh tests/security/zap-baseline.sh

.PHONY: load-rps
load-rps: ## Нагрузочный тест: постоянные 1000 запросов в секунду
	docker run --rm --network host -u $$(id -u):$$(id -g) -v $(CURDIR)/tests/load:/scripts grafana/k6 run /scripts/rps.js

.PHONY: coverage-branches
coverage-branches: ## Покрытие ветвей бэкенда (gobco)
	sh backend/scripts/branch-coverage.sh
