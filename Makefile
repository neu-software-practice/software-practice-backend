.PHONY: test lint run build docker-up docker-down smoke-test migrate-up migrate-down clean

# ==========================
# Development
# ==========================

run:
	go run ./cmd/server

build:
	go build -o bin/server ./cmd/server

# ==========================
# Testing
# ==========================

test:
	go test -race -cover ./...

test-cover:
	go test -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

test-unit:
	go test -race -cover ./internal/model/... ./internal/config/... ./internal/middleware/... ./internal/adapter/... ./pkg/...

test-integration:
	go test -race -cover ./internal/repository/... ./internal/service/... ./internal/handler/...

# ==========================
# Code Quality
# ==========================

lint:
	golangci-lint run ./...

pre-commit:
	pre-commit run --all-files

# ==========================
# Database
# ==========================

migrate-up:
	migrate -path db/migrations -database "$(DATABASE_DSN)" up

migrate-down:
	migrate -path db/migrations -database "$(DATABASE_DSN)" down

# ==========================
# Docker
# ==========================

docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

docker-build:
	docker build -t neuhis-backend .

# ==========================
# Smoke Test
# ==========================

smoke-test:
	newman run tests/newman/neuhis-agent.postman_collection.json \
		-e tests/newman/neuhis-agent.postman_environment.json

# ==========================
# Clean
# ==========================

clean:
	rm -rf bin/ coverage.out coverage.html
