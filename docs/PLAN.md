# PLAN.md — NEUHIS Agent 后端开发计划

> 本文档描述将当前项目从零开发至 SPEC.md 和 STRUCTURE.md 要求状态的完整路径。
>
> **基线文档**：
> - [SPEC.md](./SPEC.md) — 项目目标与质量要求
> - [STRUCTURE.md](./STRUCTURE.md) — 软件架构、分层设计与包结构
> - [rest-api.md](./rest-api.md) — 前端 REST/SSE API 合约（权威基线）

---

## 当前状态评估

| 维度 | 现有 | 目标 |
| --- | --- | --- |
| Go 模块 | ❌ 无 `go.mod` | Go 1.22 module |
| 目录结构 | ❌ 仅 `docs/`、`medAgent/` | 完整 `cmd/`、`internal/`、`pkg/`、`db/`、`tests/` |
| 数据模型 | ❌ 无 | `internal/model/` 全部领域实体+枚举+错误 |
| 配置系统 | ❌ 无 | `internal/config/` 含校验 |
| 数据库 | ❌ 无 migration | 4 张表 migration + Repository Pattern |
| 中间件 | ❌ 无 | auth / CORS / logging / recovery / rate-limit |
| 业务逻辑 | ❌ 无 | Patient / Visit / Workbench / medAgent 适配 |
| HTTP 层 | ❌ 无 | Gin router + 全部 endpoint handler + SSE |
| medAgent 适配 | ❌ medAgent submodule 存在 | Step → SSE / Card / Timeline 映射 |
| 测试 | ❌ 无 | 单元 + 集成(testcontainers) + Newman 冒烟 ≥90% |
| CI/CD | ❌ 无 | GitHub Actions + pre-commit hooks |
| 容器化 | ❌ 无 | Dockerfile + docker-compose.yml |

---

## 阶段总览

```
Phase 0: 项目骨架 ──► Phase 1: 数据模型 ──► Phase 2: 配置与基础设施
      │                                              │
      └──► Phase 3: 数据访问层 ◄─────────────────────┘
              │
              └──► Phase 4: 中间件 ──► Phase 5: 业务逻辑层
                                            │
              Phase 7: medAgent 适配层 ◄────┘
                     │
                     └──► Phase 6: HTTP 传输层
                                  │
                                  └──► Phase 8: 测试与冒烟
                                              │
                                              └──► Phase 9: 质量门控与收尾
```

---

## Phase 0: 项目骨架搭建

### 目的

建立项目的基础设施：Go 模块、目录结构、配置文件模板、容器化、CI/CD、代码质量工具链。这是所有后续阶段的基石。

### 需要创建/修改的文件

| 文件 | 说明 |
| --- | --- |
| `go.mod` | 初始化 `module github.com/neuhis/software-practice-backend`，Go 1.22 |
| `cmd/server/main.go` | 入口骨架：加载配置 → 初始化 DB → 注册路由 → 启动 Gin（初期可为空 `main`） |
| `.env.example` | 完整配置模板（SERVER_ADDR, DATABASE_DSN, JWT_SECRET, CORS_ALLOWED_ORIGINS, MEDAGENT_*、LOG_LEVEL） |
| `.env` | `.gitignore` 中的本地配置（从 `.env.example` 复制） |
| `.gitignore` | 忽略 `.env`、`bin/`、`coverage.out` 等 |
| `.golangci.yml` | golangci-lint 配置（启用 gosec、errcheck、govet 等） |
| `.pre-commit-config.yaml` | Pre-commit hooks：`go test -race -cover ./...` + `golangci-lint run ./...` |
| `Makefile` | 常用命令：`test`、`lint`、`migrate-up`、`migrate-down`、`run`、`docker-up`、`smoke-test` |
| `Dockerfile` | 多阶段构建：`go build` → 精简运行时镜像 |
| `docker-compose.yml` | 全量容器编排：Gin 服务 + MySQL 8.0 + medAgent |
| `.github/workflows/ci.yml` | GitHub Actions CI：go test cover ≥90% + golangci-lint + testcontainers + docker-compose up + Newman |
| `db/migrations/` 目录 | 创建空目录，migration 文件在 Phase 3 编写 |
| `tests/` 目录结构 | `tests/testutil/`、`tests/newman/`、`tests/seed/` 目录 |
| 全部 `internal/` 子目录 | 按 STRUCTURE.md §4 创建目录结构，每目录含 `doc.go` |

### 验收方案

```bash
# 1. Go 模块可编译
go build ./...

# 2. 目录结构与 STRUCTURE.md §4 一致
find . -type d | sort  # 核对

# 3. Docker Compose 可启动（服务占位，MySQL 可连接）
docker-compose up -d
docker-compose ps          # 所有服务 running
docker-compose down

# 4. Pre-commit 可安装
pre-commit install
pre-commit run --all-files # 通过（初期无 .go 文件，空运行）

# 5. CI 配置文件语法正确
# GitHub Actions workflow 通过 schema 校验
```

---

## Phase 1: 数据模型层

### 目的

定义全部领域实体、枚举类型、DTO 和业务错误 sentinel。本层零外部依赖，是后续所有层的类型基础。

### 前置依赖

- Phase 0（目录结构就绪）

### 需要创建/修改的文件

| 文件 | 说明 |
| --- | --- |
| `internal/model/enums.go` | 全部状态枚举常量：`VisitStatus`(10值)、`VisitMachineState`(17值)、`TerminalReason`(7值)、`PaymentStatus`(5值)、`VisitEntryType`、`FlowCardKind`(9值)、`FlowCardStatus`(9值)、`TimelineItemKind`(4值)、`TimelineItemStatus`(5值)、`SystemEventType`(8值)、`SSEEventType`(7值) |
| `internal/model/patient.go` | `PatientProfile`、`PatientContext`、`PatientPriorVisit`、`ProfileUpdateInput`、`CredentialType`、`ReadableScope` |
| `internal/model/visit.go` | `VisitSession`(含全部字段，对齐 rest-api.md §5.2 `visitSessionSchema`)、`VisitSessionSummary`、`VisitSummary`、`VisitSnapshot`、`CreateSessionInput`、`CreateFollowUpInput` |
| `internal/model/timeline.go` | `TimelineItem` 判别联合（message/flow_card/system_event/terminal），`TimelineItemBase`、各 kind 专属字段 |
| `internal/model/flow_card.go` | `FlowCard` 判别联合（9 种类型），`FlowCardBase`、各 kind 专属字段，对齐 rest-api.md §6.3 |
| `internal/model/sse.go` | `AssistantStreamEvent` 判别联合（7 种 type），对齐 rest-api.md §6.2 |
| `internal/model/payment.go` | `PaymentStatus`、`PaymentItem`、`PaymentInfo` |
| `internal/model/errors.go` | 业务错误 sentinel：`ErrSessionNotFound`、`ErrPatientNotFound`、`ErrCardNotFound`、`ErrValidation` 等 |

### 验收方案

```bash
# 1. 编译通过（模型层无外部依赖）
go build ./internal/model/...

# 2. 序列化/反序列化测试
go test -v ./internal/model/...

# 3. JSON tag 对齐 rest-api.md
# 手动核对关键结构体的 JSON 输出与文档一致

# 4. 枚举常量与文档一致
grep -r "VisitStatus\|VisitMachineState\|FlowCardKind" internal/model/enums.go | wc -l  # ≥ 对应数量
```

---

## Phase 2: 配置系统与公共包

### 目的

实现配置加载与校验，以及统一 API 响应信封和分页工具。这是中间件和 handler 的依赖。

### 前置依赖

- Phase 0（`.env.example` 模板就绪）

### 需要创建/修改的文件

| 文件 | 说明 |
| --- | --- |
| `internal/config/config.go` | `Config` 结构体 + `Load()` 函数：解析 `.env`，校验必填项（DATABASE_DSN、JWT_SECRET），JWT ≥32 字节 + 弱口令黑名单，production 下 CORS 不可为 `*` |
| `internal/config/env.go` | `.env` 文件解析（支持 `.env.local` 覆盖），基于 `godotenv` |
| `internal/config/config_test.go` | 配置校验全覆盖：有效配置、缺失必填项、JWT 过短、弱口令拒绝、production 通配 CORS 拒绝 |
| `pkg/api/response.go` | 统一 API 响应信封：`ApiResponse[T]`（success + data + error + meta） |
| `pkg/api/pagination.go` | `PageResult[T]` 游标分页（items + nextCursor + hasMore） |
| `pkg/api/response_test.go` | 响应序列化、分页正确性测试 |
| `internal/errors/api_error.go` | `ApiError` 结构体（code/message/status/details/retriable） |
| `internal/errors/codes.go` | 错误码常量：`SESSION_NOT_FOUND`、`PATIENT_NOT_FOUND`、`CARD_NOT_FOUND`、`VALIDATION_ERROR` 等 |
| `internal/errors/handler.go` | Gin 错误响应辅助函数 |

### 验收方案

```bash
# 1. 配置加载测试
go test -v ./internal/config/... -cover
# 覆盖率 ≥90%

# 2. 配置校验边界
# - 有效配置：无错误
# - JWT_SECRET=short：报错
# - JWT_SECRET=123456...（弱口令）：报错
# - CORS_ALLOWED_ORIGINS=* + SERVER_MODE=release：报错
# - DATABASE_DSN 缺失：报错
# - .env.local 覆盖：正确合并

# 3. API 响应信封
go test -v ./pkg/api/... -cover
# 序列化/反序列化正确
```

---

## Phase 3: 数据访问层与数据库迁移

### 目的

使用 Repository Pattern 实现数据持久化，编写 migration 脚本，搭建 testcontainers 集成测试基础设施。

### 前置依赖

- Phase 1（模型定义就绪）
- Phase 2（配置系统就绪，可读取 DATABASE_DSN）

### 需要创建/修改的文件

| 文件 | 说明 |
| --- | --- |
| `db/migrations/000001_create_patients.up.sql` | `patients` 表：id, name, gender, age, phone_masked, id_card_masked, allergies(JSON), chronic_diseases(JSON), long_term_medications(JSON), created_at, updated_at |
| `db/migrations/000001_create_patients.down.sql` | DROP TABLE patients |
| `db/migrations/000002_create_visits.up.sql` | `visits` 表：id, patient_id(FK), entry_type, status, machine_state, started_at, updated_at, ended_at, timeout_at, paused_at, ask_round, ask_round_limit, lab_round, lab_round_limit, parent_session_id, terminal_reason, active_card_id, timer_paused, summary(JSON) |
| `db/migrations/000002_create_visits.down.sql` | DROP TABLE visits |
| `db/migrations/000003_create_timeline.up.sql` | `timeline_items` 表：id, session_id(FK), kind, status, content(JSON), created_at |
| `db/migrations/000003_create_timeline.down.sql` | DROP TABLE timeline_items |
| `db/migrations/000004_create_flow_cards.up.sql` | `flow_cards` 表：id, session_id(FK), kind, status, blocking, title, content(JSON), lock_reason, created_at, handled_at |
| `db/migrations/000004_create_flow_cards.down.sql` | DROP TABLE flow_cards |
| `internal/repository/patient_repo.go` | `PatientRepository` 接口：`FindByCredential`、`FindByID`、`UpdateProfile` |
| `internal/repository/patient_mysql.go` | MySQL 实现 |
| `internal/repository/visit_repo.go` | `VisitRepository` 接口：`Create`、`FindByID`、`ListByPatient`、`UpdateStatus`、`UpdateSummary` |
| `internal/repository/visit_mysql.go` | MySQL 实现 |
| `internal/repository/timeline_repo.go` | `TimelineRepository` 接口：`Append`、`ListBySession`(cursor)、`UpdateStatus` |
| `internal/repository/timeline_mysql.go` | MySQL 实现 |
| `internal/repository/flow_card_repo.go` | `FlowCardRepository` 接口：`Create`、`FindByID`、`ListBySession`、`UpdateStatus` |
| `internal/repository/flow_card_mysql.go` | MySQL 实现 |
| `tests/testutil/mysql_container.go` | testcontainers MySQL 容器启动/销毁：`SetupMySQL(t) (dsn, teardown)` |
| `tests/testutil/dbtest.go` | per-test 临时数据库创建 + migration 自动执行 |
| `tests/seed/testdata.sql` | 测试种子数据（患者、会话、时间线样例） |

### 验收方案

```bash
# 1. Migration 正确性
make migrate-up    # 创建全部 4 张表
make migrate-down  # 回滚全部表

# 2. Repository 集成测试（testcontainers）
go test -v ./internal/repository/... -cover
# 每项 CRUD 操作验证：创建→查询→更新→删除
# 覆盖率 ≥90%

# 3. 并发安全
go test -race ./internal/repository/...

# 4. 测试隔离
# 每个测试使用独立临时数据库，测试结束后数据清除
```

---

## Phase 4: Gin 中间件

### 目的

实现 HTTP 中间件层：JWT 鉴权、CORS、请求日志、panic 恢复、速率限制。

### 前置依赖

- Phase 2（配置系统就绪）

### 需要创建/修改的文件

| 文件 | 说明 |
| --- | --- |
| `internal/middleware/auth.go` | JWT 鉴权中间件：从 `Authorization: Bearer <token>` 提取 token，解析 JWT，将 `patientId` 注入 `gin.Context` |
| `internal/middleware/cors.go` | CORS 配置：从 Config 读取 `CORS_ALLOWED_ORIGINS`，production 禁止通配符 |
| `internal/middleware/logging.go` | 请求日志：记录 method、path、status、latency、`X-Request-Id` |
| `internal/middleware/recovery.go` | Panic 恢复：捕获 panic，返回结构化 JSON 错误响应 |
| `internal/middleware/rate_limit.go` | 速率限制：基于 token 或 IP 的简单令牌桶 |
| `internal/middleware/middleware_test.go` | 全部中间件测试：使用 `httptest` + Gin 测试模式 |

### 验收方案

```bash
# 1. 中间件测试
go test -v ./internal/middleware/... -cover
# 覆盖率 ≥90%

# 2. 鉴权验证
# - 无 token → 401
# - 无效 token → 401
# - 有效 token → 通过，context 含 patientId
# - 过期 token → 401

# 3. CORS 验证
# - debug 模式：允许通配符
# - release 模式 + 通配符 → 服务启动失败（在 config 层验证）

# 4. Recovery 验证
# - handler 内 panic → 返回 500 + ApiError JSON

# 5. 日志格式验证
# - 请求/响应日志含 request_id、method、path、status、latency_ms
```

---

## Phase 5: 业务逻辑层

### 目的

实现全部业务逻辑，包括患者服务、会话生命周期与状态机、工作台编排（聊天、检验、支付、取药、治疗、体征、退出、计时），以及 medAgent HTTP 客户端。

这是整个项目最核心、代码量最大的阶段。

### 前置依赖

- Phase 1（模型定义）
- Phase 3（Repository 接口，可 mock）
- Phase 2（配置系统，medAgent 相关配置）

### 需要创建/修改的文件

#### 5a. 患者服务

| 文件 | 说明 |
| --- | --- |
| `internal/service/patient/service.go` | `PatientService`：`VerifyIdentity`（按凭证查找/创建患者）、`GetContext`（组装问诊上下文含上次就诊纪要）、`UpdateProfile`（更新过敏史/慢病/长期用药） |
| `internal/service/patient/service_test.go` | Mock repository，全覆盖三种操作 |

#### 5b. 会话服务

| 文件 | 说明 |
| --- | --- |
| `internal/service/visit/service.go` | `VisitService`：`CreateSession`（新建，生成 initialTimeline）、`CreateFollowUp`（复诊，携父会话纪要）、`GetSession`（含校验 `entryType=new` 不带 `parentSessionId`）、`ListSessions`（cursor 分页）、`GetSnapshot`（只读完整快照）、`UpdateStatus` |
| `internal/service/visit/state_machine.go` | `VisitMachineState` 转移逻辑：17 种内部态的状态转移表，含阻塞规则（`blocked` 必须携带 `activeCardId`）、终端状态不可再推进 |
| `internal/service/visit/service_test.go` | 全部操作 + 状态机转移测试 |

#### 5c. medAgent 客户端

| 文件 | 说明 |
| --- | --- |
| `internal/service/medagent/client.go` | `MedAgentClient`：HTTP 客户端封装，调用 medAgent 独立进程的 `POST /sessions` → `POST /patient-say` → `POST /test-results` → `POST /drug-info` → `POST /purchase-result` → `GET /record` → `DELETE` |
| `internal/service/medagent/adapter.go` | `MedAgentAdapter`：将 medAgent `Step` 流转换为后端内部事件（见 Phase 7 细化的 adapter 层） |

#### 5d. 工作台服务（核心）

| 文件 | 说明 |
| --- | --- |
| `internal/service/workbench/service.go` | `WorkbenchService` 构造函数与依赖注入 |
| `internal/service/workbench/chat.go` | `SendMessage`（患者消息，返回占位+会话状态）、`StreamAssistantMessage`（组装 medAgent 调用 → SSE 流式事件） |
| `internal/service/workbench/lab.go` | `SubmitLabDecision`（accepted/skipped/vetoed 三路分支）、检验结果回填 `SubmitLabResults` |
| `internal/service/workbench/payment.go` | `SubmitPayment`（创建/确认检验或药品支付，含 `defer` 暂缓）、支付状态管理 |
| `internal/service/workbench/fulfillment.go` | `SubmitFulfillment`（取药 pickup/delivery 确认） |
| `internal/service/workbench/treatment.go` | `SubmitTreatmentExecution`（schedule → confirm_arrival → start → complete → cancel）、`AckAdvice`（仅医嘱确认） |
| `internal/service/workbench/vitals.go` | `ReportVitals`（体征上报 → 急症复检判断）、`DismissEmergency`（误报申诉解除急症态） |
| `internal/service/workbench/exit.go` | `ExitVisit`（主动退出结算，四档后果：no_fee / refundable / executed_no_refund / medication_dispensed） |
| `internal/service/workbench/timer.go` | `PauseTimer` / `ResumeTimer`（暂停/恢复总计时） |
| `internal/service/workbench/service_test.go` | Mock 全部依赖（repository + medAgent client），覆盖主流程 + 急症 + 超时 + 退出四档 |

### 验收方案

```bash
# 1. 服务层全量测试
go test -v ./internal/service/... -cover
# 覆盖率 ≥90%

# 2. 状态机完整性
# - 每个合法转移路径有测试
# - 每个非法转移路径有拒绝测试
# - 终端状态（completed/emergency_terminated/exited/transferred）拒绝推进
# - blocked 状态必须携带 activeCardId

# 3. 主流程时序验证（mock medAgent）
# - 新建会话 → 发消息 → AI 追问(ASK) → 检验(NEED_TESTS) → 缴费 → 诊断(DONE) → 用药(PURCHASE) → 取药 → 完成
# - 急症(EMERGENCY) → 会话终止
# - 仅医嘱(ADVICE_ONLY) → 确认 → 完成
# - 退出结算四档后果正确

# 4. 竞态检测
go test -race ./internal/service/...
```

---

## Phase 6: medAgent 适配层

### 目的

将 medAgent 的 `Step` 输出转译为前端可消费的 `AssistantStreamEvent`（SSE 事件）、`FlowCard`（流程卡）和 `TimelineItem`（时间线条目）。

### 前置依赖

- Phase 1（模型定义）
- Phase 5c（medAgent 客户端与 adapter 接口定义）

### 需要创建/修改的文件

| 文件 | 说明 |
| --- | --- |
| `internal/adapter/step_mapping.go` | `Step.kind` → SSE type / `FlowCardKind` 映射表：`ASK`→delta/message_final、`NEED_TESTS`→card(lab_decision)、`DRUG_QUERY`→state、`PURCHASE`→card(medication_fulfillment)、`EMERGENCY`→emergency、`DONE`→card(diagnosis)+card(completed_visit/advice_only)+done |
| `internal/adapter/card_builder.go` | 从 medAgent `Step.Result`/`Step.Orders` 构造各类型 `FlowCard`（lab_decision、payment、diagnosis、treatment_plan、medication_fulfillment、advice_only、completed_visit） |
| `internal/adapter/timeline_builder.go` | 从 medAgent `SessionRecord.Turns` 构造 `TimelineItem`（message/flow_card/system_event/terminal） |
| `internal/adapter/adapter_test.go` | 全部映射 + 构造测试：fake medAgent Step → 验证 SSE 事件结构、FlowCard 字段、TimelineItem kind |

### 验收方案

```bash
# 1. 适配层测试
go test -v ./internal/adapter/... -cover
# 覆盖率 ≥90%

# 2. 映射完整性
# - 6 种 Step.kind 每种至少一个测试 case
# - 验证输出 SSE 事件 type 正确
# - 验证输出 FlowCard kind 正确
# - DRUG_QUERY 不产出卡片，仅 state 事件

# 3. 边界条件
# - 空 Result/Orders
# - 多 Turn 合并
# - 异常 Step.kind（未知类型处理）
```

---

## Phase 7: HTTP 传输层

### 目的

实现 Gin HTTP handler，将所有 endpoint 挂载到路由，完成请求解析 → Service 调用 → 响应序列化的完整链路。包括 SSE 流式传输。

### 前置依赖

- Phase 4（中间件就绪）
- Phase 5（业务逻辑就绪）
- Phase 6（adapter 就绪）

### 需要创建/修改的文件

| 文件 | 说明 |
| --- | --- |
| `internal/handler/router.go` | 路由注册：`/api` 前缀，public 路由（`POST /patients/verify`）+ 鉴权路由组（其余所有 endpoint） |
| `internal/handler/patient_handler.go` | `POST /patients/verify`、`GET /patients/:patientId/context`、`PATCH /patients/:patientId/profile` |
| `internal/handler/visit_handler.go` | `POST /visits`、`POST /visits/:sessionId/follow-up`、`GET /visits`、`GET /visits/:sessionId`、`GET /visits/:sessionId/snapshot` |
| `internal/handler/workbench_handler.go` | `/visits/:sessionId/timeline`、`/messages`、`/assistant-stream`、`/lab-decision`、`/payments`、`/fulfillment`、`/treatment-execution`、`/advice-ack`、`/lock-question`、`/classify-intent`、`/consult`、`/vitals`、`/exit`、`/timer`、`/dismiss-emergency` |
| `internal/handler/sse_handler.go` | SSE 流式传输工具：`SSEWriter`（设置 Content-Type、Flusher、定时 heartbeat），事件序列化 `WriteEvent(event)` |
| `internal/handler/middleware.go` | Handler 层通用工具：`BindJSON`、`ParseSessionID`、`ParsePatientID`、`WriteError`、`WriteSuccess` |
| `internal/handler/handler_test.go` | 全量 handler 集成测试：`httptest.NewRecorder` + Gin 测试模式，验证 HTTP 状态码、响应 JSON 结构 |
| `cmd/server/main.go` | **更新**：完整启动流程，注册全部路由，挂载中间件 |

### 验收方案

```bash
# 1. Handler 集成测试
go test -v ./internal/handler/... -cover
# 覆盖率 ≥90%

# 2. Endpoint 对比（rest-api.md §4 清单）
# 全部 22 个 endpoint 逐一验证：
curl -X POST /api/patients/verify -H "Content-Type: application/json" -d '{...}'
curl -X GET  /api/patients/:id/context
# ... 逐一核对

# 3. 响应格式
# - 非流式响应含 {success, data, error, meta}
# - 错误响应含 {code, message, status, details, retriable}
# - 分页响应含 {items, nextCursor, hasMore}

# 4. SSE 流式响应
# - Content-Type: text/event-stream
# - 每行 data: <JSON>\n\n
# - 支持 AbortSignal 中断

# 5. 状态码验证
# - 404: SESSION_NOT_FOUND / PATIENT_NOT_FOUND
# - 403: 越权访问
# - 400: VALIDATION_ERROR
```

---

## Phase 8: 集成测试与冒烟测试

### 目的

编写 testcontainers 集成测试（真实 MySQL）、Newman 冒烟测试集（Postman Collection）、以及测试种子数据。确保端到端流程正确。

### 前置依赖

- Phase 7（全部 endpoint 实现）

### 需要创建/修改的文件

| 文件 | 说明 |
| --- | --- |
| `tests/newman/neuhis-agent.postman_collection.json` | Newman 冒烟测试集：覆盖 rest-api.md §7.1–§7.5 核心时序（新建会话→聊天→检验→缴费→诊断→用药→取药→完成、急症、超时退出、完成态咨询/复诊） |
| `tests/newman/neuhis-agent.postman_environment.json` | 环境变量：Docker Compose 服务地址 |
| `tests/seed/testdata.sql` | 测试种子数据更新：预置患者、历史会话 |
| `docker-compose.yml` | **更新**：确保 Gin 服务可连接 MySQL + medAgent，Newman 作为一次性服务运行 |

### 验收方案

```bash
# 1. 全量集成测试（testcontainers）
go test -race -cover ./internal/...
# 覆盖率 ≥90%

# 2. Newman 冒烟测试
docker-compose up -d                        # 全量启动
make smoke-test                             # newman run collection
# 全部 request 通过
docker-compose down                         # 清理

# 3. CI 模拟
# GitHub Actions 上：
# - go test -race -cover ./... （含 testcontainers）
# - docker-compose up -d
# - newman run ...
# - docker-compose down
```

---

## Phase 9: 质量门控与收尾

### 目的

确保全部质量门控通过，修复所有 lint 告警，验证覆盖率达标，完成文档与配置收尾。

### 前置依赖

- Phase 8（全部测试就绪）

### 需要做的事项

| 事项 | 说明 |
| --- | --- |
| 覆盖率达标验证 | `go test -race -cover ./...` 总覆盖率 ≥90%，逐包不达标则补测试 |
| golangci-lint 零告警 | `golangci-lint run ./...` 修复全部 issue |
| Pre-commit hook 验证 | `pre-commit run --all-files` 全部通过 |
| gosec 安全扫描 | `gosec ./...` 无高危告警 |
| 代码审查 | 按 coding-style.md 检查：文件行数、函数行数、嵌套深度、不可变性、错误处理 |
| 安全审查 | 检查清单：无硬编码密钥、输入校验、SQL 注入防护（参数化查询）、XSS 防护、CSRF 防护、速率限制、错误消息不泄露敏感信息 |
| `.env.example` 最终核对 | 与代码中 `config.Load()` 校验的必填项一致 |
| `CLAUDE.md` 更新 | 反映最终项目状态 |
| `AGENTS.md` 更新 | 反映最终 agent 配置 |
| `README.md`（可选） | 项目说明、快速开始、命令速查 |

### 验收方案

```bash
# 1. 覆盖率门控
go test -race -cover ./...
# 输出 ≥90%

# 2. 静态分析
golangci-lint run ./...
# 零告警

# 3. Pre-commit
pre-commit run --all-files
# go-test-cover: Pass
# golangci-lint: Pass

# 4. 安全扫描
gosec ./...
# 无 HIGH/CRITICAL

# 5. CI 全绿
# GitHub Actions: go test cover ≥90% ✓ + golangci-lint ✓ + Newman smoke ✓
```

---

## 阶段依赖图

```
Phase 0 ─────────────────────────────────────────────────────────────
  │                                                                  │
  ├── Phase 1 (model)                                                │
  │     │                                                            │
  │     ├── Phase 3 (repository + migrations)                        │
  │     │     │                                                      │
  │     │     └── Phase 5 (service) ─────────────────────┐           │
  │     │           │                                     │           │
  │     │           ├── Phase 5c (medagent client)       │           │
  │     │           │     │                               │           │
  │     │           │     └── Phase 6 (adapter) ─────────┤           │
  │     │           │                                     │           │
  │     │           └── Phase 5a/b/d (patient/visit/workbench)       │
  │     │                                 │                           │
  │     └── Phase 2 (config + pkg/api)   │                           │
  │           │                           │                           │
  │           └── Phase 4 (middleware)    │                           │
  │                 │                     │                           │
  │                 └── Phase 7 (handler) ◄──────────────────────────┘
  │                       │
  │                       └── Phase 8 (integration + smoke tests)
  │                             │
  │                             └── Phase 9 (quality gates)
  │
  └── Phase 0 产物（Dockerfile, docker-compose, CI, Makefile）
      在各阶段持续更新
```

**注意**：
- Phase 1 和 Phase 2 可并行推进（无相互依赖）
- Phase 5c（medAgent client）和 Phase 6（adapter）紧密耦合，可合并为一个阶段实现
- Phase 0 的容器化/CI 产物在后续阶段持续更新（如 Dockerfile 的 `go build` 目标从空 main 变为完整服务）

---

## 每个阶段的测试策略

| Phase | 单元测试 | 集成测试 | 覆盖目标 |
| --- | --- | --- | --- |
| 0 | — | — | —（骨架搭建） |
| 1 | Model 序列化/反序列化 | — | ≥90% |
| 2 | Config 校验全边界 | — | ≥90% |
| 3 | — | Repository + testcontainers MySQL | ≥90% |
| 4 | httptest + Gin 测试模式 | — | ≥90% |
| 5 | Mock Repository + Mock medAgent Client | — | ≥90% |
| 6 | Fake medAgent Step | — | ≥90% |
| 7 | — | httptest + Mock Service | ≥90% |
| 8 | — | testcontainers 全链路 + Newman E2E | 核心流程 |
| 9 | — | CI 全量 | 总覆盖率 ≥90% |

---

## 风险与注意事项

| 风险 | 缓解措施 |
| --- | --- |
| medAgent 子模块 API 变更 | 锁定 submodule commit；Phase 5c 前阅读 `medAgent/docs/后端接入指南.md` 确认最新 API |
| testcontainers 需 Docker 环境 | CI 使用 `ubuntu-latest` + Docker；本地开发需 Docker Desktop/Docker Engine |
| 状态机复杂度（17 种内部态） | Phase 1 先定义完整状态转移表；Phase 5b 用表驱动测试覆盖全部转移 |
| SSE 流式传输调试困难 | Phase 7 先实现 `SSEWriter` 工具，用 `httptest` 的 `ResponseRecorder` 验证事件序列 |
| rest-api.md 与 medAgent 的边界差异 | 严格按照 STRUCTURE.md §6.5 边界表处理：治疗执行映射为 REFERRAL、急症恢复需后端显式支持、总计时由前端发起 |

---

## 预计文件统计（完成后）

| 目录 | 文件数（约） | 说明 |
| --- | --- | --- |
| `cmd/server/` | 1 | main.go |
| `internal/model/` | 8 | 全部模型 + 枚举 + 错误 |
| `internal/config/` | 3 | 配置加载 + env 解析 + 测试 |
| `internal/repository/` | 8 | 4 个接口 + 4 个 MySQL 实现 |
| `internal/service/` | 17 | patient(2) + visit(3) + workbench(10) + medagent(2) |
| `internal/handler/` | 7 | router + patient + visit + workbench + sse + middleware + test |
| `internal/middleware/` | 6 | auth + cors + logging + recovery + rate-limit + test |
| `internal/adapter/` | 4 | step_mapping + card_builder + timeline_builder + test |
| `internal/errors/` | 3 | api_error + codes + handler |
| `pkg/api/` | 3 | response + pagination + test |
| `db/migrations/` | 8 | 4 对 up/down |
| `tests/` | 5 | testutil(2) + newman(2) + seed(1) |
| 根目录 | 12 | go.mod, Makefile, Dockerfile, docker-compose.yml, .env.example, .golangci.yml, .pre-commit-config.yaml, .gitignore, CLAUDE.md, AGENTS.md 等 |
| `.github/workflows/` | 1 | ci.yml |

**总计：约 85–95 个文件**
