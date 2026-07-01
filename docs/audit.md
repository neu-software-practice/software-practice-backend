# AUDIT.md — 项目全面扫描审计报告

> 生成日期：2026-07-01  
> 审计方法：五组独立审计 + 对抗性交叉验证（11 agent，736K tokens，369 次工具调用）  
> 扫描范围：`internal/`、`pkg/`、`cmd/`、`tests/` 全部 Go 源文件（~90 文件）

---

## 一、概述

### 发现问题总数

| 严重程度 | 数量 |
|---------|------|
| **HIGH** | 1 |
| **MEDIUM** | 27 |
| **LOW** | 33 |
| **INFO** | 4 |
| **总计** | **65** |

### 按类别分布

| 类别 | 数量 |
|------|------|
| 简化机会（Simplification） | 18 |
| SPEC/STRUCTURE 合规性 | 10 |
| 测试设计缺陷 | 7 |
| 审计遗漏/盲区 | 8 |
| 正确性缺陷 | 22 |

---

## 二、简化机会（Simplification）

### MEDIUM

#### M1. `ValidAdminRoles` 是死代码 — `internal/model/admin.go:15-19`

- **当前实现**：定义了 `map[AdminRole]bool` 和 `AdminRoleSuperAdmin`/`AdminRoleAdmin`/`AdminRoleOperator` 常量，但在任何生产代码中均未被引用（仅测试文件使用这些常量）。`RequireAdminRole` 中间件直接使用字符串字面量 `"super_admin"` 和 `"admin"`。
- **修复方向**：删除 `ValidAdminRoles` 和未使用的 `AdminRole` 常量，或将中间件改为引用该映射，消除双源事实。
- **预期收益**：消除 ~15 行死代码，强制使用类型安全的管理员角色。

#### M2. `StepMappingTable` 及 `GetMapping` 是死代码 — `internal/adapter/step_mapping.go`

- **当前实现**：定义了完整的 `StepMapping` 结构体（含 `SSETypes`、`CardKind`、`IsTerminal`）和映射表，但没有任何生产代码路径调用 `GetMapping`。所有 SSE 事件通过 `chat.go` 中的 handler 直接发出。
- **修复方向**：删除整个 `step_mapping.go` 文件及其在 `adapter_test.go` 中的测试引用，或将映射表接入运行时路由。
- **预期收益**：消除约 80 行无人消费的死代码，降低开发者的认知负荷。

#### M3. `NewDBTest` 是死代码 — `tests/testutil/dbtest.go`

- **当前实现**：76 行的 `NewDBTest` 函数用于创建独立测试数据库，但**全代码库零调用方**。此外，其 DSN 拼接方式（`baseDSN + dbName`）在 `baseDSN` 包含查询参数时是有缺陷的。
- **修复方向**：删除 `tests/testutil/dbtest.go` 整个文件，或在测试基础设施中接入该函数。
- **预期收益**：消除 76 行误导性的死代码，消除一个隐蔽的 DSN 拼接 bug。

#### M4. `RunMigrationsWithGolangMigrate` 是死代码 — `tests/testutil/mysql_container.go`

- **当前实现**：23 行的 `RunMigrationsWithGolangMigrate` 函数完全未被调用。实际的迁移运行器是 `RunMigrations`（直接执行 `.up.sql` 文件）。
- **修复方向**：删除该函数。
- **预期收益**：消除 23 行死代码和一种不存在的迁移策略的幻觉。

#### M5. `middleware.GenerateAccessToken` / `GenerateAdminAccessToken` 是不必要的包装 — `internal/middleware/auth.go:85-87`, `admin_auth.go:96-98`

- **当前实现**：两个函数是单行委托：直接调用 `auth.GenerateAccessToken(...)`。唯一的调用方是 `cmd/jwtgen/main.go`，而服务层代码已经直接调用 `auth.*` 版本。
- **修复方向**：删除这两个包装函数，将 `cmd/jwtgen` 改为直接调用 `auth.GenerateAccessToken`（注意需同时更新 `middleware_test.go` 中引用它们的测试）。
- **预期收益**：消除不必要的间接层和依赖边（`jwtgen → middleware → auth` 简化为 `jwtgen → auth`）。

---

### LOW

#### L1. 重复的 `os.Unsetenv` 样板 — `internal/config/config_test.go:106-116`

- **当前实现**：每个子测试前重复 11 次 `os.Unsetenv`。
- **修复方向**：提取 `clearEnv(t *testing.T)` 辅助函数。
- **预期收益**：将 11 行样板压缩为 1 行。

#### L2. `TestTerminalReasons` 重复覆盖 — `internal/model/model_test.go:338-360`

- **当前实现**：完全重复了 `TestEnumConstants` 中 `TerminalReason` 段的测试（第 239-246 行），零增量覆盖。
- **修复方向**：删除该测试函数。
- **预期收益**：消除约 20 行无价值的重复测试代码。

#### L3. 手动 WHERE 子句组装 — `internal/repository/dashboard_mysql.go:147-153`

- **当前实现**：使用 `for/append` 循环手动拼接 WHERE 条件。
- **修复方向**：改为 `strings.Join(whereParts, " AND ")`。
- **预期收益**：提高可读性，无行为变化。

#### L4. `patient_mysql.go` 和 `visit_mysql.go` 定义相同的 scanner 接口

- **当前实现**：两个文件分别定义了结构相同的 `scanner`/`patientScanner` 接口。
- **修复方向**：移入共享的文档包或 `helpers.go`。
- **预期收益**：消除接口定义重复。

#### L5. `llm/client.go` 中多余的 `ctx.Err()` 检查 — `internal/llm/client.go:85-87`

- **当前实现**：方法入口处的 `ctx.Err()` 检查与 `http.NewRequestWithContext` 内部检查重复。
- **修复方向**：删除该方法入口检查。
- **预期收益**：消除 3 行冗余代码。

#### L6. `llm/client.go` 中 `Close()` 的不必要闭包包装 — `internal/llm/client.go:132`

- **当前实现**：`defer func() { _ = resp.Body.Close() }()`
- **修复方向**：改为 `defer resp.Body.Close()`
- **预期收益**：采用 Go 惯用模式。

#### L7. 查询 `machine_state` 列后丢弃 — `internal/repository/visit_mysql.go:88,96`

- **当前实现**：`SELECT ... machine_state ...` 将结果扫描到 `machineState` 变量后立即用 `_ = machineState` 丢弃。`VisitSessionSummary` 没有 `MachineState` 字段。
- **修复方向**：从 SELECT 列表中移除 `machine_state` 和对应的扫描目标。
- **预期收益**：减少每次列表查询的 I/O。

#### L8. `dashboard_mysql.go` 硬编码 `'' as birth_date` — `internal/repository/dashboard_mysql.go:96`

- **当前实现**：SELECT 中写死 `'' as birth_date`，因为对应的迁移（`000013_add_patient_birth_date`）从未提交。
- **修复方向**：提交迁移并改为使用实际的 `birth_date` 列，或删除此字段。
- **预期收益**：消除永远不会返回真实数据的伪列。

#### L9. `dbtest.go` 中的 `USE` 语句是死代码 — `tests/testutil/dbtest.go:39-41`

- **当前实现**：对 `baseDB` 执行 `USE <dbname>`，但 `baseDB` 仅用于创建/删除数据库，实际测试操作通过已包含数据库名的 `testDSN` 连接。
- **修复方向**：删除该 `USE` 语句。
- **预期收益**：消除 3 行不产生任何作用的代码。

#### L10. `mysql_container.go` 中多余的 `db.Ping()` — `tests/testutil/mysql_container.go:59-61`

- **当前实现**：重试循环在 `db.Ping()` 成功后 `break`，但紧随其后又调用一次 `db.Ping()`，要么冗余（break 后），要么同样失败（循环超时后）。
- **修复方向**：删除多余的 `db.Ping()`。
- **预期收益**：消除重复的 ping 操作。

#### L11. `CountPatientsSince` / `CountSessionsSince` 接受 `string` — `internal/repository/dashboard_repo.go:13,16`

- **当前实现**：方法签名使用 `since string`，将日期格式化责任推给调用方。
- **修复方向**：改为 `since time.Time`，将转换内聚在 repository 层。
- **预期收益**：更好的类型安全。

---

## 三、SPEC/STRUCTURE 合规性问题

### HIGH

#### C7. `handleDone` 中 REFERRAL 分支的无保护解引用 — `internal/service/workbench/chat.go:489`

- **不符合**：在同一方法中，第 503-504 行在访问 `result.Diagnosis.Name` 前检查了 `result.Diagnosis != nil`，但第 489 行没有。
- **当前实现**：第 489 行直接访问 `result.Diagnosis.Name`，若 `Diagnosis` 为 nil 则 panic。
- **修复方向**：在解引用前添加 nil 检查。

---

### MEDIUM

#### C1. `ApiResponse` 缺少 `Meta` 字段 — `pkg/api/response.go:4-8` vs `docs/STRUCTURE.md §6.2`

- **不符合**：STRUCTURE.md §6.2 定义响应信封为 `{success, data, error, meta}`，其中 `meta` 包含 `{"total": 100, "pageSize": 20}`。
- **当前实现**：`ApiResponse` 仅定义 `Success`, `Data`, `Error` 三个字段，分页元数据嵌入在 `Data` 内部。
- **修复方向**：在 `ApiResponse` 中添加 `Meta interface{} \`json:"meta,omitempty"\``，并更新 `SuccessResponse`/`ErrorResponse` 等构造函数。需评估前端实际消费方式后落地（`front-api.md`（真正的 API 契约）未提及 `meta`，因此严重性限于内部文档与实际实现不同步）。

#### C2. STRUCTURE.md 中路径不正确 — `docs/STRUCTURE.md §4`

- **不符合**：STRUCTURE.md §4 将 `response.go` 和 `pagination.go` 列在 `internal/testutil/` 下。
- **当前实现**：两个文件实际位于 `pkg/api/response.go` 和 `pkg/api/pagination.go`。
- **修复方向**：更新 STRUCTURE.md 中的路径。

#### C3. 硬编码的限流值 — `cmd/server/main.go:102`

- **不符合**：与项目配置原则（§8："配置应通过 .env 和 config.Config"）不一致。
- **当前实现**：`middleware.RateLimitMiddleware(10, 20)` 使用硬编码值，未通过配置系统外部化。
- **修复方向**：在 `config.Config` 中添加 `RateLimitRate` 和 `RateLimitBurst` 字段，在 `main.go` 中读取。

#### C4. 缺少优雅关闭 — `cmd/server/main.go:110`

- **不符合**：生产级服务应支持信号驱动的优雅关闭。
- **当前实现**：`engine.Run(addr)` 阻塞直到进程被杀死，SIGTERM/Ctrl-C 立即终止，丢弃所有进行中的连接。
- **修复方向**：添加 `signal.Notify` + `http.Server.Shutdown()`。

#### C5. REFERRAL 状态的 MachineState 使用错误 — `internal/service/workbench/chat.go:497`

- **不符合**：`MachineStateToStatus` 映射表（`state_machine.go:127`）规定 `Completed` 映射到 `VisitStatusCompleted`，但 REFERRAL 分支应使用 `VisitStatusTransferred`。
- **当前实现**：`handleDone` 中 REFERRAL 分支设置 `State: VisitMachineStateCompleted` 但设置 `Status: VisitStatusTransferred`，状态不一致。
- **修复方向**：将 MachineState 改为 `VisitMachineStateTransferred`，使其与 `VisitStatusTransferred` 一致。

#### C6. `DismissEmergency` 违反状态机终端规则 — `internal/service/workbench/vitals.go:124`

- **不符合**：`state_machine.go:88-90` 规定所有终端状态禁止转换，`IsTerminal(Terminated)` 返回 `true`。
- **当前实现**：`DismissEmergency` 从 `VisitStatusEmergencyTerminated` / `VisitMachineStateEmergencyPending` 转换为 `Chatting`。
- **修复方向**：要么在状态机中记录此路径为有意为之的逃生口（更新 `AllowedTransitions` 添加 `EmergencyPending → Chatting`），要么移除该功能。

#### C8. CORS 配置中未使用的 `ServerMode` 字段 — `internal/middleware/cors.go`

- **不符合**：STRUCTURE.md §8.1 要求生产环境 `AllowedOrigins` 不可为 `*`，但运行时 CORS 中间件未读取 `ServerMode`。
- **当前实现**：`CORSConfig.ServerMode` 从 `cfg.ServerMode` 设置但从未被 `CORSMiddleware` 读取。该约束仅在 `config.Load()` 验证中实施。
- **修复方向**：要么在 `CORSMiddleware` 中添加运行时检查，要么从 `CORSConfig` 中移除未使用的 `ServerMode` 字段。

#### C9. 前端接口文档有 17 个状态但代码中有 18 个 — `internal/model/enums.go:24-42` vs `docs/front-api.md §3.2`

- **不符合**：`front-api.md` §3.2 记录了 17 个 `VisitMachineState` 值，排除了 `transferred`。
- **当前实现**：代码定义了 18 个常量，包括 `VisitMachineStateTransferred`。
- **修复方向**：要么将 `transferred` 加入前端文档，要么在文档中注明其为后端内部过渡状态。

#### C10. `DismissEmergency` 和部分 handler 遗漏 `session.UpdatedAt` 更新

- **当前实现**：仅 `SendMessage` 和 `handleAsk` 在持久化前更新 `session.UpdatedAt`。其他 14 个 handler 均省略此操作。
- **修复方向**：所有修改会话的 handler 都应在持久化前更新 `UpdatedAt`。

---

## 四、测试设计问题

### MEDIUM

#### T1. 缺少测试级数据库隔离 — `tests/testutil/dbtest.go`

- **问题**：`NewDBTest` 全代码库零调用方（见 M3）。这意味着集成测试连接到 `baseDSN` 指向的某个共享数据库，违反 STRUCTURE.md §7.2 和 CI 统一 MySQL 的约定。
- **影响**：共享数据库状态可能导致测试间相互干扰和 CI 不稳定。
- **修复方向**：将 `NewDBTest` 接入测试基础设施，或确认当前测试是否使用了其他隔离机制并更新文档。

#### T2. 仅 4/10 的 Repository 接口有共享 Mock — `internal/testutil/mocks.go`

- **问题**：共享 mock 文件仅为 `PatientRepository`、`VisitRepository`、`TimelineRepository`、`FlowCardRepository` 提供了 mock。以下 7 个接口缺少共享 mock：`AddressRepository`、`UserRepository`、`RefreshTokenRepository`、`AdminRepository`、`AdminRefreshTokenRepository`、`DashboardRepository`、`SettingsRepository`。
- **影响**：涉及这些接口的 Service 层测试必须各自重新实现 mock，导致代码重复。
- **修复方向**：为所有 Repository 接口生成或编写共享 mock，或使用 mock 生成工具（如 mockgen）。

#### T3. 迁移文件未排序 — `tests/testutil/mysql_container.go:82`

- **问题**：`RunMigrations` 使用 `filepath.Glob` 查找迁移文件，但 `filepath.Glob` **不保证排序顺序**。此外，有两个迁移共享 `000008` 前缀，即使字母排序也存在歧义。
- **修复方向**：在执行前通过 `sort.Strings(files)` 对 glob 结果排序，并确保迁移前缀唯一。

#### T4. `TestErrorCodes` 仅覆盖 25 个错误码中的 11 个 — `internal/errors/errors_test.go:197-216`

- **问题**：仅测试了 11 个错误码常量，遗漏了 `CodeAuthPhoneExists`、`CodeRateLimited`、`CodeLLMUnavailable`、`CodeAdminInvalidCredentials`、`CodeAddressNotFound` 等 14 个。
- **影响**：新错误码若被意外定义为空字符串 `""`，无法被该测试捕获。
- **修复方向**：增加循环反射或显式枚举所有 25 个常量。

---

### LOW

#### T5. `handleDone` 中时间线追加操作的错误被忽略 — `internal/service/workbench/chat.go:462-465`

- **问题**：已完成就诊卡的创建通过了错误检查，但后续的时间线追加操作错误被忽略。
- **修复方向**：统一错误处理逻辑，不忽略任何步骤的错误。

#### T6. 多个 handler 的 `switch err { case sentinel: }` 未使用 `errors.Is` — `internal/handler/workbench_handler.go:209-218`

- **问题**：`SubmitFulfillment` 等 handler 使用值比较 `switch err { case model.ErrCardNotFound: }`，而该文件的其他所有错误检查都使用 `errors.Is`/`errors.As`。如果 Service 层将来包装这些 sentinel 错误，值比较会静默失败，返回通用内部错误而非恰当的状态码。
- **修复方向**：统一改为 `switch { case errors.Is(err, model.ErrCardNotFound): ... }`。

#### T7. `TokenExpired` 检查未使用 `errors.Is` — `internal/middleware/auth.go:104-108`

- **问题**：`TokenExpired` 使用 `strings.Contains(err.Error(), "token is expired")`，而不是 `errors.Is(err, jwt.ErrTokenExpired)`。当前有效但若 jwt 库更改错误措辞则静默失效。
- **修复方向**：改为 `errors.Is(err, jwt.ErrTokenExpired)`，正确解包 `%w` 错误链。

---

## 五、遗漏问题（审计盲区——对抗性验证发现）

### HIGH

#### H1. 缺少测试级数据库隔离（同 T1）

**严重**：这是测试基础设施的根本性缺失，直接违反 STRUCTURE.md 和 CI 约定，可能导致不可重现的测试失败。

---

### MEDIUM

#### H3. `timeline_mysql.go` `UpdateStatus` 返回裸错误 — `internal/repository/timeline_mysql.go:132`

- **问题**：这是整个 repository 包中唯一不包装错误的方法，与其他 50+ 个方法不一致。调用方无法区分裸 SQL 驱动错误和业务错误。
- **修复方向**：添加 `fmt.Errorf("update timeline status: %w", err)` 包装。

#### H5. `handleDone` 中 RESULT 诊断的多余赋值 — `internal/service/workbench/chat.go:489-506`

- **问题**：`handleDone` 在 switch 内部（REFERRAL 分支）设置 `session.Summary.Diagnosis`，之后又对所有情况设置同一字段（第 503-506 行）。外层赋值冗余覆盖内层值。
- **修复方向**：删除第 489 行 REFERRAL 分支内的冗余 `session.Summary.Diagnosis` 赋值，仅保留外层赋值。

#### H6. `handleOK` 绕过映射表 — `internal/service/workbench/chat.go:518-527`

- **问题**：`handleOK` 直接发出 `"done"` SSE 事件，但 `step_mapping.go` 中 `StepOK` 声明 `SSETypes: []string{}`。若映射表未来接入运行时路径，`StepOK` 将停止发出事件，行为改变。
- **修复方向**：要么更新映射表使其与运行时行为一致（`StepOK` → `SSETypes: ["done"]`），要么删除映射表。

---

### LOW

#### L14. `visit_mysql.go` 中静默丢弃的 `json.Marshal` 错误 — `internal/repository/visit_mysql.go:23,157`

- **问题**：`summaryJSON, _ := json.Marshal(visit.Summary)` — 与 `flow_card_mysql.go` 中检查 marshal 错误的模式不一致。虽然 `VisitSummary` 理论上不会 marshal 失败，但模式不一致。
- **修复方向**：统一错误处理模式（检查并包装 marshal 错误），或在注释中说明为何安全丢弃。

#### L15. `SubmitPayment` 实验室路径静默丢弃错误 — `internal/service/workbench/payment.go:77`

- **问题**：`_ = s.SubmitLabResults(...)` — 如果实验室结果处理失败，调用方仍收到成功响应。
- **修复方向**：捕获并处理 `SubmitLabResults` 的错误（至少记录日志），或将其返回值纳入 `result`。

#### L19. `StreamEvents` 使用不一致的错误信封 — `internal/handler/sse_handler.go:94-106`

- **问题**：使用 `c.JSON(http.StatusInternalServerError, gin.H{"error": "streaming not supported"})` 而非项目标准的 `apperrors.WriteError(c, ...)`。
- **修复方向**：统一使用 `apperrors.WriteError`。

#### L20. `SubmitLabResults` 创建的 card 未持久化 — `internal/service/workbench/lab.go:140-164`

- **问题**：`SubmitLabResults` 构造了 `FlowCard` 但从未调用 `flowCardRepo.Create(ctx, card)`，仅追加了时间线条目。
- **修复方向**：添加 `s.flowCardRepo.Create(ctx, card)` 调用。

---

## 六、优先修复路线图

### 第一优先级：正确性缺陷（立即修复）

| # | 发现 | 文件 | 严重度 | 估计工作量 |
|---|------|------|--------|-----------|
| P0-1 | `handleDone` REFERRAL 分支无保护解引用（panic 风险） | `internal/service/workbench/chat.go:489` | HIGH | 1 行 |
| P0-2 | `DismissEmergency` 终端状态违反规则 | `internal/service/workbench/vitals.go:124` | MEDIUM | 评估：移除功能或更新状态机 |
| P1-1 | REFERRAL 状态 MachineState/Status 不一致 | `internal/service/workbench/chat.go:497` | MEDIUM | 1 行 |
| P1-2 | `SubmitLabResults` card 未持久化 | `internal/service/workbench/lab.go:140-164` | MEDIUM | 1 行 |
| P1-3 | `handleDone` RESULT 诊断多余赋值 | `internal/service/workbench/chat.go:489-506` | MEDIUM | 删除冗余行 |

### 第二优先级：合规性与结构一致性（本周完成）

| # | 发现 | 文件 | 严重度 | 估计工作量 |
|---|------|------|--------|-----------|
| P2-1 | `ApiResponse` 添加 `Meta` 字段或更新文档 | `pkg/api/response.go`, `docs/STRUCTURE.md` | MEDIUM | 1 天（含前端协调） |
| P2-2 | 迁移文件排序（`sort.Strings`） | `tests/testutil/mysql_container.go:82` | MEDIUM | 3 行 |
| P2-3 | `TestErrorCodes` 覆盖全部 25 个错误码 | `internal/errors/errors_test.go:197-216` | MEDIUM | 0.5 天 |
| P2-4 | `TokenExpired` 改为 `errors.Is(err, jwt.ErrTokenExpired)` | `internal/middleware/auth.go:104-108` | LOW | 3 行 |
| P2-5 | handler `switch err` 改为 `errors.Is` | `internal/handler/workbench_handler.go:209-218` | LOW | 5 行 |
| P2-6 | 14 个 handler 添加 `session.UpdatedAt` 更新 | 多个 handler 文件 | MEDIUM | 0.5 天 |
| P2-7 | `timeline_mysql.go` `UpdateStatus` 错误包装 | `internal/repository/timeline_mysql.go:132` | MEDIUM | 3 行 |
| P2-8 | 缺少测试级数据库隔离 | `tests/testutil/dbtest.go` | HIGH | 2-3 天 |
| P2-9 | `handleOK` 绕过映射表（一致性修复） | `internal/service/workbench/chat.go:518-527` | MEDIUM | 1 行 |

### 第三优先级：简化与清理（两周内完成）

| # | 发现 | 文件 | 严重度 | 估计工作量 |
|---|------|------|--------|-----------|
| P3-1 | 删除 `ValidAdminRoles` 死代码 | `internal/model/admin.go` | MEDIUM | 1 天 |
| P3-2 | 删除 `StepMappingTable` 死代码 | `internal/adapter/step_mapping.go` | MEDIUM | 0.5 天 |
| P3-3 | 删除 `NewDBTest` 死代码 | `tests/testutil/dbtest.go` | MEDIUM | 0.5 天 |
| P3-4 | 删除 `RunMigrationsWithGolangMigrate` 死代码 | `tests/testutil/mysql_container.go` | MEDIUM | 0.3 天 |
| P3-5 | 删除 `middleware.GenerateAccessToken` 包装函数 | `internal/middleware/auth.go`, `admin_auth.go`, `cmd/jwtgen/main.go` | MEDIUM | 0.5 天 |
| P3-6 | 硬编码限流值外部化到 Config | `cmd/server/main.go`, `internal/config/config.go` | MEDIUM | 0.5 天 |
| P3-7 | 添加 `signal.Notify` 优雅关闭 | `cmd/server/main.go` | MEDIUM | 1 天 |
| P3-8 | `CountPatientsSince` 等 string → `time.Time` | `internal/repository/dashboard_repo.go` | LOW | 0.3 天 |
| P3-9 | `visit_mysql.go` 移除冗余 `machine_state` 列 | `internal/repository/visit_mysql.go:88,96` | LOW | 0.3 天 |
| P3-10 | `visit_mysql.go` 处理 `json.Marshal` 错误 | `internal/repository/visit_mysql.go:23,157` | LOW | 0.3 天 |
| P3-11 | `SubmitPayment` 处理 `SubmitLabResults` 错误 | `internal/service/workbench/payment.go:77` | LOW | 0.3 天 |
| P3-12 | `dashboard_mysql.go` 移除硬编码 `'' as birth_date` | `internal/repository/dashboard_mysql.go:96` | LOW | 0.2 天 |
| P3-13 | 为缺失的 7 个 Repository 接口添加共享 Mock | `internal/testutil/mocks.go` | MEDIUM | 1-2 天 |
| P3-14 | `StreamEvents` 统一使用 `apperrors.WriteError` | `internal/handler/sse_handler.go:94-106` | LOW | 3 行 |

### 第四优先级：文档与风格优化（持续改进）

| # | 发现 | 文件 | 严重度 | 估计工作量 |
|---|------|------|--------|-----------|
| P4-1 | STRUCTURE.md 路径修正 | `docs/STRUCTURE.md §4` | LOW | 0.2 天 |
| P4-2 | 注释 17 → 18 值修正 | `internal/model/model_test.go:220` | LOW | 1 行 |
| P4-3 | 移除未使用的 `CORSConfig.ServerMode` | `internal/middleware/cors.go` | LOW | 0.3 天 |
| P4-4 | 重复的 scanner 接口合并 | `internal/repository/patient_mysql.go`, `visit_mysql.go` | LOW | 0.3 天 |
| P4-5 | `config_test.go` 样板代码抽取 `clearEnv` | `internal/config/config_test.go` | LOW | 0.3 天 |
| P4-6 | `getEnv` vs `os.Getenv` 统一 | `internal/config/config.go` | LOW | 0.3 天 |
| P4-7 | `TestTerminalReasons` 重复测试删除 | `internal/model/model_test.go` | LOW | 0.1 天 |
| P4-8 | WHERE 子句 `strings.Join` 简化 | `internal/repository/dashboard_mysql.go` | LOW | 0.1 天 |
| P4-9 | `llm/client.go` 移除多余 `ctx.Err()` | `internal/llm/client.go:85-87` | LOW | 0.1 天 |
| P4-10 | `llm/client.go` `defer` 闭包简化 | `internal/llm/client.go:132` | LOW | 0.1 天 |
| P4-11 | 数据库查询格式对齐 | 多个 `_mysql.go` 文件 | LOW | 0.5 天 |
| P4-12 | `front-api.md` 与代码 `VisitMachineState` 枚举同步 | `docs/front-api.md`, `internal/model/enums.go` | LOW | 0.3 天 |

---

## 七、各阶段总估计工作量

| 阶段 | 发现问题数 | 估计总工作量 |
|------|-----------|-------------|
| 第一优先级（正确性） | 5 | 1-2 天 |
| 第二优先级（合规性） | 9 | 3-5 天 |
| 第三优先级（简化清理） | 14 | 4-7 天 |
| 第四优先级（文档风格） | 12 | 2-4 天 |
| **总计** | **40** | **10-18 天** |

---

> **注**：本报告由 5 组独立审计 agent + 5 组对抗性验证 agent + 1 综合 agent 协作生成，基于代码静态分析。部分发现（如 `front-api.md` 同步、Meta 字段添加）需与前端团队协调确认后实施。
