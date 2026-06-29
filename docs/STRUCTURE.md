# STRUCTURE.md — 软件总体架构

> 本文档描述 NEUHIS Agent（东软云脑智能医疗）后端服务的整体架构，包括技术栈、依赖关系、分层设计与包结构。与 `SPEC.md`（项目目标与要求）、`front-api.md`（REST/SSE API 合约）配合阅读。

---

## 1. 项目概述

本项目是前端先行项目「NEUHIS Agent」的 **Golang gin单体业务后端**，承担东软云医院传统业务逻辑的 Gin HTTP 实现。核心职责：

- 实现 `docs/front-api.md` 定义的全部 REST/SSE endpoint
- 作为业务编排层，衔接患者身份、就诊会话生命周期、流程卡交互、支付/取药/治疗执行等医院域逻辑
- 对接 `medAgent` AI 诊疗引擎，将 medAgent 的 `Step` 指令转译为前端可消费的 SSE 事件与时间线条目

**产品名**：东软云脑智能医疗  
**前端合约基线**：`docs/front-api.md`（由前端 Zod schema 自动梳理，已通过 mock/契约测试）

---

## 2. 技术栈

### 2.1 语言与运行时

| 项目 | 选型 | 说明 |
| --- | --- | --- |
| 语言 | Go 1.22+ | 与 medAgent 模块一致 |
| HTTP 框架 | [Gin](https://github.com/gin-gonic/gin) | 高性能 HTTP router，中间件生态丰富 |
| 数据库 | MySQL 8.0 | 经 Docker 提供，所有集成测试使用 per-test 临时数据库 |
| 数据库驱动 | [go-sql-driver/mysql](https://github.com/go-sql-driver/mysql) | 纯 Go MySQL 驱动 |
| 迁移工具 | [golang-migrate](https://github.com/golang-migrate/migrate) | 数据库 schema 版本管理 |
| 鉴权 | JWT | `JWT_SECRET` 经 `.env` 注入，≥32 字节，含弱口令黑名单校验 |
| 配置 | `.env` 文件 + `config.Load()` | 启动时解析 `.env`，校验必填项与约束；支持 `.env.local` 本地覆盖 |

### 2.2 基础设施与 DevOps

| 项目 | 选型 | 说明 |
| --- | --- | --- |
| 测试框架 | `go test`（标准库） | 表驱动测试 + `-race` 竞态检测 |
| 断言库 | 标准库 `testing` + `reflect`/`cmp` | 无第三方断言依赖 |
| Mock 生成 | 手写 interface mock | 遵循「接受接口，返回结构体」原则 |
| 静态分析 | `golangci-lint` | 含 `gosec` 安全扫描 |
| 测试覆盖率 | `go test -cover` | 门控 ≥90% |
| Pre-commit | GitHub `pre-commit` hooks | `.pre-commit-config.yaml`：go test cover ≥90% + golangci-lint 均通过 |
| 集成测试 | [testcontainers-go](https://github.com/testcontainers/testcontainers-go) | 代码内通过 testcontainers 拉起 Docker MySQL 容器，per-test 临时数据库 |
| 冒烟测试 | Docker Compose + [Newman](https://github.com/postmanlabs/newman) | `docker-compose up` 全量起容器（Gin + MySQL + medAgent），Newman 跑 Postman Collection 集成脚本 |
| CI/CD | GitHub Actions | pre-commit hook + testcontainers 集成测试 + docker-compose up + Newman 冒烟测试 |
| 容器化 | Docker（多阶段构建）/ Docker Compose | Gin 服务与 MySQL 均 Docker 部署；本地开发与 CI 统一容器环境 |

### 2.3 medAgent 诊疗引擎（子模块）

| 项目 | 说明 |
| --- | --- |
| 路径 | `./medAgent`（Git submodule） |
| 语言 | Go 1.22，**零外部依赖**（仅标准库） |
| 运行模式 | 独立进程（`cmd/server`）或嵌入式库（`agent.New()`） |
| LLM Provider | DeepSeek / Qwen / OpenAI（可切换） |
| 接入文档 | `medAgent/docs/后端接入指南.md` |
| 核心接口 | `POST /sessions` → `POST /patient-say` → `POST /test-results` → `POST /drug-info` → `POST /purchase-result` → `GET /record` → `DELETE` |
| Step 指令 | `ASK` / `NEED_TESTS` / `DRUG_QUERY` / `PURCHASE` / `EMERGENCY` / `DONE` / `OK` |

---

## 3. 系统架构（分层设计）

```
┌──────────────────────────────────────────────────────────────┐
│                      Frontend (React/TS)                      │
│              baseUrl: /api   ·   REST + SSE                   │
└──────────────────────────┬───────────────────────────────────┘
                           │
┌──────────────────────────▼───────────────────────────────────┐
│              Docker Network (docker-compose)                   │
│  ┌─────────────────────────────┐  ┌─────────────────────────┐ │
│  │   Gin Server Container      │  │  medAgent Container      │ │
│  │   Transport Layer            │  │  (独立进程)              │ │
│  │   Gin Router → Middleware    │  │  POST /sessions ...      │ │
│  │   → Handler → Service        │  │  LLM Provider API        │ │
│  │   → Repository               │  └─────────────────────────┘ │
│  └──────────┬──────────────────┘                               │
│             │                                                  │
│  ┌──────────▼──────────────────┐                               │
│  │   MySQL 8.0 Container       │                               │
│  │  · patients  · visits       │                               │
│  │  · timeline_items           │                               │
│  │  · flow_cards               │                               │
│  └─────────────────────────────┘                               │
└──────────────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────────────┐
│                    Integration Test                            │
│  go test → testcontainers-go → 拉起临时 Docker MySQL 容器     │
│  per-test 独立数据库，测试结束自动销毁容器                       │
└──────────────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────────────┐
│                    Smoke / E2E Test                            │
│  docker-compose up -d (全量容器) → Newman run collection       │
│  → docker-compose down (清理)                                  │
└──────────────────────────────────────────────────────────────┘
```

### 3.1 分层职责

| 层 | 包路径 | 职责 | 依赖方向 |
| --- | --- | --- | --- |
| **Transport** | `internal/handler/` | 解析 HTTP 请求，调用 Service，序列化响应；SSE 流管理 | → Service |
| **Service** | `internal/service/` | 业务逻辑编排、状态机、流程卡生命周期、medAgent 适配 | → Repository, medAgent |
| **Repository** | `internal/repository/` | 数据持久化接口与 MySQL 实现（Repository Pattern） | → MySQL |
| **Model** | `internal/model/` | 领域实体、枚举、DTO 定义 | 无外部依赖 |
| **Adapter** | `internal/adapter/` | medAgent Step ↔ SSE 事件 / FlowCard / TimelineItem 映射 | → medAgent, Model |
| **Middleware** | `internal/middleware/` | Gin 中间件（auth、CORS、logging、recovery、rate-limit） | → Config |

---

## 4. 包结构

```
software-practice-backend/
│
├── cmd/
│   └── server/
│       └── main.go                  # 入口：加载配置、初始化 DB、注册路由、启动 Gin
│
├── internal/                        # 私有应用代码（不可外部导入）
│   ├── config/
│   │   ├── config.go               # Config 结构体 + Load()：读取 .env 并校验
│   │   ├── env.go                   # .env 文件解析（支持 .env.local 覆盖）
│   │   └── config_test.go
│   │
│   ├── model/                       # 领域模型（纯数据结构，零依赖）
│   │   ├── patient.go              # PatientProfile, PatientContext, PriorVisit
│   │   ├── visit.go                # VisitSession, VisitSummary, VisitStatus 枚举
│   │   ├── timeline.go             # TimelineItem 判别联合（message/flow_card/system_event/terminal）
│   │   ├── flow_card.go            # FlowCard 各类型 + FlowCardStatus 枚举
│   │   ├── sse.go                  # AssistantStreamEvent 各类型
│   │   ├── payment.go              # PaymentStatus 枚举
│   │   ├── enums.go                # 全部状态枚举常量
│   │   └── errors.go               # 业务错误 sentinel（ErrSessionNotFound, ErrPatientNotFound...）
│   │
│   ├── repository/                  # 数据访问层（Repository Pattern）
│   │   ├── patient_repo.go         # PatientRepository 接口定义
│   │   ├── patient_mysql.go        # MySQL 实现
│   │   ├── visit_repo.go           # VisitRepository 接口
│   │   ├── visit_mysql.go
│   │   ├── timeline_repo.go        # TimelineRepository 接口
│   │   ├── timeline_mysql.go
│   │   ├── flow_card_repo.go       # FlowCardRepository 接口
│   │   └── flow_card_mysql.go
│   │
│   ├── service/                     # 业务逻辑层
│   │   ├── patient/
│   │   │   ├── service.go          # PatientService：身份核验、上下文查询、资料更新
│   │   │   └── service_test.go
│   │   ├── visit/
│   │   │   ├── service.go          # VisitService：会话生命周期、状态机、创建/复诊/列表/详情/快照
│   │   │   ├── state_machine.go    # VisitMachineState 转移逻辑
│   │   │   └── service_test.go
│   │   ├── workbench/
│   │   │   ├── service.go          # WorkbenchService：聊天编排、卡片动作、支付、取药、治疗、退出
│   │   │   ├── chat.go             # 消息发送 + assistant-stream 编排
│   │   │   ├── lab.go              # 检验决定 + 结果回填
│   │   │   ├── payment.go          # 支付创建/确认
│   │   │   ├── fulfillment.go      # 取药/配送确认
│   │   │   ├── treatment.go        # 治疗执行推进
│   │   │   ├── vitals.go           # 体征上报 + 急症复检
│   │   │   ├── exit.go             # 主动退出结算（四档后果）
│   │   │   ├── timer.go            # 暂停/恢复总计时
│   │   │   └── service_test.go
│   │   └── medagent/
│   │       ├── adapter.go          # MedAgentAdapter：Step → SSE/Card 映射
│   │       ├── client.go           # medAgent HTTP 客户端（或嵌入式调用）
│   │       └── adapter_test.go
│   │
│   ├── handler/                     # HTTP 处理器（Gin handlers）
│   │   ├── router.go               # 路由注册：将 handler 挂载到 Gin router
│   │   ├── patient_handler.go      # /patients/* endpoints
│   │   ├── visit_handler.go        # /visits CRUD endpoints
│   │   ├── workbench_handler.go    # /visits/:id/messages, /assistant-stream, /lab-decision, etc.
│   │   ├── sse_handler.go          # SSE 流式传输工具（delta/message_final/card/state/emergency/done/error）
│   │   ├── middleware.go           # Handler 层通用工具（参数解析、错误响应封装）
│   │   └── handler_test.go
│   │
│   ├── middleware/                   # Gin 中间件
│   │   ├── auth.go                 # JWT 鉴权中间件（从 Header/Cookie 提取 token）
│   │   ├── cors.go                 # CORS 配置（production 禁止通配符）
│   │   ├── logging.go             # 请求日志（含 request_id）
│   │   ├── recovery.go            # Panic 恢复 + 结构化错误响应
│   │   ├── rate_limit.go          # 速率限制
│   │   └── middleware_test.go
│   │
│   ├── adapter/                     # medAgent 适配层
│   │   ├── step_mapping.go         # medAgent Step.kind → 前端 SSE type / FlowCardKind 映射
│   │   ├── card_builder.go         # 从 medAgent Result/Orders 构造 FlowCard
│   │   ├── timeline_builder.go     # 从 medAgent Turns 构造 TimelineItem
│   │   └── adapter_test.go
│   │
│   └── errors/                      # 统一错误处理
│       ├── api_error.go            # ApiError 结构体（code/message/status/details/retriable）
│       ├── codes.go                # 错误码常量（SESSION_NOT_FOUND, PATIENT_NOT_FOUND, CARD_NOT_FOUND...）
│       └── handler.go              # Gin 错误响应辅助函数
│
├── pkg/                             # 可公开导出的共享包
│   └── api/
│       ├── response.go             # 统一 API 响应信封（success + data + error + metadata）
│       ├── pagination.go           # PageResult[T] 游标分页
│       └── response_test.go
│
├── db/
│   └── migrations/                  # golang-migrate SQL 迁移文件
│       ├── 000001_create_patients.up.sql
│       ├── 000001_create_patients.down.sql
│       ├── 000002_create_visits.up.sql
│       ├── 000002_create_visits.down.sql
│       ├── 000003_create_timeline.up.sql
│       ├── 000003_create_timeline.down.sql
│       ├── 000004_create_flow_cards.up.sql
│       └── 000004_create_flow_cards.down.sql
│
├── tests/                           # 测试支撑文件
│   ├── testutil/
│   │   ├── mysql_container.go      # testcontainers MySQL 容器启动/销毁工具
│   │   └── dbtest.go               # per-test 临时数据库创建与迁移
│   ├── newman/
│   │   ├── neuhis-agent.postman_collection.json   # Newman 冒烟测试集
│   │   └── neuhis-agent.postman_environment.json  # 环境变量（Docker Compose 地址）
│   └── seed/
│       └── testdata.sql             # 测试种子数据
│
├── medAgent/                        # Git submodule：AI 诊疗引擎
│   ├── agent/                       # 公开包（HTTP handler + 库 API）
│   │   ├── new.go                  # agent.New() 构造函数
│   │   ├── service.go              # Service 核心方法（PatientSay, TestResults, DrugInfo...）
│   │   ├── httpapi.go              # HTTP handler 挂载
│   │   ├── types.go                # Step, Result, SessionRecord, Config 类型
│   │   ├── session.go              # 内存会话管理 + TTL 回收
│   │   ├── guardian.go             # 急症守护（并发拦截）
│   │   ├── convert.go              # 类型转换
│   │   ├── record.go               # SessionRecord 导出
│   │   └── errors.go               # Sentinel errors
│   ├── internal/                    # medAgent 内部（不可外部访问）
│   │   ├── ai/                     # AI 决策编排层
│   │   │   ├── agent.go            # 基础 agent 抽象
│   │   │   ├── agent_triage.go     # 分诊 agent
│   │   │   ├── agent_interview.go  # 问诊 agent
│   │   │   ├── agent_treatment.go  # 处置 agent
│   │   │   ├── agent_guardian.go   # 急症守护 agent
│   │   │   ├── layer.go            # LLM 调用 + 护栏（轮数限制）
│   │   │   ├── llm.go              # LLM 客户端接口
│   │   │   ├── prompts.go          # Prompt 模板
│   │   │   ├── snapshot.go         # 诊断快照
│   │   │   ├── intent.go           # 意图识别
│   │   │   └── config.go           # AI 层配置
│   │   ├── consultlog/             # 诊疗日志（JSONL）
│   │   ├── openaicompat/           # OpenAI 兼容客户端
│   │   └── envfile/                # .env 文件读取
│   ├── cmd/
│   │   ├── server/main.go          # medAgent 独立进程入口
│   │   ├── consult/main.go         # 命令行手动接诊工具
│   │   └── smoke/main.go           # 冒烟测试
│   └── docs/
│       └── 后端接入指南.md
│
├── docs/
│   ├── SPEC.md                     # 项目目标与要求
│   ├── STRUCTURE.md                # 本文档
│   └── front-api.md                # 前端接口合约（权威 REST/SSE 基线）
│
├── .github/
│   └── workflows/
│       └── ci.yml                  # GitHub Actions CI：go test -cover ≥90% + golangci-lint
│
├── .pre-commit-config.yaml          # Pre-commit hooks：go test cover ≥90% + golangci-lint
├── .golangci.yml                   # golangci-lint 配置
├── .env.example                    # 配置模板（含注释说明，提交到 git）
├── .env                            # 本地配置（从 .env.example 复制，gitignore）
├── go.mod
├── go.sum
├── Makefile                         # 常用命令：test, lint, migrate, run, docker-up, smoke-test
├── Dockerfile                       # 多阶段构建（Gin 服务）
├── docker-compose.yml               # 全量容器编排（Gin + MySQL + medAgent）
├── CLAUDE.md                        # Claude Code 项目指令
└── AGENTS.md                        # Agent 配置
```

---

## 5. 依赖关系

### 5.1 Go Module 外部依赖（计划）

```
module github.com/neuhis/software-practice-backend

go 1.22

require (
    // HTTP 框架
    github.com/gin-gonic/gin

    // 数据库
    github.com/go-sql-driver/mysql        // MySQL 驱动
    github.com/golang-migrate/migrate/v4  // 数据库迁移

    // 鉴权
    github.com/golang-jwt/jwt/v5          // JWT 处理

    // 配置
    github.com/joho/godotenv              // .env 文件解析

    // 校验
    github.com/go-playground/validator/v10 // 请求体校验（配合 Gin binding）

    // 集成测试
    github.com/testcontainers/testcontainers-go // Docker 容器生命周期管理
    github.com/testcontainers/testcontainers-go/modules/mysql // MySQL 模块
)
```

### 5.2 内部依赖图

```
cmd/server
  └── internal/config
  └── internal/handler
        └── internal/service/patient
        │     └── internal/repository (PatientRepo)
        └── internal/service/visit
        │     └── internal/repository (VisitRepo)
        └── internal/service/workbench
        │     └── internal/repository (TimelineRepo, FlowCardRepo)
        │     └── internal/service/medagent
        │           └── medAgent/agent (HTTP client / 嵌入)
        └── internal/middleware
              └── internal/config
```

### 5.3 medAgent 集成模式

支持两种模式，通过配置切换：

1. **嵌入式（库用法）**：`agent.New(cfg)` → `svc.Handler()` 挂载到 Gin router 子路径（`/ai/`）
2. **独立进程（推荐）**：medAgent 作为独立 HTTP 服务运行，后端通过 `MedAgentClient`（HTTP client）调用

生产环境推荐独立进程模式，便于独立扩缩容与故障隔离。

---

## 6. 关键设计决策

### 6.1 Repository Pattern

数据访问统一通过接口抽象：

```go
type PatientRepository interface {
    FindByCredential(ctx context.Context, credType, cred string) (*model.Patient, error)
    FindByID(ctx context.Context, id string) (*model.Patient, error)
    UpdateProfile(ctx context.Context, id string, input model.ProfileUpdate) (*model.Patient, error)
}
```

- 业务逻辑依赖接口，不依赖具体存储
- 测试时用 mock 实现，隔离数据库
- 集成测试用 per-test MySQL 临时数据库

### 6.2 统一 API 响应信封

所有非流式响应遵循统一格式：

```json
{
  "success": true,
  "data": { ... },
  "error": null,
  "meta": { "total": 100, "pageSize": 20 }
}
```

流式响应（SSE）按 `AssistantStreamEvent` 格式逐事件下发，每个事件为一行 `data: <JSON>\n\n`。

### 6.3 medAgent Step ↔ 前端映射

| medAgent `Step.kind` | 后端动作 | SSE 事件 | 前端卡片 |
| --- | --- | --- | --- |
| `ASK` | 透传 `doctor_say` | `delta` × n + `message_final` | AI 追问气泡 |
| `NEED_TESTS` | 构造 `lab_decision` 卡 | `card(lab_decision)` + `state` | 是否检验阻塞卡 |
| `DRUG_QUERY` | 查药品规格（后台）→ 调 `/drug-info` | `state` | 系统事件「正在核对药品规格」 |
| `PURCHASE` | 构造 `medication_fulfillment` 卡 | `card(medication_fulfillment)` + `state` | 购药/取药确认卡 |
| `EMERGENCY` | 立即终止会话 | `emergency` | 急症 Overlay |
| `DONE` | 构造 `diagnosis` + `completed_visit`/`advice_only` | `card(diagnosis)` + `card(...)` + `done` | 诊断卡 + 完成卡 |

### 6.4 状态机

VisitSession 有 10 种对外状态（`VisitStatus`），内部状态机 17 种（`VisitMachineState`）。后端 Service 层驱动状态转移，关键约束：

- `blocked` 状态必须携带 `activeCardId`
- `new` 入口不得带 `parentSessionId`
- 终端状态（`completed`/`emergency_terminated`/`exited`/`transferred`）不可再推进

### 6.5 边界与未实现

对齐 `front-api.md` §8：

| 事项 | 状态 | 说明 |
| --- | --- | --- |
| 院内治疗执行 | 前端/mock 演示 | medAgent 处置暂只有 `MEDICATION`/`ADVICE_ONLY`/`REFERRAL`；治疗类按 `REFERRAL` 终止 |
| 总计时超时 | 前端/mock 机制 | medAgent 无总超时；后端后续可补转诊收口 |
| 急症恢复 | 前端/mock 演示 | medAgent 急症后会话关闭；`dismiss-emergency` 需后端显式支持 |
| 检验项目 | 当前恒为血常规 | Schema 保留 `testItems[]` 以备扩展 |

---

## 7. 测试策略

### 7.1 测试分层

| 层级 | 类型 | 工具 | 覆盖率要求 | 说明 |
| --- | --- | --- | --- | --- |
| Model | 单元测试 | `go test` | ≥90% | 纯数据验证、序列化/反序列化 |
| Repository | 集成测试 | `go test` + testcontainers MySQL | ≥90% | testcontainers-go 拉起临时 Docker MySQL 容器，per-test 独立数据库，测试结束自动销毁 |
| Service | 单元测试 + 集成测试 | `go test` + mock repository | ≥90% | Mock 依赖，验证业务逻辑 |
| Handler | 集成测试 | `go test` + `httptest` | ≥90% | Gin 测试模式，真实 HTTP 请求 |
| Adapter | 单元测试 | `go test` + fake medAgent | ≥90% | Step 映射正确性 |
| 冒烟测试 | E2E | Docker Compose + Newman | 核心流程 | `docker-compose up -d` 全量起容器后，Newman 跑 Postman Collection |

### 7.2 testcontainers 集成测试模式

```go
// tests/testutil/mysql_container.go
func SetupMySQL(t *testing.T) (dsn string, teardown func()) {
    ctx := context.Background()
    req := testcontainers.ContainerRequest{
        Image:        "mysql:8.0",
        ExposedPorts: []string{"3306/tcp"},
        Env: map[string]string{
            "MYSQL_ROOT_PASSWORD": "test",
            "MYSQL_DATABASE":      "neuhis_test",
        },
    }
    mysqlContainer, _ := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
        ContainerRequest: req,
        Started:          true,
    })
    // ... 获取端口，运行 migration ...
    return dsn, func() { mysqlContainer.Terminate(ctx) }
}
```

### 7.3 Newman 冒烟测试

```bash
# CI 或本地：全量启动 → 跑冒烟脚本 → 清理
docker-compose up -d               # 启动 Gin + MySQL + medAgent
make smoke-test                    # newman run tests/newman/neuhis-agent.postman_collection.json
docker-compose down                # 清理
```

- Postman Collection 覆盖 `front-api.md` 核心时序（§7.1–§7.5）
- 所有测试运行 `-race` 标志
- CI 门控：pre-commit hook 确保 `go test -cover ./...` ≥90% + `golangci-lint run ./...` 零告警

---

## 8. 配置（.env）

项目采用 `.env` 文件管理配置，`.env.example` 作为模板提交到 git，开发者复制为 `.env` 后填入真实值。`.env` 纳入 `.gitignore`。支持 `.env.local` 覆盖（用于本地调试，优先级最高）。

### 8.1 .env.example 模板

```bash
# =========================
# NEUHIS Agent 后端配置
# 复制此文件为 .env 并修改
# =========================

# --- 服务 ---
SERVER_ADDR=:8080
SERVER_MODE=release               # debug | test | release

# --- 数据库 ---
DATABASE_DSN=root:password@tcp(mysql:3306)/neuhis?charset=utf8mb4&parseTime=True&loc=Local

# --- JWT ---
JWT_SECRET=change-me-to-a-random-string-at-least-32-bytes-long
# 约束：≥32 字节；禁用弱口令黑名单（如 123456..., changeme..., secret... 等）

# --- CORS ---
CORS_ALLOWED_ORIGINS=http://localhost:5173
# production 下不可为 *

# --- medAgent 集成 ---
MEDAGENT_MODE=http                 # http（独立进程）| embedded（库用法）
MEDAGENT_BASE_URL=http://medagent:8080
MEDAGENT_API_KEY=sk-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
MEDAGENT_PROVIDER=deepseek         # deepseek | qwen | openai
MEDAGENT_MODEL=deepseek-chat

# --- 日志 ---
LOG_LEVEL=info                     # debug | info | warn | error
```

### 8.2 配置加载流程

```
.env.example ──(复制)──► .env ──(覆盖)──► .env.local (可选)
                              │
                              ▼
                       config.Load()
                              │
                     ┌───────┴────────┐
                     │ 解析 .env       │
                     │ 校验必填项       │
                     │ JWT ≥32 字节    │
                     │ JWT 弱口令黑名单  │
                     │ production CORS  │
                     │ 返回 Config 结构体│
                     └────────────────┘
```

### 8.3 配置项清单

| 变量 | 说明 | 默认值 | 约束 |
| --- | --- | --- | --- |
| `SERVER_ADDR` | 监听地址 | `:8080` | — |
| `SERVER_MODE` | Gin 运行模式 | `release` | `debug` / `test` / `release` |
| `DATABASE_DSN` | MySQL DSN | 必填 | — |
| `JWT_SECRET` | JWT 签名密钥 | 必填 | ≥32 字节，弱口令黑名单校验 |
| `CORS_ALLOWED_ORIGINS` | 允许的跨域来源 | 必填(production) | production 下不可为 `*` |
| `MEDAGENT_MODE` | medAgent 集成模式 | `http` | `http`（独立进程）/ `embedded`（库用法） |
| `MEDAGENT_BASE_URL` | medAgent 服务地址 | `http://localhost:8080` | `MEDAGENT_MODE=http` 时必填 |
| `MEDAGENT_API_KEY` | medAgent LLM API Key | 必填 | DeepSeek / Qwen / OpenAI key |
| `MEDAGENT_PROVIDER` | medAgent LLM Provider | `deepseek` | `deepseek` / `qwen` / `openai` |
| `MEDAGENT_MODEL` | medAgent 模型名 | `deepseek-chat` | — |
| `LOG_LEVEL` | 日志级别 | `info` | `debug` / `info` / `warn` / `error` |

### 8.4 Pre-commit 配置

`.pre-commit-config.yaml` 定义 Git pre-commit hooks：

```yaml
# .pre-commit-config.yaml
repos:
  - repo: local
    hooks:
      - id: go-test-cover
        name: Go Test Cover (≥90%)
        entry: bash -c 'go test -race -cover ./...'
        language: system
        pass_filenames: false
        types: [go]

      - id: golangci-lint
        name: golangci-lint
        entry: golangci-lint run ./...
        language: system
        pass_filenames: false
        types: [go]
```

**安装**：`pre-commit install`  
**手动触发**：`pre-commit run --all-files`

---

## 9. 命令速查

```bash
# ==========================
# 本地开发
# ==========================

# 复制配置文件
cp .env.example .env
# 编辑 .env 填入真实值

# 安装 pre-commit hooks
pre-commit install

# 启动全量服务（Gin + MySQL + medAgent）
docker-compose up -d

# 查看日志
docker-compose logs -f app

# 停止并清理
docker-compose down

# ==========================
# 测试
# ==========================

# 单元测试（含竞态检测 + 覆盖率）
go test -race -cover ./...

# 集成测试（testcontainers 自动拉起 Docker MySQL 容器）
go test -race -cover ./internal/...

# 查看覆盖率报告
go test -coverprofile=coverage.out ./... && go tool cover -html=coverage.out

# 冒烟测试（需先 docker-compose up -d）
make smoke-test
# 或手动：
newman run tests/newman/neuhis-agent.postman_collection.json \
  -e tests/newman/neuhis-agent.postman_environment.json

# ==========================
# 代码质量
# ==========================

# 静态分析
golangci-lint run ./...

# Pre-commit 全量检查
pre-commit run --all-files

# 安全扫描
gosec ./...

# ==========================
# 数据库
# ==========================

# 运行迁移
make migrate-up

# 回滚迁移
make migrate-down

# ==========================
# 构建与部署
# ==========================

# 本地编译
go build -o bin/server ./cmd/server

# Docker 构建
docker build -t neuhis-backend .

# Docker Compose 重建
docker-compose up -d --build
```
