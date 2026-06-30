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
	docker compose up -d

docker-down:
	docker compose down

docker-build:
	docker build -t neuhis-backend .

# ==========================
# Smoke Test (Newman Black-Box)
# ==========================

smoke-test: ## 运行 Newman 黑盒测试 (默认 localhost:8080)
	@bash tests/newman/run-smoke.sh http://localhost:8080

smoke-test-remote: ## 运行 Newman 黑盒测试 (指定地址: make smoke-test-remote BASE_URL=http://host:8080)
	@bash tests/newman/run-smoke.sh $(BASE_URL)

smoke-test-docker: ## 启动 Docker 服务 → 运行黑盒测试 → 清理
	@docker compose up -d --build
	@sleep 8
	@bash tests/newman/run-smoke.sh http://localhost:8080; EXIT_CODE=$$?; \
	docker compose down; \
	exit $$EXIT_CODE

smoke-test-quick: ## 快速冒烟测试 (仅运行基础端点)
	@echo "Running quick smoke test..."
	@curl -sf http://localhost:8080/api/health && echo "✅ Health OK" || echo "❌ Health FAIL"
	@curl -sf -X POST http://localhost:8080/api/auth/register \
		-H "Content-Type: application/json" \
		-d '{"phone":"13800000001","password":"TestPass123!","realName":"测试"}' > /dev/null && echo "✅ Auth OK" || echo "⚠️ Auth (may already exist)"

# ==========================
# Admin Smoke Test (Newman Black-Box)
# ==========================

smoke-test-admin: ## 运行 Admin 黑盒测试 (默认 localhost:8080)
	@bash tests/newman/run-admin-smoke.sh http://localhost:8080

smoke-test-admin-remote: ## 运行 Admin 黑盒测试 (指定地址: make smoke-test-admin-remote BASE_URL=http://host:8080)
	@bash tests/newman/run-admin-smoke.sh $(BASE_URL)

smoke-test-admin-docker: ## 启动 Docker 服务 → 运行 Admin 黑盒测试 → 清理
	@docker compose up -d --build
	@sleep 8
	@bash tests/newman/run-admin-smoke.sh http://localhost:8080; EXIT_CODE=$$?; \
	docker compose down; \
	exit $$EXIT_CODE

# ==========================
# Combined Smoke Tests
# ==========================

smoke-test-all: ## 运行所有黑盒测试（患者端 + Admin，需先启动服务）
	@echo "=== Patient API Tests ==="
	@bash tests/newman/run-smoke.sh http://localhost:8080; PATIENT_EXIT=$$?; \
	echo "=== Restarting server to clear rate limits ==="; \
	kill $$(lsof -ti:8080 2>/dev/null) 2>/dev/null || true; \
	sleep 2; \
	go run ./cmd/server &>/tmp/server_restart.log & \
	sleep 4; \
	echo "=== Admin API Tests ==="; \
	bash tests/newman/run-admin-smoke.sh http://localhost:8080; ADMIN_EXIT=$$?; \
	if [ $$PATIENT_EXIT -ne 0 ] || [ $$ADMIN_EXIT -ne 0 ]; then \
		echo "❌ Some test suites failed (patient=$$PATIENT_EXIT, admin=$$ADMIN_EXIT)"; \
		exit 1; \
	fi; \
	echo "✅ All test suites passed"

smoke-test-all-docker: ## 启动 Docker 服务 → 运行所有黑盒测试 → 清理
	@docker compose up -d --build
	@sleep 8
	@bash tests/newman/run-smoke.sh http://localhost:8080; PATIENT_EXIT=$$?; \
	echo ""; \
	bash tests/newman/run-admin-smoke.sh http://localhost:8080; ADMIN_EXIT=$$?; \
	docker compose down; \
	if [ $$PATIENT_EXIT -ne 0 ] || [ $$ADMIN_EXIT -ne 0 ]; then \
		echo "❌ Some test suites failed (patient=$$PATIENT_EXIT, admin=$$ADMIN_EXIT)"; \
		exit 1; \
	fi; \
	echo "✅ All test suites passed"

# ==========================
# Clean
# ==========================

clean:
	rm -rf bin/ coverage.out coverage.html tests/newman/reports/
