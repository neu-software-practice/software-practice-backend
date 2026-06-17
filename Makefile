.PHONY: help tidy build run migrate migrate-down seed fmt lint test cover swag docker-up docker-down test-mysql-up test-mysql-down

BINDIR ?= bin
COVER_THRESHOLD ?= 80

# Load .env for local development (silent fail when missing, e.g. CI).
-include .env
export TEST_DATABASE_DSN

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN{FS=":.*?## "}{printf "  \033[36m%-14s\033[0m %s\n",$$1,$$2}'

tidy: ## go mod tidy
	go mod tidy

build: ## Build the server binary
	go build -o $(BINDIR)/server ./cmd/server

run: ## Run the API server
	go run ./cmd/server

migrate: ## Apply database migrations
	go run ./cmd/migrate -dir up

migrate-down: ## Revert database migrations
	go run ./cmd/migrate -dir down

seed: ## Load demonstration base data
	go run ./cmd/seed

fmt: ## Format the code
	gofmt -w .

lint: ## Run golangci-lint
	golangci-lint run ./...

test: test-mysql-up ## Run all tests (starts test MySQL first)
	go test -v ./... -count=1

cover: test-mysql-up ## Run tests with the coverage gate
	go test -race ./... -coverpkg=./... -coverprofile=cover.out -covermode=atomic
	@bash scripts/coverage.sh cover.out $(COVER_THRESHOLD)

swag: ## Generate Swagger docs into internal/swagger
	swag init -g cmd/server/main.go -o internal/swagger --parseDependency --parseInternal

docker-up: ## Build & start the full stack (MySQL + backend)
	docker compose up -d --build

docker-down: ## Stop the stack and remove volumes
	docker compose down -v

test-mysql-up: ## Start the test-only MySQL container (port 3307)
	docker compose -f docker-compose.test.yml up -d --wait

test-mysql-down: ## Stop and remove the test MySQL container
	docker compose -f docker-compose.test.yml down -v

smoke-test: ## Run Newman smoke tests against the full Docker stack
	@test/smoke/run_smoke.sh

smoke-test-no-teardown: ## Run smoke tests and leave the stack running
	@test/smoke/run_smoke.sh --no-teardown
