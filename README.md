# NEUHIS Agent — 东软云脑智能医疗 后端服务

基于 **Golang Gin** 的单体业务后端，实现「AI + 诊疗」Agentic 聊天平台的全部 REST/SSE API，作为前端 React/TypeScript 应用与 [medAgent](./medAgent) AI 诊疗引擎之间的业务编排层。

---

## 技术栈

| 类别 | 选型 | 说明 |
|------|------|------|
| 语言 | Go 1.25 | 与 medAgent 子模块共用生态 |
| HTTP 框架 | [Gin](https://github.com/gin-gonic/gin) v1.9 | 中间件丰富，性能优异 |
| 数据库 | MySQL 8.0 | Docker 容器提供；CI 用 GitHub Services MySQL |
| 数据库驱动 | [go-sql-driver/mysql](https://github.com/go-sql-driver/mysql) v1.8 | 纯 Go 实现 |
| 迁移 | SQL 文件 + `golang-migrate` CLI | `db/migrations/` 目录下按序号管理 |
| 鉴权 | JWT HS256 | accessToken 15min + refreshToken 7天 (rotation) |
| 密码哈希 | bcrypt cost≥12 | `golang.org/x/crypto` |
| 配置 | `.env` + godotenv | 支持 `.env.local` 本地覆盖 |
| 容器化 | Docker 多阶段构建 + Docker Compose | 开发/CI/生产统一 |
| 测试 | `go test -race` + testcontainers-go + Newman | 三层：单元 → 集成 → 冒烟 |
| 静态分析 | golangci-lint (含 gosec) | v2 配置，8 个 linter |
| CI/CD | GitHub Actions | PR → main：test + lint + smoke；push main：deploy |

---

## 项目结构

```
software-practice-backend/
├── cmd/
│   ├── server/main.go          # 应用入口：配置 → DB → 路由 → 启动
│   └── jwtgen/main.go          # 辅助工具：生成冒烟测试用 JWT
├── internal/
│   ├── config/                 # .env 加载与校验 (JWT≥32B, CORS, 弱口令)
│   ├── model/                  # 领域实体、枚举、DTO、业务错误
│   ├── repository/             # 数据访问层 (Repository Pattern: 接口 + MySQL 实现)
│   ├── service/                # 业务逻辑层
│   │   ├── patient/            #   患者身份核验、上下文、资料更新
│   │   ├── visit/              #   会话生命周期 + 17态状态机
│   │   ├── workbench/          #   聊天编排、流程卡、支付、取药、治疗、退出
│   │   ├── medagent/           #   medAgent HTTP 客户端
│   │   ├── auth/               #   用户注册/登录/刷新/登出
│   │   ├── address/            #   收货地址 CRUD
│   │   ├── billing/            #   账单记录
│   │   ├── medicalorder/       #   医嘱查询
│   │   └── admin/              #   管理面板 (仪表盘、患者/会话管理、设置)
│   ├── handler/                # Gin HTTP 处理器 (路由注册 + 各域 handler)
│   ├── middleware/              # Gin 中间件 (auth, CORS, logging, recovery, rate-limit)
│   ├── adapter/                # medAgent Step → SSE/Card/Timeline 映射
│   ├── errors/                 # ApiError 结构体 + 错误码 + Gin 响应辅助
│   ├── llm/                    # OpenAI 兼容 LLM 客户端 (标题生成)
│   └── testutil/               # 共享 Mock (MockPatientRepo 等)
├── pkg/api/                    # 公开可复用包: ApiResponse[T], PageResult[T]
├── db/migrations/              # 13 对 up/down SQL 迁移文件
├── medAgent/                   # Git submodule: AI 诊疗引擎 (Go, 零外部依赖)
├── tests/
│   ├── testutil/               # testcontainers MySQL 容器 + per-test 临时数据库
│   ├── newman/                 # Postman 集合 (患者端 + Admin) + 运行脚本
│   └── seed/testdata.sql       # 测试种子数据
├── docs/
│   ├── STRUCTURE.md            # 项目架构设计文档
│   ├── PLAN.md                 # 分阶段实现计划
│   ├── SPEC.md                 # 项目目标与要求
│   └── front-api.md            # 前端 REST/SSE API 合约 (权威基线)
├── scripts/precommit-check.sh  # Pre-commit 覆盖率检查脚本
├── Makefile                    # 常用命令速查
├── Dockerfile                  # 多阶段构建
├── docker-compose.yml          # 全量容器编排 (Gin + MySQL + medAgent)
├── .golangci.yml               # golangci-lint 配置
├── .pre-commit-config.yaml     # Git pre-commit hooks
├── .env.example                # 配置模板
└── .github/workflows/
    ├── ci.yml                  # CI: test + lint + 集成 + 冒烟
    └── cd.yml                  # CD: push main → 自动部署
```

### 分层架构

```
HTTP Request
  → Gin Router
    → Middleware (recovery → logging → CORS → rate-limit → auth)
      → Handler (参数解析、调用 Service、序列化响应)
        → Service (业务逻辑编排、状态机、medAgent 适配)
          → Repository (数据持久化接口 → MySQL 实现)
            → MySQL
```

**依赖方向**: Handler → Service → Repository → MySQL。上层依赖接口，下层实现接口。测试时注入 Mock 实现。

---

## 快速开始

### 前置条件

| 工具 | 版本要求 | 用途 |
|------|----------|------|
| Go | ≥1.25 | 编译与运行 |
| Docker + Docker Compose | 最新稳定版 | 运行 MySQL、medAgent、冒烟测试 |
| Node.js | ≥20 (可选) | 仅冒烟测试 Newman 需要 |
| pre-commit | 最新版 (可选) | 本地代码质量门控 |

### 1. 克隆项目

```bash
git clone --recurse-submodules <repo-url>
cd software-practice-backend
```

### 2. 配置环境变量

```bash
cp .env.example .env
# 编辑 .env，至少修改:
#   JWT_SECRET=<32字节以上随机字符串>
#   MEDAGENT_API_KEY=<你的 LLM API Key>
```

> **注意**: JWT_SECRET 必须 ≥32 字节，且不能是弱口令（如 `123456...`、`changeme...`）。生产环境 `CORS_ALLOWED_ORIGINS` 不可为 `*`。

### 3. 启动服务 (Docker Compose，推荐)

```bash
# 启动全部服务 (Gin 后端 + MySQL + medAgent)
docker compose up -d

# 运行数据库迁移
for f in db/migrations/*.up.sql; do
  docker compose exec -T mysql mysql -u root -ppassword neuhis < "$f"
done

# 验证
curl http://localhost:8080/api/health
# → {"status":"ok"}

# 查看日志
docker compose logs -f app
```

### 4. 本地开发

```bash
# 安装 pre-commit hooks
pre-commit install

# 启动 MySQL (Docker)
docker compose up -d mysql

# 运行迁移
make migrate-up

# 启动开发服务器
go run ./cmd/server
# 或: make run
```

---

## 环境变量参考

| 变量 | 必填 | 默认值 | 说明 |
|------|------|--------|------|
| `SERVER_ADDR` | 否 | `:8080` | 监听地址 |
| `SERVER_MODE` | 否 | `debug` | Gin 模式: `debug` / `test` / `release` |
| `DATABASE_DSN` | **是** | — | MySQL 连接串 |
| `JWT_SECRET` | **是** | — | JWT 签名密钥，≥32 字节，弱口令拒绝 |
| `CORS_ALLOWED_ORIGINS` | 否 | `http://localhost:5173` | 允许的跨域来源；production 不可为 `*` |
| `MEDAGENT_MODE` | 否 | `http` | medAgent 集成: `http` (独立进程) / `embedded` (库) |
| `MEDAGENT_BASE_URL` | 否 | `http://medagent:8080` | medAgent 服务地址 |
| `MEDAGENT_API_KEY` | **是** | — | LLM API Key (DeepSeek/Qwen/OpenAI) |
| `MEDAGENT_PROVIDER` | 否 | `deepseek` | LLM Provider: `deepseek` / `qwen` / `openai` |
| `MEDAGENT_MODEL` | 否 | `deepseek-chat` | 模型名称 |
| `LOG_LEVEL` | 否 | `info` | 日志级别: `debug` / `info` / `warn` / `error` |

配置加载顺序: `.env.example` → `.env` → `.env.local` (后者覆盖前者)。`.env` 和 `.env.local` 已加入 `.gitignore`。

---

## API 端点概览

所有端点位于 `/api` 前缀下。完整字段级合约见 [`docs/front-api.md`](./docs/front-api.md)。

### 鉴权说明

系统采用 **JWT 双令牌认证**：

- **accessToken**: HS256 JWT，15 分钟有效，`Authorization: Bearer <token>` 传递
- **refreshToken**: 不透明字符串，7 天有效，单次使用后轮换 (rotation)
- 患者和管理员使用**两套独立的 JWT 系统** (不同密钥、不同端点)

### 端点总表 (49 个端点)

#### 健康检查

| 方法 | 路径 | 鉴权 | 说明 |
|------|------|------|------|
| GET | `/api/health` | 无 | 健康检查 |

#### 认证 (公开，限流 5 req/min/IP)

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/auth/register` | 注册，签发令牌对 |
| POST | `/api/auth/login` | 手机号+密码登录 |
| POST | `/api/auth/refresh` | 刷新 accessToken |
| POST | `/api/auth/logout` | 注销，使 refreshToken 失效 |

#### 患者域

| 方法 | 路径 | 鉴权 | 说明 |
|------|------|------|------|
| POST | `/api/patients/verify` | 无 | 身份核验 (身份证/手机号) |
| GET | `/api/patients/:patientId/context` | JWT | 问诊上下文 |
| PATCH | `/api/patients/:patientId/profile` | JWT | 更新过敏史/慢病/用药 |
| GET | `/api/patients/:patientId/addresses` | JWT | 地址列表 |
| POST | `/api/patients/:patientId/addresses` | JWT | 新增地址 |
| PATCH | `/api/patients/:patientId/addresses/:addressId` | JWT | 修改地址 |
| DELETE | `/api/patients/:patientId/addresses/:addressId` | JWT | 删除地址 |
| PUT | `/api/patients/:patientId/addresses/:addressId/default` | JWT | 设为默认地址 |

#### 会话域

| 方法 | 路径 | 鉴权 | 说明 |
|------|------|------|------|
| POST | `/api/visits` | JWT | 创建新出诊 |
| POST | `/api/visits/:sessionId/follow-up` | JWT | 创建复诊 |
| GET | `/api/visits` | JWT | 历史就诊列表 (cursor 分页) |
| GET | `/api/visits/:sessionId` | JWT | 会话详情 |
| GET | `/api/visits/:sessionId/snapshot` | JWT | 只读快照 (含完整时间线) |

#### 工作台域 (聊天 + 流程操作)

| 方法 | 路径 | 鉴权 | 说明 |
|------|------|------|------|
| GET | `/api/visits/:sessionId/timeline` | JWT | 时间线分页 |
| POST | `/api/visits/:sessionId/messages` | JWT | 发送患者消息 |
| POST | `/api/visits/:sessionId/assistant-stream` | JWT | **SSE** AI 流式回复 |
| POST | `/api/visits/:sessionId/lab-decision` | JWT | 检验决定 (accepted/skipped/vetoed) |
| POST | `/api/visits/:sessionId/payments` | JWT | 创建/确认支付 |
| POST | `/api/visits/:sessionId/fulfillment` | JWT | 取药/配送确认 |
| POST | `/api/visits/:sessionId/treatment-execution` | JWT | 治疗推进 |
| POST | `/api/visits/:sessionId/advice-ack` | JWT | 仅医嘱确认 |
| POST | `/api/visits/:sessionId/lock-question` | JWT | **SSE** 锁定态旁路问答 |
| POST | `/api/visits/:sessionId/classify-intent` | JWT | 完成态意图分类 |
| POST | `/api/visits/:sessionId/consult` | JWT | **SSE** 完成态咨询 |
| POST | `/api/visits/:sessionId/vitals` | JWT | 体征上报与急症复检 |
| POST | `/api/visits/:sessionId/exit` | JWT | 主动退出结算 (四档后果) |
| POST | `/api/visits/:sessionId/timer` | JWT | 暂停/恢复总计时 |
| POST | `/api/visits/:sessionId/dismiss-emergency` | JWT | 解除急症态 |
| POST | `/api/visits/:sessionId/generate-title` | JWT | LLM 生成会话标题 |

#### 账单与医嘱

| 方法 | 路径 | 鉴权 | 说明 |
|------|------|------|------|
| GET | `/api/billing/records` | JWT | 历史账单汇总 |
| GET | `/api/medical-orders` | JWT | 医嘱记录 |

#### 管理员 (独立 JWT, `/admin` 前缀)

| 方法 | 路径 | 鉴权 | 说明 |
|------|------|------|------|
| POST | `/admin/auth/login` | 无 | 管理员登录 |
| POST | `/admin/auth/logout` | Admin JWT | 注销 |
| POST | `/admin/auth/refresh` | 无 | 刷新管理员令牌 |
| GET | `/admin/dashboard/stats` | Admin JWT | 仪表盘统计 |
| GET | `/admin/patients` | Admin JWT | 患者列表 |
| GET | `/admin/patients/:id` | Admin JWT | 患者详情 |
| GET | `/admin/sessions` | Admin JWT | 会话列表 |
| GET | `/admin/sessions/:id` | Admin JWT | 会话详情 |
| GET | `/admin/settings` | Admin JWT | 系统设置 |
| PUT | `/admin/settings` | Admin JWT | 更新系统设置 |

### 统一响应格式

所有非流式响应:

```json
{
  "success": true,
  "data": { ... },
  "error": null,
  "meta": { "total": 100, "pageSize": 20 }
}
```

错误响应:

```json
{
  "success": false,
  "data": null,
  "error": {
    "code": "SESSION_NOT_FOUND",
    "message": "找不到这次就诊记录",
    "status": 404,
    "retriable": false
  }
}
```

### 分页

游标式分页 (cursor-based):

```json
{
  "items": [ ... ],
  "nextCursor": "opaque-cursor",
  "hasMore": true
}
```

### SSE 流式响应

`Content-Type: text/event-stream`，每行 `data: <JSON>\n\n`。事件类型: `delta` (文本增量)、`message_final` (消息完成)、`card` (流程卡)、`state` (状态变更)、`emergency` (急症)、`done` (流结束)、`error` (错误)。

---

## 开发指南

### 添加新 API 端点

遵循分层架构，自底向上：

1. **model** — 定义请求/响应结构体、枚举、错误
2. **repository** — 定义接口 (`*_repo.go`) + MySQL 实现 (`*_mysql.go`)
3. **service** — 实现业务逻辑，依赖 repository 接口
4. **handler** — Gin handler: `BindJSON` → 调用 service → `WriteSuccess`/`WriteError`
5. **router** — 在 `SetupRoutes()` 中注册路由

原则:
- **接受接口，返回结构体** — 依赖注入通过构造函数
- **接口要小** — Repository 接口保持 1-3 个方法
- **错误要包裹** — `fmt.Errorf("...: %w", err)`
- **SQL 参数化** — 所有查询使用 `?` 占位符

### 数据库迁移

```bash
# 运行迁移
make migrate-up
# 回滚迁移
make migrate-down

# 或在 Docker 中
docker compose exec -T mysql mysql -u root -ppassword neuhis < db/migrations/00000x_xxx.up.sql
```

### 分层约定

| 层 | 目录 | 职责 | 不可做的事 |
|----|------|------|------------|
| **Model** | `internal/model/` | 数据结构、枚举、sentinel error | 依赖任何其他层 |
| **Repository** | `internal/repository/` | 数据持久化 | 包含业务逻辑 |
| **Service** | `internal/service/` | 业务规则、状态机、编排 | 直接操作 HTTP 请求 |
| **Handler** | `internal/handler/` | HTTP 解析、调用 Service、序列化 | 包含业务逻辑 |
| **Adapter** | `internal/adapter/` | medAgent Step → 前端事件映射 | 依赖 medAgent 之外的业务逻辑 |
| **Middleware** | `internal/middleware/` | 横切关注点 (auth, CORS, 日志) | 修改业务数据 |

---

## 测试

### 测试分层

| 层级 | 命令 | 工具 | 覆盖目标 |
|------|------|------|----------|
| 单元测试 | `make test-unit` | `go test` + Mock | ≥90% |
| 集成测试 | `make test-integration` | testcontainers MySQL | ≥90% (服务层) |
| 全量测试 | `make test` | `go test -race -cover ./...` | 总覆盖率 ≥70% |
| 冒烟测试 | `make smoke-test` | Newman + Docker Compose | 核心流程 |

### 运行测试

```bash
# 全量测试 (带竞态检测)
make test

# 仅单元测试
make test-unit

# 集成测试 (自动拉起 Docker MySQL 容器)
make test-integration

# 覆盖率报告
make test-cover
# → 生成 coverage.html

# 冒烟测试 (需先启动服务)
docker compose up -d
make smoke-test        # 患者端
make smoke-test-admin  # Admin 端
make smoke-test-all    # 两者都跑

# Docker 中一键冒烟 (启动 → 测试 → 清理)
make smoke-test-docker
```

### 覆盖率门控

| 范围 | 阈值 | 强制执行 |
|------|------|----------|
| 服务层 (patient + visit + workbench) | ≥90% | Pre-commit + CI |
| 总覆盖率 (单元测试) | ≥70% | CI |
| 总覆盖率 (pre-commit 本地) | ≥75% | Pre-commit |

### 集成测试数据库

集成测试使用 **testcontainers-go** 为每个测试创建临时 MySQL 数据库：

- `DBTest` — 在同一容器中按测试名创建独立数据库，测试结束自动删除
- `SetupMySQL` — 每个测试启动独立 Docker 容器，完全隔离

```go
func TestMyRepo(t *testing.T) {
    dbt := testutil.NewDBTest(t, dsn, "../../db/migrations")
    defer dbt.Close()
    repo := NewMySQLRepo(dbt.DB)
    // ...
}
```

CI 中集成测试使用 GitHub Services MySQL 容器，以 `neuhis_test` 数据库运行。

### 冒烟测试 (Newman/Postman)

两套 Postman 集合位于 `tests/newman/`：

- `neuhis-agent.postman_collection.json` — 患者端核心流程 (150KB)
- `admin.postman_collection.json` — Admin 端 (62KB)

预请求脚本自动管理 JWT 令牌 (登录 → 缓存 → 过期前自动刷新)，支持无头运行。

环境变量 `BAIL=1` 可在首次失败时退出；`SKIP_SSE=1` 跳过 SSE 测试。

---

## 代码质量

```bash
# 静态分析
make lint
# → golangci-lint run ./... (errcheck, govet, gosec, staticcheck 等 8 个 linter)

# Pre-commit hooks
pre-commit install          # 安装
make pre-commit             # 手动全量运行
# → Go Test Cover (≥90% 服务层) + golangci-lint

# 安全扫描
gosec ./...
```

### golangci-lint 启用的 Linter

`errcheck`, `govet`, `ineffassign`, `staticcheck`, `unused`, `gosec`, `misspell`, `unconvert`。格式化器: `gofmt` + `goimports`。测试文件和 `tests/` 目录排除 gosec。

---

## CI/CD

### CI — Pull Request → main

```
Lint & Test (MySQL service)
  → golangci-lint
  → go test -race -cover (单元 + 部分集成)
  → 检查服务层覆盖率 ≥90% + 总覆盖率 ≥70%

Integration Tests
  → go test ./internal/repository/... (testcontainers MySQL, 超时 10min)

Smoke Test (Docker Compose + Newman)
  → docker compose up -d --build
  → 运行迁移 → Newman 患者端测试 → Newman Admin 测试
  → docker compose down
```

### CD — Push main → 自动部署

```yaml
触发: push main
运行环境: self-hosted runner
流程: git pull → docker compose up -d --build app → 迁移 → 健康检查
```

---

## medAgent 集成

medAgent 是本项目的 **AI 诊疗引擎**（Git submodule，Go 1.22，零外部依赖）。负责 LLM 驱动的问诊决策，通过 HTTP API 对外暴露。

### 集成模式

当前使用**独立进程模式**（推荐）：medAgent 作为独立 HTTP 服务运行，后端通过 `MedAgentClient`（`internal/service/medagent/client.go`）调用。

```
Gin Backend                    medAgent (独立进程)
    │                                │
    │ POST /sessions                 │
    │ ─────────────────────────────> │  初始化会话
    │                                │
    │ POST /sessions/{id}/patient-say│
    │ ─────────────────────────────> │  患者发言 → AI 分析
    │ <───────────────────────────── │  Step{kind: "ASK"/"NEED_TESTS"/...}
    │                                │
    │ ... (循环: 检验→回填→用药→购药) │
    │                                │
    │ GET /sessions/{id}/record      │
    │ ─────────────────────────────> │  导出会话纪要
    │ DELETE /sessions/{id}          │
    │ ─────────────────────────────> │  销毁会话
```

### Step.kind → 前端 SSE 映射

| medAgent Step | 后端动作 | SSE 事件 | 前端表现 |
|---------------|----------|----------|----------|
| `ASK` | 透传 `doctor_say` | `delta` × n + `message_final` | AI 追问气泡 |
| `NEED_TESTS` | 构造检验卡 | `card(lab_decision)` + `state` | 是否检验阻塞卡 |
| `DRUG_QUERY` | 查药品规格 (后台) | `state` | 系统事件 |
| `PURCHASE` | 构造取药卡 | `card(medication_fulfillment)` + `state` | 购药确认卡 |
| `EMERGENCY` | 立即终止会话 | `emergency` | 急症 Overlay |
| `DONE` | 构造诊断+完成卡 | `card(diagnosis)` + `card(...)` + `done` | 诊断卡 + 完成卡 |

详细接入文档见 [`medAgent/docs/后端接入指南.md`](./medAgent/docs/后端接入指南.md)。

---

## 前端接口合约

[`docs/front-api.md`](./docs/front-api.md) 是前后端交互的**权威基线**，由前端 TypeScript Zod schema 自动梳理而成。包含:

- 全部端点请求/响应字段、枚举取值
- SSE 事件结构与示例序列
- FlowCard 9 种类型的完整字段定义
- 典型时序: 新建出诊 → AI 追问 → 检验 → 缴费 → 诊断 → 用药 → 完成
- 急症、超时、退出结算、复诊流程
- medAgent Step 映射边界

后端实现、联调与验收均以此文档为基准。

---

## 故障排除

### WSL2 环境下 Docker MySQL 连接超时

**现象**: 宿主机 `127.0.0.1:3306` 连接 Docker MySQL 卡死。

**原因**: WSL2 的 iptables 规则 `OUTPUT DROP` 丢弃了 `172.16.0.0/12` 网段（含 Docker 网络）的报文。

**解决方案**:
```bash
# 方案 1: 放行 Docker 网段
sudo iptables -I OUTPUT -d 172.16.0.0/12 -j ACCEPT

# 方案 2: 直接在 Docker 网络内访问
# 用 docker inspect 获取 MySQL 容器的网关 IP，替换 DSN 中的 host

# 方案 3: 通过 eth0 IP 连接
DSN=root:password@tcp($(hostname -I | awk '{print $1}'):3306)/neuhis?...
```

### JWT_SECRET 校验失败

```
JWT_SECRET must be at least 32 bytes
JWT_SECRET is a weak password
```

确保 `.env` 中的 `JWT_SECRET`:
- 长度 ≥32 字节
- 不是 `123456...`、`changeme...`、`secret...`、`password...` 等弱口令

### testcontainers 需要 Docker 环境

```bash
# 确保 Docker 可用
docker info

# 如果不想跑集成测试，跳过
go test -short ./...
```

### 常见错误码

| code | HTTP | 含义 | 处理 |
|------|------|------|------|
| `AUTH_TOKEN_EXPIRED` | 401 | accessToken 过期 | 调用 `/auth/refresh` |
| `AUTH_REFRESH_INVALID` | 401 | refreshToken 无效 | 重新登录 |
| `SESSION_NOT_FOUND` | 404 | 会话不存在 | 检查 sessionId |
| `CARD_NOT_FOUND` | 404 | 流程卡已失效 | 刷新重试 |
| `RATE_LIMITED` | 429 | 请求过于频繁 | 稍后重试 |
| `LLM_UNAVAILABLE` | 503 | 大模型不可用 | 降级或稍后重试 |

### Docker Compose 服务无法启动

```bash
# 查看全部日志
docker compose logs

# 重建
docker compose up -d --build

# 重置数据库
docker compose down -v   # 删除数据卷
docker compose up -d
```

---

## 参考文档

| 文档 | 说明 |
|------|------|
| [`docs/STRUCTURE.md`](./docs/STRUCTURE.md) | 项目架构设计、技术栈、分层详情 |
| [`docs/PLAN.md`](./docs/PLAN.md) | 分阶段实现计划与进度 |
| [`docs/front-api.md`](./docs/front-api.md) | 前端 REST/SSE API 合约 (权威基线) |
| [`medAgent/docs/后端接入指南.md`](./medAgent/docs/后端接入指南.md) | medAgent 接入指南 |
| [`CLAUDE.md`](./CLAUDE.md) | Claude Code AI 助手指令 |

---

## 关联仓库

| 组件 | 仓库地址 |
|------|----------|
| **前端** | [neuhis-agent-front](https://github.com/neu-software-practice/neuhis-agent-front) |
| **后端** | [software-practice-backend](https://github.com/neu-software-practice/software-practice-backend)（本仓库） |
| **AI 智能体** | [medAgent](https://github.com/neu-software-practice/medAgent) |
