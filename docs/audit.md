# NEUHIS Agent 后端审计报告

> 审计日期：2026-06-30 | 审计方式：7 Agent 并行 Workflow
> 审计范围：123 Go 源文件 + 20 测试文件（排除 medAgent 子模块）
> 审计维度：代码简化 / STRUCTURE 一致性 / API 合约一致性 / 测试质量 / 数据流与依赖

---

## A. 总体评估

项目基础扎实：testcontainers 集成测试基础设施完善、分层架构清晰、核心诊疗流程功能完整。但存在明显的**架构漂移**——16 个未记录的包/文件偏离了 STRUCTURE.md 设计文档，2 处**层级违规**（依赖方向反转），API 合约有 3 个高危泄露（内部状态暴露给前端），测试覆盖率严重不均匀（handler 仅 37.8%，远低于 90% 要求）。

关键指标：
- 总文件数：123（非 medAgent Go 源文件）+ 20（测试文件）
- 未记录在 STRUCTURE.md 的包/文件：16 个
- 架构层级违规：2 处
- API 合约违反：3 处高危
- 测试覆盖率未达标包：5 个

---

## B. 关键问题（必须修复）

### B.1 VisitSession 泄露内部机器状态到 API 响应 — `json` tag 违规

- **文件**：`internal/model/visit.go:14,27`
- **严重度**：🔴 Critical
- **规范参考**：`front-api.md §5.2` — VisitSession 有 18 个已声明字段；`machineState` 和 `medagentSessionId` 不在其中
- **现状**：`MachineState`（第 14 行）带 `json:"machineState"` tag，`MedAgentSessionID`（第 27 行）带 `json:"medagentSessionId,omitempty"` tag，每次序列化 VisitSession 都会暴露内部状态机值和 medAgent 会话 ID
- **修复**：将两个字段的 JSON tag 改为 `json:"-"`，或为 API 响应创建独立的 DTO 结构体

### B.2 PatientProfile 暴露非规范字段

- **文件**：`internal/model/patient.go:18-19`
- **严重度**：🔴 Critical
- **规范参考**：`front-api.md §5.1` — PatientProfile 有 10 个已定义字段；`medicalHistory` 和 `createdAt` 不在其中
- **现状**：结构体包含 `MedicalHistory []string json:"medicalHistory"`（第 18 行）和 `CreatedAt time.Time json:"createdAt,omitempty"`（第 19 行），两者在所有 PatientProfile 响应中序列化。`medicalHistory` 仅应存在于 PatientContext/PriorVisit
- **修复**：从 PatientProfile 结构体移除 MedicalHistory 和 CreatedAt，或使用单独的响应 DTO

### B.3 SSE 错误事件违反 API 合约格式

- **文件**：`internal/handler/sse_handler.go:54-62`
- **严重度**：🔴 Critical
- **规范参考**：`front-api.md §6.2` — error 类型 SSE 事件必须包含 `error: ApiError` 对象（code/message/status/details/retriable）
- **现状**：`SSEWriter.WriteError` 将错误文本放入 `message` 字符串字段。`AssistantStreamEvent.Error *SSEEventError` 字段从未填充。事件产出 `{"type":"error","message":"some error"}` 而非规范的 `{"type":"error","error":{"code":"...","message":"...","status":...}}`
- **修复**：让 `WriteError` 接受 `*errors.ApiError` 并设置 `event.Error = &SSEEventError{...}`

### B.4 Handler 包覆盖率 37.8%（要求 ≥90%）

- **文件**：`internal/handler/handler_test.go`
- **严重度**：🔴 Critical
- **规范参考**：`SPEC.md` + `STRUCTURE.md §7.1`
- **现状**：AuthHandler、AdminHandler、MedicalOrderHandler 零 HTTP 层测试。WorkbenchHandler 大部分端点缺少测试（SubmitFulfillment、SubmitTreatmentExecution、ReportVitals、DismissEmergency、ToggleTimer、AskLockedQuestion、StreamConsultationReply、GenerateTitle、ClassifyIntent、ExitVisit、SubmitPayment、StreamAssistantMessage、SubmitLabDecision、SubmitLabResults）
- **修复**：按 `TestVisitHandler_CreateSession` / `TestPatientHandler_VerifyIdentity` 模式添加 `httptest` 测试

### B.5 Service 层导入 Middleware（层级违规——依赖方向反转）

- **文件**：`internal/service/admin/service.go:17`
- **严重度**：🔴 Critical
- **规范参考**：`STRUCTURE.md §3.1` — "Middleware → Config"；`§5.2` — Service 依赖 Repository/medAgent/Model
- **现状**：`admin` service 导入 `internal/middleware` 调用 `middleware.GenerateAdminAccessToken`，反转了文档化的依赖方向
- **修复**：将 `GenerateAdminAccessToken` 提取到 `internal/auth` 或 `pkg/auth` 共享包

### B.6 `transferred` 状态的状态机不一致

- **文件**：`internal/model/enums.go:15`、`state_machine.go:10-14,112-130`、`internal/service/workbench/chat.go:482`
- **严重度**：🟠 High
- **规范参考**：`STRUCTURE.md §6.4` — 4 种终端状态包括 "transferred"
- **现状**：`VisitStatusTransferred` 在 enums 中已定义，但在 `MachineStateToStatus` 中无映射条目，`TerminalStates` 中也缺失。chat.go 处理转诊时设置 `session.Status = VisitStatusTransferred`，但机器状态为 `completed`，造成不一致：`GetStatusForState("completed")` 返回 `"completed"` 而非 `"transferred"`
- **修复**：添加 `VisitMachineStateTransferred`、补全映射、添加到 TerminalStates

### B.7 Workbench Service 使用具体 medAgent Client 类型（不可 Mock）

- **文件**：`internal/service/workbench/service.go:23`
- **严重度**：🟠 High
- **规范参考**：Go 最佳实践 "接受接口，返回结构体"
- **现状**：workbench `Service` 持有具体类型 `*medagent.Client` 而非接口，导致无法在无真实 medAgent 服务器的情况下进行单元测试。同文件中的 `LLMClient` 字段已正确抽象为接口
- **修复**：在 workbench 包定义 `medAgentClient` 接口，将接口存入 Service 结构体

---

## C. 简化机会

### C.1 高影响

| # | 文件 | 类别 | 发现 | 建议 |
|---|------|------|------|------|
| 1 | `internal/middleware/auth.go` | 死代码 | 5 个导出函数零生产调用者（IsAuthenticated、SetPatientID、CurrentPatient、RespondWithJSON、WriteJSONError） | 从生产代码移除 4 个；将 SetPatientID 移至 testutil |
| 2 | `internal/middleware/auth.go` + `admin_auth.go` | 代码重复 | ~55 行几乎相同的 JWT 提取/解析/签名验证/错误处理代码 | 提取共享 `parseJWT(c, jwtSecret) (*jwt.MapClaims, error)` 函数 |
| 3 | `internal/repository/visit_mysql.go:101-151` + `timeline_mysql.go:84-138` | 代码重复 | 完全相同的游标分页实现（if/else 分支、pageSize+1 技巧、hasMore 裁剪、ISO 格式光标字符串） | 提取 `PaginateCursor[T]` 到 `internal/repository/pagination.go` |
| 4 | `internal/handler/workbench_handler.go` | 代码重复 | 10 个 handler 方法定义几乎相同的行内输入结构体，手动逐字段复制到 service 类型 | 在 model 或 request 包定义共享输入类型；使用 `BindAndSetSessionID` 辅助函数 |

### C.2 中影响

| # | 文件 | 类别 | 发现 | 建议 |
|---|------|------|------|------|
| 5 | `internal/handler/middleware.go:55` + `pkg/api/response.go:21` | 死代码 | `WriteSuccessWithMeta` 和 `SuccessResponseWithMeta` 零生产调用者 | 移除两者及 ApiResponse 中未使用的 `Meta` 字段 |
| 6 | `internal/handler/admin_handler.go:214` | 冗余抽象 | `WritePageResponse[T]` 仅调用 `c.JSON` 包装 `api.SuccessResponse`——仅 2 个调用者 | 移除并用 `WriteSuccess(c, http.StatusOK, result)` 替换 |
| 7 | `internal/model/patient.go:26` | 数据冗余 | PatientContext 中的 MedicalHistory/Allergies/LongTermMedications 字段与 PatientProfile 重复 | 移除顶层字段；消费者应从 `ctx.Patient.*` 读取 |
| 8 | `internal/service/visit/service.go:137` + `workbench/service.go:52` | 代码重复 | 两个 service 中完全相同的 `GetSession` 透传 `visitRepo.FindByID` | 让 workbench 委托给 visit.Service.GetSession |

### C.3 低影响

| # | 文件 | 类别 | 发现 | 建议 |
|---|------|------|------|------|
| 9 | `cmd/server/main.go:113-114` | 死代码 | `log.Fatalf` 后跟不可达的 `os.Exit(1)` | 移除不可达的 `os.Exit(1)` |
| 10 | 多个 MySQL 仓库文件 | 重复模式 | 5 个仓库独立初始化 CreatedAt/UpdatedAt 时间戳 | 考虑 `touchTimestamps` 辅助函数或数据库级别的 DEFAULT CURRENT_TIMESTAMP |

---

## D. STRUCTURE/SPEC 偏离

### D.1 需更新文档（STRUCTURE.md / front-api.md）

| # | 文件 | 严重度 | 违反章节 | 差距 | 建议 |
|---|------|--------|----------|------|------|
| 1 | `front-api.md` | 🟠 High | §4 端点清单 | `GET /api/medical-orders`（router.go:119）未在文档中 | 添加到 front-api.md 或移除路由 |
| 2 | `internal/service/address/`、`admin/`、`auth/`、`billing/`、`medicalorder/` | 🟡 Medium | STRUCTURE §4 | 5 个未记录的 service 子包 | 添加到 STRUCTURE.md 包树 |
| 3 | `internal/model/address.go`、`billing.go`、`medical_order.go`、`admin.go`、`admin_queries.go`、`user.go`、`settings.go`、`helpers.go` | 🟡 Medium | STRUCTURE §4 | 8 个未记录的 model 文件 | 添加到 STRUCTURE.md model 文件清单 |
| 4 | `internal/middleware/admin_auth.go` + `admin_auth_test.go` | 🟠 High | STRUCTURE §3.1 / §4 | 第 6 种中间件类型未记录；token 生成逻辑分散在 middleware 和 service | 添加到 STRUCTURE.md 或将 token 生成移至共享包 |
| 5 | `internal/llm/` 包 | 🟠 High | STRUCTURE §4 / §5.2 | 未记录的包被 workbench 导入 | 添加到 STRUCTURE.md 或移入 medagent/pkg |
| 6 | `internal/service/workbench/consult.go`、`title.go`、`title_test.go` | 🟠 High | STRUCTURE §4 | 3 个未记录的 workbench 文件 | 添加到 STRUCTURE.md 文件清单 |
| 7 | `internal/config/env.go` | 🟠 High | STRUCTURE §4 | 文档声明但文件不存在；env 解析内联在 config.go 中 | 创建 env.go 或更新 STRUCTURE.md |
| 8 | `internal/service/medagent/` 文件清单 | 🟢 Low | STRUCTURE §4 | 列出 `adapter.go`/`adapter_test.go`，但实际文件为 `client.go`/`types.go` | 更新 STRUCTURE.md |
| 9 | `internal/testutil/` | 🟡 Medium | STRUCTURE §4 | 仅记录了 `tests/testutil/`；`internal/testutil/` 及 mocks.go 未记录 | 添加到 STRUCTURE.md 或合并 |
| 10 | go.mod 版本 | 🟢 Low | STRUCTURE §2.1 | 文档写 "Go 1.22+"，go.mod 指定 `1.22.0` | 对齐文档与实际版本约束 |
| 11 | workbench title.go | 🟢 Low | STRUCTURE §2.3 / §4 | 标题生成绕过 medAgent 引擎；未记录的架构决策 | 在 STRUCTURE.md 中记录此 LLM 客户端复用决策 |

### D.2 需修复代码

| # | 文件 | 严重度 | 规范章节 | 差距 | 建议 |
|---|------|--------|----------|------|------|
| 12 | `internal/adapter/step_mapping.go` | 🟡 Medium | STRUCTURE §6.3 | StepDone 的 SecondaryCardKind 为 `treatment_plan` 而非 `completed_visit`/`advice_only` | 更新 STRUCTURE.md 或修复 step_mapping.go；为 chat.go handleDone 添加单元测试 |
| 13 | `internal/model/enums.go` + `state_machine.go` | 🟠 High | STRUCTURE §6.4 | `transferred` 在 MachineStateToStatus 和 TerminalStates 中缺失 | 添加 transferred 机器状态；修复映射 |
| 14 | `internal/model/visit.go:14,27` | 🔴 Critical | `front-api.md §5.2` | 内部字段在 JSON 中泄露 | 标记为 `json:"-"` |
| 15 | `internal/model/patient.go:18-19` | 🔴 Critical | `front-api.md §5.1` | 非规范字段被序列化 | 移除或使用 DTO |
| 16 | `internal/handler/sse_handler.go:54-62` | 🔴 Critical | `front-api.md §6.2` | SSE 错误格式错误（字符串 message 而非 SSEEventError 对象） | 填充 `.Error` 字段 |
| 17 | `internal/service/admin/service.go:17` | 🔴 Critical | STRUCTURE §3.1 / §5.2 | Service 导入 middleware | 将 token 生成移至共享包 |
| 18 | `internal/service/workbench/title.go:10` | 🟠 High | STRUCTURE §3.1 | Service 导入 `internal/errors`（HTTP 错误码） | 使用 model sentinel 错误 |
| 19 | `internal/handler/router.go` | 🟢 Low | 测试最佳实践 | Handler 构造函数接受具体 service 类型而非接口 | 在 handler 包定义小型接口 |

---

## E. 测试缺口

### E.1 覆盖率未达标（低于 90%）

| 包 | 当前覆盖率 | 目标 | 差距说明 |
|----|-----------|------|----------|
| `internal/handler` | **37.8%** | 90% | AuthHandler、AdminHandler、MedicalOrderHandler 零 HTTP 层测试 |
| `internal/repository` | **56.6%** | 90% | Dashboard、Settings、Admin、AdminRefreshToken 仓库零集成测试 |
| `internal/service/address` | 88.0% | 90% | 少量未覆盖分支 |
| `internal/service/workbench` | 89.6% | 90% | 少量未覆盖分支 |
| `pkg/api` | 87.5% | 90% | 小包，直接补齐 |

### E.2 缺失的测试套件

- **`internal/service/medagent/client.go`**：HTTP 客户端零专用测试。Adapter 测试使用 medagent 类型，但不覆盖 HTTP 请求/响应处理、错误场景或超时
- **Model 层**：address.go、admin.go、admin_queries.go、billing.go、medical_order.go、settings.go、user.go 无直接序列化/验证测试

### E.3 测试设计问题

1. **Mock 重复定义**：每个 service 测试包在 8+ 个包中重复定义完全相同的 mock 仓库（mockPatientRepo、mockVisitRepo 等）。共享的 `internal/testutil/mocks.go` 已提供可复用的 mock，但 service 测试未使用它们
2. **仓库隔离模型**：仓库测试在单个 MySQL 容器上跨子测试共享，而非 `STRUCTURE.md §7.1` 规定的 "per-test 独立数据库"
3. **MedAgent 客户端不可 mock**：Workbench service 持有具体类型 `*medagent.Client`（非接口），使 service 层测试需要真实的 medAgent 服务器
4. **CI 覆盖率门控未对齐**：`ci.yml` 仅对 3 个 service 包执行 90%，总门控为 75%，而非 SPEC.md 规定的全包 90%

---

## F. 数据流与架构违规

### F.1 层级违规（文档化依赖方向被反转）

| 文件 | 方向 | 违规 | 影响 |
|------|------|------|------|
| `internal/service/admin/service.go:17` | Service → Middleware | 导入 `internal/middleware` 调用 `GenerateAdminAccessToken` | Token 生成逻辑耦合到 HTTP 传输层；middleware 签名的任何更改都会破坏 service |
| `internal/service/workbench/title.go:10` | Service → HTTP errors | 导入 `internal/errors` 并在业务逻辑中创建 `apperrors.NewNotFoundError()` 等 | Service 层产出 HTTP 错误码而非领域错误；handler 无法独立重映射错误 |

### F.2 跨层耦合

| 文件 | 耦合 | 影响 |
|------|------|------|
| `internal/adapter/card_builder.go` | 导入 `medagent "internal/service/medagent"` 获取类型 | Adapter 依赖 service 层，在 workbench service 和 adapter 之间产生双向依赖 |
| `internal/model/` | 领域实体与请求/响应 DTO 及验证逻辑混合在同一包中 | API 格式变更影响领域逻辑；JSON tag 将内部表示耦合到外部格式 |

### F.3 不可变性违规

| 文件 | 违规 |
|------|------|
| `internal/service/address/service.go:105-125` | `UpdateAddress` 在写入仓库前原地修改 `addr` 领域对象（`addr.Name = *input.Name` 等） |
| `internal/service/workbench/chat.go` | 多个方法直接修改 `session`（如 `session.AskRound++`） |

### F.4 错误处理缺陷

| 文件 | 问题 | 影响 |
|------|------|------|
| `internal/service/medagent/client.go:52-57` | 返回 `fmt.Errorf("medagent session not found")` 等，未包装 sentinel 错误 | Handler/service 代码无法使用 `errors.Is()` 区分 medAgent 错误类型；所有 medAgent 错误变为通用 500 |

### F.5 请求 DTO 不一致

| 文件 | 问题 |
|------|------|
| `internal/handler/workbench_handler.go` | 部分端点定义本地输入结构体，部分使用 `model.*Input` ——请求 DTO 分散在 handler 和 model 之间，无明确规则 |

---

## G. Quick Wins（低工作量高收益）

| # | 文件 | 修复 | 预计耗时 | 影响 |
|---|------|------|----------|------|
| 1 | `internal/model/visit.go:14,27` | `json:"machineState"` → `json:"-"`；`json:"medagentSessionId,omitempty"` → `json:"-"` | 5 分钟 | 🔴 停止在每个 API 响应中泄露内部状态值和 medAgent 会话 ID |
| 2 | `internal/model/patient.go:18-19` | 从 PatientProfile 移除 MedicalHistory 和 CreatedAt | 10 分钟 | 🔴 使 PatientProfile 符合规范，防止数据泄露 |
| 3 | `cmd/server/main.go:114` | 移除不可达的 `os.Exit(1)` | 2 分钟 | 🟢 消除死代码 |
| 4 | `internal/handler/sse_handler.go:54-62` | WriteError 接受 `*errors.ApiError` 并填充 `event.Error` 字段 | 20 分钟 | 🟠 修复所有流式端点的 SSE 错误合约合规性 |
| 5 | `internal/handler/middleware.go` + `pkg/api/response.go` | 移除 `WriteSuccessWithMeta` 和 `SuccessResponseWithMeta` | 5 分钟 | 🟡 消除死 API 表面积 |

---

# 重构方案

## H.1 概述

基于以上审计发现，制定分阶段重构方案。按优先级排序为 4 个阶段，总计约 **40-50 工作小时**。

## H.2 Phase 1：止血修复（预计 4 小时）

**目标**：修复所有 Critical 级别问题（数据泄露、合约违反、层级违规）

| 任务 | 文件 | 预计耗时 | 说明 |
|------|------|----------|------|
| 1.1 | `internal/model/visit.go` | 0.5h | 将 MachineState、MedAgentSessionID 标记为 `json:"-"`；验证所有序列化点 |
| 1.2 | `internal/model/patient.go` | 0.5h | 从 PatientProfile 移除 MedicalHistory、CreatedAt；检查所有消费方 |
| 1.3 | `internal/handler/sse_handler.go` | 1h | 重写 WriteError 接受 ApiError；填充 SSEEventError；更新所有调用方 |
| 1.4 | `internal/service/admin/service.go` | 1h | 创建 `internal/auth/token.go`（GenerateAccessToken、GenerateAdminAccessToken）；更新 admin service + middleware 导入 |
| 1.5 | `internal/service/workbench/title.go` | 0.5h | 移除 `internal/errors` 导入；改用 model sentinel 错误 |
| 1.6 | `internal/model/enums.go` + `state_machine.go` | 0.5h | 添加 VisitMachineStateTransferred；补全 MachineStateToStatus 和 TerminalStates |

**验收标准**：
- `go build ./...` 通过
- 现有测试全部通过
- VisitSession JSON 输出不含 machineState/medagentSessionId
- admin service 不再导入 middleware 包

## H.3 Phase 2：消除重复与死代码（预计 6 小时）

**目标**：清理死代码、提取共享工具、减少重复

| 任务 | 文件 | 预计耗时 | 说明 |
|------|------|----------|------|
| 2.1 | `internal/middleware/auth.go` | 0.5h | 移除 IsAuthenticated、CurrentPatient、RespondWithJSON、WriteJSONError |
| 2.2 | `internal/middleware/auth.go` + `admin_auth.go` | 1.5h | 提取 `parseJWT()` 共享函数；两个 middleware 均调用之 |
| 2.3 | `internal/repository/` 新建 `pagination.go` | 2h | 实现泛型 `PaginateCursor[T]`；重构 visit_mysql.go 和 timeline_mysql.go |
| 2.4 | `internal/handler/workbench_handler.go` | 1.5h | 统一请求 DTO 定义（在 handler 包或 model 包）；消除 10 处重复的行内结构体 |
| 2.5 | `cmd/server/main.go` + 死代码清理 | 0.5h | 移除不可达 os.Exit(1)、WriteSuccessWithMeta、SuccessResponseWithMeta、WritePageResponse |

**验收标准**：
- `golangci-lint run ./...` 零告警
- 重复代码行数减少 ≥200 行
- 提取的共享函数有单元测试覆盖

## H.4 Phase 3：测试覆盖率补齐（预计 20 小时）

**目标**：所有包覆盖率达到 ≥90%

### H.4.1 Handler 层（预计 10h）

| 子任务 | 文件 | 预计耗时 |
|--------|------|----------|
| 3.1a | `handler_test.go` — AuthHandler（Register/Login/Refresh/Logout） | 2h |
| 3.1b | `handler_test.go` — AdminHandler（Login/Logout/Refresh/ListPatients/ListSessions） | 2h |
| 3.1c | `handler_test.go` — MedicalOrderHandler（ListMedicalOrders） | 1h |
| 3.1d | `handler_test.go` — WorkbenchHandler 缺失端点（Fulfillment/Treatment/Vitals/Exit/Timer/Dismiss/LockQuestion/Consult/Title/Classify） | 5h |

### H.4.2 Repository 层（预计 6h）

| 子任务 | 文件 | 预计耗时 |
|--------|------|----------|
| 3.2a | `repository_test.go` — Dashboard 仓库集成测试（CountPatients/CountSessions/ListPatients/ListSessions） | 2h |
| 3.2b | `repository_test.go` — Settings 仓库集成测试（Get/Update） | 1h |
| 3.2c | `repository_test.go` — Admin + AdminRefreshToken 仓库集成测试 | 2h |
| 3.2d | 重构现有仓库测试使用 `NewDBTest` per-test 隔离 | 1h |

### H.4.3 Service 层 + 其他（预计 4h）

| 子任务 | 文件 | 预计耗时 |
|--------|------|----------|
| 3.3a | `service/address/service_test.go` — 补齐至 90% | 1h |
| 3.3b | `service/workbench/service_test.go` — 补齐至 90% | 1h |
| 3.3c | `service/medagent/client_test.go` — 新建，使用 httptest.NewServer | 1.5h |
| 3.3d | `pkg/api/response_test.go` — 补齐至 90% | 0.5h |

### H.4.4 Model 层序列化测试（预计 2h）

| 子任务 | 文件 | 预计耗时 |
|--------|------|----------|
| 3.4a | `model/model_test.go` — 为 address/admin/billing/medical_order/settings/user 类型添加 JSON 序列化/反序列化测试 | 2h |

**验收标准**：
- `go test -race -cover ./...` 总覆盖率 ≥90%
- 每包覆盖率 ≥90%
- CI 中 coverage gate 更新至 90%

## H.5 Phase 4：架构对齐与文档更新（预计 10 小时）

**目标**：消除层级违规、统一接口设计、更新文档

### H.5.1 架构修复（预计 6h）

| 任务 | 文件 | 预计耗时 | 说明 |
|------|------|----------|------|
| 4.1a | `internal/service/workbench/service.go` | 2h | 定义 `medAgentClient` 接口；重构 workbench service 持有接口 |
| 4.1b | `internal/service/medagent/` → `internal/model/` | 2h | 将 medAgent 类型（Step、StepKind、Result 等）移至 `internal/model/` 消除 adapter↔service 循环依赖 |
| 4.1c | `internal/service/medagent/client.go` | 1h | 定义 sentinel 错误（ErrMedAgentSessionNotFound 等）；使用 `%w` 包装 |
| 4.1d | `internal/handler/router.go` | 1h | Handler 构造函数改为接受接口；定义小型 service 接口 |

### H.5.2 文档更新（预计 4h）

| 任务 | 预计耗时 | 说明 |
|------|----------|------|
| 4.2a | 更新 `docs/STRUCTURE.md` | 2h | 补充 16 个未记录包/文件；更新依赖图；更新 middleware 清单；修正 config/env.go 引用；修正 medagent 文件清单 |
| 4.2b | 更新 `docs/front-api.md` | 0.5h | 添加 GET /medical-orders 端点文档 |
| 4.2c | 更新 `ci.yml` | 1h | 全包覆盖率门控 ≥90%（替换当前 75% 总门控和仅 3 个 service 包的 90% 门控） |
| 4.2d | 更新 `CLAUDE.md` / `AGENTS.md` | 0.5h | 反映最终项目状态 |

### H.5.3 不可变性修复（预计 2h）

| 任务 | 文件 | 预计耗时 |
|------|------|----------|
| 4.3a | `internal/service/address/service.go` | 1h | UpdateAddress 替换原地修改为函数式更新模式 |
| 4.3b | `internal/service/workbench/chat.go` | 1h | session 修改替换为不可变更新模式 |

**验收标准**：
- 无可跨层反向依赖
- STRUCTURE.md 与实际代码一致
- CI 90% 门控生效

## H.6 时间线总结

```
Phase 1: ████ (4h)   止血修复
Phase 2: ██████ (6h)   消除重复与死代码
Phase 3: ████████████████████ (20h)  测试覆盖率补齐
Phase 4: ██████████ (10h)  架构对齐与文档更新
─────────────────────────────────
Total:   ████████████████████████████████████████ (40-50h)
```

## H.7 优先级矩阵

```
高影响 │  Phase 1 (止血)    │  Phase 3 (测试)
       │  B.1-B.7           │  E.1 覆盖率补齐
       │  G.1-G.4           │
───────┼─────────────────────┼─────────────────────
低影响 │  Phase 2 (清理)    │  Phase 4 (架构)
       │  C.1-C.10          │  F.1-F.5
       │  G.5               │  D.1-D.19
       └─────────────────────┴─────────────────────
         低工作量              高工作量
```

---

## 附录：审计方法

- **工具**：7 个并行 Agent 通过 Workflow 编排
- **Agent 角色**：代码简化专家 / 架构一致性审计师 / API 合约审计师 / 测试质量审计师 / 数据流审计师 / 综合报告师
- **分析方法**：静态代码阅读 + 结构化输出 + 交叉验证
- **读取文件数**：273 次工具调用，覆盖全部 123 个源文件和 20 个测试文件
- **基准文档**：`docs/SPEC.md`、`docs/STRUCTURE.md`、`docs/front-api.md`、`docs/PLAN.md`
