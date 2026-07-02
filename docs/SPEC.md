# 项目规范

## 项目定位

本项目是前端先行的 Golang Gin 单体业务后端，用于实现东软云医院传统业务逻辑，并作为前端 React/TypeScript 应用与 `./medAgent` 诊疗模块之间的业务编排层。

后端实现以接口契约为准，优先保证前端可用性、业务可靠性、可测试性和可维护性。

## 权威输入

| 文档/模块 | 作用 |
|-----------|------|
| `docs/rest-api.md` | 前端 REST/SSE API 合约的权威基线 |
| `docs/rest-api-patch-v*.md` | 版本化 API 变更补丁 |
| `docs/STRUCTURE.md` | 项目总体架构设计 |
| `docs/PLAN.md` | 分阶段实现计划 |
| `medAgent/docs/后端接入指南.md` | medAgent 诊疗模块接入指南 |

接口实现存在冲突时，以 `docs/rest-api.md` 加最新 `rest-api-patch-v*.md` 为准；API 漂移检测报告用于辅助发现实现偏差。

## 本地开发入口

根目录 `Makefile` 是本地开发和校验的统一入口：

```bash
make help
make run
make build
make test
make quality
make api-drift
make smoke-test
```

常用目标：

| 目标 | 说明 |
|------|------|
| `make run` | 启动本地 Gin 服务 |
| `make build` | 编译 `bin/server` |
| `make test` | 运行全部 Go 测试（race + cover） |
| `make test-cover` | 生成 `coverage.out` 和 `coverage.html` |
| `make test-cover-gate` | 运行本地覆盖率门控 |
| `make lint` | 运行 `golangci-lint run ./...` |
| `make quality` | 运行 lint 与覆盖率门控 |
| `make pre-commit` | 运行全部 pre-commit hooks |
| `make api-drift` | 运行 API 契约漂移检测 |

## 质量门控

项目要求通过 Go 测试覆盖率与静态检查门控：

- Go 测试使用 `go test` 与 coverage 产物。
- 本地覆盖率门控由 `scripts/precommit-check.sh` 执行。
- 服务层重点包覆盖率目标为 >=90%。
- 总覆盖率阈值沿用 `scripts/precommit-check.sh` 当前设置。
- 静态检查使用 `golangci-lint run ./...`。
- `.pre-commit-config.yaml` 同时执行覆盖率门控和 `golangci-lint`。

当前本地门控命令：

```bash
make test-cover-gate
make lint
make quality
pre-commit run --all-files
```

GitHub Actions 的 CI 配置独立维护在 `.github/workflows/ci.yml`。如需调整 CI 阈值，应同步修改 CI 与本地脚本，避免本地和远端门控长期分叉。

## API 漂移检测

脚本位于 `scripts/`，详细说明见 `scripts/README.md`。

默认约定：

- 前端仓库路径为 `../neuhis-agent-front`。
- 前端提取产物为 `api-contract.json` 和 `frontend-fields.json`。
- 后端提取产物为 `backend-api.json` 和 `backend-fields.json`。
- 对比报告为 `drift-report.json`、`drift-report-fields.json`、`drift-report-request.json`。

推荐入口：

```bash
make api-extract-backend
make api-drift-endpoints
make api-drift-fields
make api-drift-request
make api-drift
```

需要指定前端仓库路径时：

```bash
make api-drift-loop FRONTEND_DIR=/path/to/neuhis-agent-front
```

## 实现原则

- Handler 负责参数解析、鉴权上下文读取、调用 Service 和响应序列化。
- Service 负责业务编排、状态机、幂等性和 medAgent 适配。
- Repository 负责持久化，业务层依赖接口而非 MySQL 细节。
- API 响应字段、请求字段、错误码和 SSE 事件必须与权威接口文档保持一致。
- 新增或修改接口时，应同步补充测试，并在必要时刷新 API 漂移检测产物。

## 验收标准

一次后端功能或契约变更完成前，至少应满足：

- `make test` 或更聚焦的相关 Go 测试通过。
- `make lint` 通过。
- `make test-cover-gate` 通过，或明确记录覆盖率缺口与原因。
- 涉及前端接口契约时，运行相应 `make api-drift-*` 目标并处理报告。
- 涉及用户流程时，运行相关 Newman 冒烟测试。
