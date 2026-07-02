# 项目脚本说明

本目录主要维护 API 契约漂移检测和本地质量门控脚本。推荐优先通过根目录 `Makefile` 调用，只有排查问题时再直接运行单个脚本。

## 推荐入口

```bash
make help
make quality
make test-cover-gate
make api-drift
make api-drift-loop FRONTEND_DIR=../neuhis-agent-front
```

## 质量门控

| 脚本 | 用途 | 推荐入口 |
|------|------|----------|
| `precommit-check.sh` | 运行 Go 测试并检查覆盖率；服务层重点包目标为 >=90%，总覆盖率沿用脚本当前阈值 | `make test-cover-gate` |

`.pre-commit-config.yaml` 已调用 `bash scripts/precommit-check.sh`，并保留 `golangci-lint run ./...` 作为静态检查门控。

覆盖率脚本支持通过环境变量临时调整阈值和输出路径：`SERVICE_COVERAGE_THRESHOLD`、`TOTAL_COVERAGE_THRESHOLD`、`COVER_PROFILE`。

## API 漂移检测

API 漂移检测用于对齐前端 Zod/接口契约与后端 Gin/Go struct 实现。前端仓库默认位于 `../neuhis-agent-front`，字段和端点对比依赖前端仓库中已生成的 `api-contract.json` 与 `frontend-fields.json`。

### 后端提取

| 脚本 | 输入 | 输出 | 推荐入口 |
|------|------|------|----------|
| `extract-backend-api.mjs` | `internal/handler/router.go` 及路由注册代码 | `backend-api.json` | `make api-extract-backend` |
| `extract-go-fields.mjs` | `internal/model/`、`internal/handler/` 等 Go struct | `backend-fields.json` | `make api-extract-backend` |

### 前端提取

| 脚本 | 输入 | 输出 | 说明 |
|------|------|------|------|
| `extract-frontend-api.mjs` | 前端 `src/` 下 API facade 与 schema | `api-contract.json` | 该脚本按自身所在仓库的 `src/` 解析，通常应在前端仓库运行 |
| `extract-frontend-fields.mjs` | 前端 `src/` 下 Zod schema | `frontend-fields.json` | 旧字段提取器，保留用于兼容 |
| `extract-zod-fields.mjs` | 前端 `src/` 下 Zod schema | `frontend-fields.json` | 推荐字段提取器；`fix-drift-loop.sh` 默认调用前端仓库中的同名脚本 |

如果前端仓库没有生成产物，先在前端仓库运行对应提取脚本，或执行：

```bash
make api-drift-loop FRONTEND_DIR=/path/to/neuhis-agent-front
```

### 对比报告

| 脚本 | 输入 | 输出 | 推荐入口 |
|------|------|------|----------|
| `compare-api.mjs` | `../neuhis-agent-front/api-contract.json` + `backend-api.json` | `drift-report.json` | `make api-drift-endpoints` |
| `compare-fields.mjs` | `../neuhis-agent-front/frontend-fields.json` + `backend-fields.json` + `doc-snapshot.json` | `drift-report-fields.json` | `make api-drift-fields` |
| `compare-request.mjs` | `../neuhis-agent-front/frontend-fields.json` + `backend-fields.json` | `drift-report-request.json` | `make api-drift-request` |

`make api-drift` 会先刷新后端快照，再运行端点、响应字段、请求字段三类对比。报告文件为排查产物，不代表自动修复。

## 编排脚本

| 脚本 | 用途 |
|------|------|
| `fix-drift-loop.sh` | 编排前端字段提取、后端字段提取、字段/请求漂移对比；发现漂移后暂停并提示人工或 agent 修复 |

`fix-drift-loop.sh` 支持通过 `FRONTEND_DIR` 覆盖前端仓库路径：

```bash
FRONTEND_DIR=/path/to/neuhis-agent-front bash scripts/fix-drift-loop.sh
```

## 常见产物

| 文件 | 说明 |
|------|------|
| `backend-api.json` | 后端路由快照 |
| `backend-fields.json` | 后端 Go struct 字段快照 |
| `drift-report.json` | 端点级漂移报告 |
| `drift-report-fields.json` | 响应字段级漂移报告 |
| `drift-report-request.json` | 请求体和查询参数漂移报告 |
| `coverage.out`、`coverage.html` | Go 覆盖率产物 |

可用 `make clean` 清理本地构建、覆盖率和 Newman 报告产物。
