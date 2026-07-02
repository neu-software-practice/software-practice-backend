.DEFAULT_GOAL := help

BASE_URL ?= http://localhost:8080
FRONTEND_DIR ?= ../neuhis-agent-front

.PHONY: help run build test test-cover test-unit test-integration test-cover-gate lint quality pre-commit \
	api-extract-backend api-drift-endpoints api-drift-fields api-drift-request api-drift api-drift-loop \
	migrate-up migrate-down docker-up docker-down docker-build \
	smoke-test smoke-test-remote smoke-test-docker smoke-test-quick \
	smoke-test-admin smoke-test-admin-remote smoke-test-admin-docker \
	smoke-test-all smoke-test-all-docker clean

help: ## 显示可用命令
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make <target>\n"} /^[a-zA-Z0-9_-]+:.*##/ {printf "  \033[36m%-24s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# ==========================
# Development
# ==========================

run: ## 启动本地开发服务器
	go run ./cmd/server

build: ## 编译后端服务到 bin/server
	go build -o bin/server ./cmd/server

# ==========================
# Testing
# ==========================

test: ## 运行全部 Go 测试（race + cover）
	go test -race -cover ./...

test-cover: ## 生成 coverage.out 和 coverage.html
	go test -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

test-unit: ## 运行轻量单元测试包
	go test -race -cover ./internal/model/... ./internal/config/... ./internal/middleware/... ./internal/adapter/... ./pkg/...

test-integration: ## 运行 repository/service/handler 集成相关测试
	go test -race -cover ./internal/repository/... ./internal/service/... ./internal/handler/...

test-cover-gate: ## 运行本地覆盖率门控（服务层重点包 >=90%）
	bash scripts/precommit-check.sh

# ==========================
# Code Quality
# ==========================

lint: ## 运行 golangci-lint
	golangci-lint run ./...

quality: lint test-cover-gate ## 运行本地质量门控（lint + coverage gate）

pre-commit: ## 运行所有 pre-commit hooks
	pre-commit run --all-files

# ==========================
# API Drift
# ==========================

api-extract-backend: ## 提取后端路由和 Go 字段快照
	node scripts/extract-backend-api.mjs
	node scripts/extract-go-fields.mjs

api-drift-endpoints: api-extract-backend ## 对比前后端端点契约
	node scripts/compare-api.mjs

api-drift-fields: api-extract-backend ## 对比响应字段契约
	node scripts/compare-fields.mjs

api-drift-request: api-extract-backend ## 对比请求体和查询参数契约
	node scripts/compare-request.mjs

api-drift: api-drift-endpoints api-drift-fields api-drift-request ## 运行全部 API 漂移检测（需要前端产物）

api-drift-loop: ## 运行 API 漂移检测编排脚本
	FRONTEND_DIR="$(FRONTEND_DIR)" bash scripts/fix-drift-loop.sh

# ==========================
# Database
# ==========================

migrate-up: ## 应用数据库迁移（需要 DATABASE_DSN）
	migrate -path db/migrations -database "$(DATABASE_DSN)" up

migrate-down: ## 回滚数据库迁移（需要 DATABASE_DSN）
	migrate -path db/migrations -database "$(DATABASE_DSN)" down

# ==========================
# Docker
# ==========================

docker-up: ## 启动 docker compose 服务
	docker compose up -d

docker-down: ## 停止 docker compose 服务
	docker compose down

docker-build: ## 构建后端 Docker 镜像
	docker build -t neuhis-backend .

# ==========================
# Smoke Test (Newman Black-Box)
# ==========================

smoke-test: ## 运行 Newman 黑盒测试（默认 BASE_URL=http://localhost:8080）
	@bash tests/newman/run-smoke.sh $(BASE_URL)

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
		-d '{"phone":"13800000001","password":"TestPass123!","realName":"测试","gender":"unknown","birthDate":"1990-01-01"}' > /dev/null && echo "✅ Auth OK" || echo "⚠️ Auth (may already exist)"

# ==========================
# Admin Smoke Test (Newman Black-Box)
# ==========================

smoke-test-admin: ## 运行 Admin 黑盒测试（默认 BASE_URL=http://localhost:8080）
	@bash tests/newman/run-admin-smoke.sh $(BASE_URL)

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

clean: ## 清理本地构建、覆盖率和 Newman 报告产物
	rm -rf bin/ coverage.out coverage.html tests/newman/reports/
