.PHONY: help tidy build run migrate migrate-down seed fmt lint test cover swag docker-up docker-down

BINDIR ?= bin
COVER_THRESHOLD ?= 80

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

test: ## Run all tests
	go test ./... -count=1

cover: ## Run tests with the coverage gate
	go test -race ./... -coverpkg=./... -coverprofile=cover.out -covermode=atomic
	@bash scripts/coverage.sh cover.out $(COVER_THRESHOLD)

swag: ## Generate Swagger docs into internal/swagger
	swag init -g cmd/server/main.go -o internal/swagger --parseDependency --parseInternal

docker-up: ## Build & start the full stack (MySQL + backend)
	docker compose up -d --build

docker-down: ## Stop the stack and remove volumes
	docker compose down -v
