<!--
  审查日期: 2026-07-01
  审查范围: 后端实现 vs docs/front-api.md (2026-06-30)
  审查方法: 5 阶段 Workflow — Scan → Extract → Cross-Reference → Verify → Synthesize
  代理数量: 38 | 工具调用: 530 | Token: 1,699,920
-->
# API 实现差距分析报告

## 总体评估摘要

| 指标 | 数值 |
|---|---|
| 前端接口文档 `front-api.md` 已记录 endpoint | **45** 个 |
| 后端已实现 endpoint（`router.go`） | **47** 个 |
| 文档-实现完全匹配 endpoint | **45** (100%) |
| 已实现但未记录 endpoint | **2** (suspend, health) |
| 已记录但未实现 endpoint | **0** |
| 整体 endpoint 完成度 | **100%**（按文档基线） |
| 文档缺陷（缺 endpoint/枚举/字段） | **9** 处 |
| 实现缺陷（bug/安全/性能） | **8** 处 |
| 代码-文档 schema 不匹配 | **7** 处 |

**结论**：所有文档中的 endpoint 均已完全实现。但存在 2 个 CRITICAL 级运行时缺陷、6 个 HIGH 级安全和功能缺陷，以及大量文档与实现之间的不一致。主要根因：v9 空闲挂起特性（commit `15e8ae4`）的 endpoint、枚举值和字段尚未合并入 `front-api.md`。

---

## 按严重程度排序的发现

---

### CRITICAL

#### C1. SSE Heartbeat goroutine 从未启动

- **描述**：`Heartbeat` 方法在 `internal/handler/sse_handler.go:72-87` 完整实现（每 30 秒发送 keepalive 注释帧），但在三个 SSE endpoint（`StreamAssistantMessage`、`AskLockedQuestion`、`StreamConsultationReply`）中均未被调用。所有生产 SSE 连接一旦经过反向代理（Nginx/ALB），将在代理空闲超时后断开。
- **受影响文件**：`internal/handler/sse_handler.go:72-87`、`internal/handler/workbench_handler.go:101-138`（line 115 创建 SSEWriter，未启动心跳）、第 424-453 行、第 455-484 行
- **具体差距**：SSEWriter 创建后缺少 `go w.Heartbeat(ctx, ...)` 调用
- **建议修复**：在三处 SSE handler 的 SSEWriter 创建之后各增加 `go w.Heartbeat(ctx, cancel, ...)` 启动 goroutine

#### C2. RecoveryMiddleware 绕过统一响应信封

- **描述**：`recovery.go:21-25` 使用 `c.AbortWithStatusJSON(http.StatusInternalServerError, apperrors.NewApiError(...))` 直接返回裸 JSON `{"code":"INTERNAL_ERROR","message":"internal server error","status":500}`，而没有经过 `apperrors.WriteError()` 的 `{"success":false,"data":null,"error":{...}}` 标准信封。`docs/STRUCTURE.md §6.2` 明确要求**所有**非流式响应使用统一信封。
- **受影响文件**：`internal/middleware/recovery.go:21-25`
- **具体差距**：Recovery 中间件是全局 panic 的最后防线，其响应格式与全系统不一致
- **建议修复**：将第 21 行替换为 `apperrors.WriteError(c, apperrors.NewApiError(http.StatusInternalServerError, apperrors.CodeInternalError, "internal server error"))`

---

### HIGH

#### H1. 患者端与管理后台共用 JWT 签名密钥

- **描述**：`router.go:77` 的 `AuthMiddleware(cfg.JWTSecret)` 与 `router.go:137` 的 `AdminAuthMiddleware(cfg.JWTSecret)` 注入的是同一个 `cfg.JWTSecret`。虽然中间件通过 payload 字段（`patientId` vs `role`）做逻辑隔离，但密钥共用意味着密钥泄露即可伪造任意角色的令牌。`front-api.md:1010` 明确要求"密钥、载荷、过期策略与患者端完全分离"。
- **受影响文件**：`internal/handler/router.go:77,137`、`internal/config/config.go:17`（仅有单个 JWTSecret 字段）、`internal/service/auth/service.go:228`、`internal/service/admin/service.go:77,151,296`
- **具体差距**：后端无 `AdminJWTSecret` 配置字段；两个服务注入相同密钥
- **建议修复**：在 Config 中增加 `AdminJWTSecret` 字段（同样满足 JWT>=32 字节要求），路由层对 admin 中间件和 admin auth service 注入该新密钥

#### H2. GET /admin/sessions/:id 返回的 VisitSession 不包含 timeline

- **描述**：`AdminHandler.GetSessionDetail`（`admin_handler.go:156`）调用 `svc.GetSessionDetail`（`admin/service.go:257`），仅执行 `visitRepo.FindByID` 查询 `visits` 表。文档（`front-api.md:1214`）明确要求返回"完整 `VisitSession` 对象，含完整 timeline"。`GetSnapshot`（`visit/service.go:154-172`）已有 timeline 组装逻辑可复用。
- **受影响文件**：`internal/handler/admin_handler.go:156`、`internal/service/admin/service.go:257`、`internal/repository/visit_mysql.go:51`
- **具体差距**：admin session detail 端点缺少 timelineRepo.ListBySession 调用及 timeline 字段组装
- **建议修复**：在 `GetSessionDetail` 中增加 `timelineRepo.ListBySession` 查询，将 timeline 附加到响应中

#### H3. 管理后台患者列表 birthDate 始终返回空字符串

- **描述**：`dashboard_mysql.go:97` 的 SQL 查询硬编码了 `'' as birth_date` 占位符。`front-api.md:1141` 将 `birthDate` 定义为 `string (YYYY-MM-DD)`（必填），但所有患者返回的 birthDate 均为空字符串。
- **受影响文件**：`internal/repository/dashboard_mysql.go:97`
- **具体差距**：SQL 中写死占位符，未从 patients 表实际查询生日字段
- **建议修复**：将 `dashboard_mysql.go:97` 的 `'' as birth_date` 替换为实际列，如 `COALESCE(p.birth_date, '') as birth_date`，并通过 JOIN 或子查询关联 patients 表

#### H4. CodeTimeout 已定义但不可达——无任何中间件或 handler 能产生该错误码

- **描述**：`CodeTimeout = "TIMEOUT"`（`codes.go:13`）定义在错误码表中，`front-api.md §2.2` 将其列为预期错误码，`§2.3` 将 408 列为可重试状态码。但整个后端没有任何 middleware 或 handler 引用 `CodeTimeout`，`internal/middleware/` 下也不存在超时中间件。这是死代码。
- **受影响文件**：`internal/errors/codes.go:13`
- **具体差距**：文档承诺了 TIMEOUT/408 错误语义，后端永远不会产生
- **建议修复**：在 router 或 middleware 层增加超时中间件（如 `gin.Context.Timeout` 或自定义 middleware）产生 408/CodeTimeout，或从文档中移除该错误码

#### H5. TITLE_ALREADY_EXISTS 和 LLM_UNAVAILABLE 错误码在生产代码中永远不可达

- **描述**：`CodeTitleAlreadyExists`（`codes.go:23`）和 `CodeLLMUnavailable`（`codes.go:24`）定义在错误码表中。但 `GenerateTitle` 服务在标题已存在时以幂等方式静默返回已有标题（`title.go:35-37`），在 LLM 不可用时降级使用 chiefComplaint 截断（`title.go:44-51`）。没有任何代码路径能产生这两个错误码。
- **受影响文件**：`internal/errors/codes.go:23-24`、`internal/service/workbench/title.go:35-51`
- **具体差距**：文档列出了这些错误码作为预期的 API 契约，但后端永远不会返回
- **建议修复**：要么修改服务层，在重复调用或 LLM 不可用时实际返回这些错误；要么从文档中移除

#### H6. INVALID_STATE 错误码未在文档中列出

- **描述**：`CodeInvalidState = "INVALID_STATE"`（`codes.go:26`）在 `SuspendVisit` handler（`visit_handler.go:160-164`）中活跃使用（当会话状态不允许挂起时返回），但 `front-api.md §2.2` 错误码表（69-96 行）未列出该错误码。
- **受影响文件**：`internal/errors/codes.go:26`、`internal/handler/visit_handler.go:160-164`
- **具体差距**：后端实际返回的错误码在文档中没有对应条目
- **建议修复**：在 front-api.md §2.2 错误码表中增加 `INVALID_STATE` 条目（HTTP 409 或 400）

---

### MEDIUM

#### M1. POST /visits/:sessionId/suspend endpoint 未在文档 endpoint 表中记录

- **描述**：路由注册于 `router.go:89`，handler 完整实现（`visit_handler.go:139-173`），服务逻辑包括状态转换、流式消息中断、系统事件追加（`service.go:192-245`）。但 `front-api.md §4` endpoint 表（185-231 行）缺少该路由。
- **具体差距**：v9 空闲挂起功能的唯一业务 endpoint 未在文档端点上体现
- **建议修复**：在 §4 endpoint 表中增加一行：`POST /visits/:sessionId/suspend`，写入时间线标记为"是"

#### M2. VisitStatus 枚举缺少 `suspended` 值

- **描述**：后端 `internal/model/enums.go:15` 定义了 `VisitStatusSuspended = "suspended"`，共 11 个值。文档 `front-api.md:137` 声称"10 值"且仅列出 10 个值，缺少 `suspended`。
- **具体差距**：文档取值集合和计数均落后于代码
- **建议修复**：在 `front-api.md §3.1` 枚举列表中增加 `suspended`，将计数更新为 11

#### M3. VisitMachineState 枚举缺少 `suspended` 值

- **描述**：后端 `internal/model/enums.go:38` 定义了 `VisitMachineStateSuspended = "suspended"`，共 19 个值。状态机中 13 处引用该值（`state_machine.go` 转移表/映射）。文档 `front-api.md:141` 声称"18 值"且仅列出 18 个值。
- **具体差距**：文档取值集合落后于代码；v9 状态机重度依赖此枚举
- **建议修复**：在 `front-api.md §3.2` 枚举列表中增加 `suspended`，计数更新为 19

#### M4. SystemEventType 枚举缺少 `session_suspended` 值

- **描述**：后端 `internal/model/enums.go:141` 定义了 `SystemEventTypeSessionSuspended = "session_suspended"`，共 9 个值。挂起服务在 `service.go:233` 中使用。文档 `front-api.md:173` 声称"8 值"且仅列出 8 个值。
- **具体差距**：文档取值集合落后于代码
- **建议修复**：在 `front-api.md §3.10` 枚举列表中增加 `session_suspended`，计数更新为 9

#### M5. VisitSession 和 VisitSessionSummary 缺少 `lastActivityAt` 字段

- **描述**：后端 `model/visit.go:20`（VisitSession）和 `model/visit.go:50`（VisitSessionSummary）均包含 `LastActivityAt *time.Time json:"lastActivityAt,omitempty"`。文档 §5.2 字段表（440-459 行）和第 434 行的摘要描述均未提及该字段。
- **具体差距**：响应中会出现该字段，但前端 schema 未定义它
- **建议修复**：在文档 §5.2 的 VisitSession 字段表中增加 `lastActivityAt`（ISO8601，可选）；在 VisitSessionSummary 行内描述中增加 `lastActivityAt?`

#### M6. VisitSnapshot.Readonly 使用 `bool` + `omitempty` 与文档的 `literal true` 冲突

- **描述**：`model/visit.go:61` 的 `Readonly bool json:"readonly,omitempty"` 在值为 `false`（Go 零值）时会在 JSON 中消失。文档（`front-api.md:475`）定义 `readonly` 为 `literal true`（必填）。
- **具体差距**：模型允许 Readonly=false 甚至字段缺失，违反文档契约
- **建议修复**：从模型中移除该字段，在序列化层（如自定义 JSON marshal 或 response wrapper）硬编码 `"readonly": true`

#### M7. medication_fulfillment FlowCard 缺少 `deliveryAddress` 文档

- **描述**：`model/flow_card.go:91` 包含 `DeliveryAddress *DeliveryAddress json:"deliveryAddress,omitempty"`。文档 §6.3 medication_fulfillment 字段列表（932 行）未提及 `deliveryAddress`。
- **具体差距**：文档未覆盖实际存在的字段
- **建议修复**：在文档 §6.3 medication_fulfillment 字段列表中增加 `deliveryAddress?`（指向已有的 DeliveryAddressSummary/§5.14）

#### M8. system_event 和 terminal 时间线条目的 `title` 字段：omitempty vs 文档标记为必填

- **描述**：`model/timeline.go:28` 的 `Title string json:"title,omitempty"` 由 both kinds 共享。文档 §6.1（894-895 行）的 system_event 和 terminal `title` 均未加 `?`，暗示是必填。当 title 为空时字段会消失。
- **具体差距**：模型与文档对 title 的可选性认知不一致
- **建议修复**：移除 Title 的 omitempty 标签（改为 `json:"title"`），因为 terminal 事件始终应有标题（如"问诊完成"、"紧急转诊"）

#### M9. PatientProfile.updatedAt 带有 omitempty，但文档标记为必填

- **描述**：`model/patient.go:20` 的 `UpdatedAt time.Time json:"updatedAt,omitempty"` 在零值时省略。文档 §5.1（272 行）将 `updatedAt` 标记为"是"（必填）。
- **具体差距**：理论上数据库记录始终有非零 updatedAt，但模型定义与文档不一致
- **建议修复**：移除 omitempty，改为 `json:"updatedAt"` 以匹配文档

#### M10. Token 盗用撤销采用物理删除，丢失审计追溯能力

- **描述**：`refresh_token_mysql.go:67-75` 和 `admin_refresh_token_mysql.go:67-76` 在检测到 token 重用后执行 `DELETE FROM ... WHERE user_id = ?`。物理删除无法保留撤销时间、原因等审计信息。同时，删除范围是"该用户全部 token"而非仅被重用的 token 链，会误伤该用户的其他活跃会话。
- **受影响文件**：`internal/repository/refresh_token_mysql.go:67-75`、`internal/repository/admin_refresh_token_mysql.go:67-76`、`internal/service/auth/service.go:122`
- **建议修复**：改为 `UPDATE SET revoked_at=NOW(), revoked_reason='token_reuse_detected'` 软删除；在服务层区分"撤销当前链"与"撤销该用户全部 token"两种场景

#### M11. 管理后台 GET /admin/patients/:id 缺少地址列表

- **描述**：文档（`front-api.md:1157`）要求返回 `PatientProfile` "含过敏史、慢病、长期用药、地址列表"。但 `PatientProfile` 模型（`model/patient.go:8-21`）没有 `Addresses` 字段，admin 服务的 `GetPatientProfile`（`admin/service.go:233`）原样返回 `PatientProfile` 而未附加任何地址数据。
- **具体差距**：基础 PatientProfile schema 本身也不含地址，但 admin 端点的文档说明额外要求了地址列表
- **建议修复**：为 admin 端点创建一个扩展 DTO（或在 PatientProfile 中添加 `Addresses` 字段），在 `GetPatientProfile` 中调用 `addressRepo.ListByPatient` 并将结果附加到响应中

#### M12. pageSize 查询参数缺少上限校验

- **描述**：`ParseQueryInt`（`middleware.go:37-47`）仅校验 `n >= 1`，无上限。文档对 `/admin/patients`（1121 行）和 `/admin/sessions`（1177 行）均明确规定了 pageSize 上限为 100。客户端传入 `pageSize=10000` 会生成 `LIMIT 10000`，可能引发数据库性能问题。
- **具体差距**：缺少基于文档上限的防滥用校验
- **建议修复**：扩展 `ParseQueryInt` 支持 `max` 参数，在 admin handler 调用时传入 `100`

#### M13. EMERGENCY 严重程度在 terminal 事件中硬编码为 `critical`

- **描述**：`chat.go:376` 的 `Severity: string(model.EmergencySeverityCritical)` 将所有紧急事件的 severity 固定为 `critical`。`front-api.md:907` 允许 `suspected | critical` 两个值。medAgent 当前不提供 severity 信息，因此代码使用 `critical` 作为安全默认值，但这丢失了 `suspected` 的可能性。
- **具体差距**：medAgent 合约不提供 severity 数据，但代码始终输出 `critical`
- **建议修复**：要么扩展 medAgent 接口以支持 severity 字段，要么在文档中注明此场景下后端始终输出 `critical`

#### M14. suggestedDepartment 在 terminal 时间线中始终为空

- **描述**：`BuildTerminalTimelineItem`（`timeline_builder.go:51-63`）没有 department 参数。7 个调用点（chat.go:381, 479; fulfillment.go:104; vitals.go:103; treatment.go:97, 198; exit.go:52）均未设置 `SuggestedDepartment`。字段存在于模型（`timeline.go:33`）中但始终为 nil。
- **具体差距**：文档 §6.1（895 行）将 `suggestedDepartment?` 列为 terminal 的可选字段，但实际响应中永远不会出现
- **建议修复**：为 `BuildTerminalTimelineItem` 增加 department 参数，在合适的调用点（如诊断、处置决策）注入科室信息

#### M15. 支付卡金额字段使用 *float64 + omitempty（schema 与文档呈现方式冲突）

- **描述**：`model/flow_card.go:62-65` 对 `totalAmount`、`insuranceAmount`、`selfPayAmount` 使用 `*float64 json:"...,omitempty"`，对 `paymentStatus` 使用 omitempty。这是判别联合体（discriminated union）的有意设计——非支付卡类型时这些字段不出现。文档（928 行）将支付卡字段展示为始终存在（未加 `?`）。服务层 `BuildPaymentCard`（`card_builder.go:136-147`）始终填充这四个字段，因此运行时无实际影响。
- **具体差距**：模型的有意设计与文档的"始终存在"呈现方式存在语义不一致
- **建议修复**：保持模型不变（非指针的 `float64` 会使非支付卡响应中出现 `"totalAmount":0`，违反判别联合体约定）。在服务层通过契约测试确保这些字段始终被填充，并在文档中加注说明

#### M16. BillingLineItem.quantity 使用 `*int` 而文档写为 `number | null`

- **描述**：`model/billing.go:21` 使用 `Quantity *int json:"quantity,omitempty"`。系统中所有数量以整数值表示，领域角度 `int` 是正确的。`front-api.md:812` 写为 `number | null`。文档内部也存在矛盾：同一位置说明"仅 >1 时返回"（省略语义）与 `number | null`（null 语义）不一致。
- **具体差距**：文档类型描述不精确；文档内部的 null 与省略语义矛盾
- **建议修复**：在文档中将类型改为 `integer`，并明确表达省略语义（"数量 >1 时返回；否则该字段不出现"）而非 null 语义

---

### LOW

#### L1. InterruptedBy 枚举缺少 `idle` 值

- **描述**：`internal/model/enums.go:348` 定义 `InterruptedByIdle = "idle"`，并由空闲挂起服务使用（`service.go:211-212`）。文档 §6.1（892 行）将 interruptedBy 合法值列为 `emergency | timeout | exit`，缺少 `idle`。
- **建议修复**：在文档 §6.1 interruptedBy 值列表中添加 `idle`

#### L2. 速率限制中间件覆盖整个 /auth 路由组（超出文档范围）

- **描述**：`router.go:63-64` 将 `RateLimitMiddleware(5.0/60.0, 5)` 应用于整个 `/auth` 组。`front-api.md:732` 的安全要求表格指定该速率限制仅应用于 `/auth/login` 和 `/auth/register`。多标签页并发 refresh 可能因限速导致 token 过期后无法刷新。
- **建议修复**：将 login/register 拆分到独立子路由组（如 `authGroup.POST("/login", ...)` 不挂载组级别限速），仅对这两个端点应用 `Use(RateLimitMiddleware(...))`

#### L3. AdminPatientItem.gender 缺少枚举约束（前端 Zod 可能拒绝）

- **描述**：`model/admin_queries.go:17` 的 `Gender string json:"gender"` 纯字符串无约束。文档（1140 行）定义为 `male | female | unknown`，排除了 `other`。数据库患者性别为 `other` 时，管理员列表响应将包含 `"other"`，前端 Zod schema 会拒绝该响应。
- **建议修复**：在 admin 查询层添加转换逻辑，将 `"other"` 映射为 `"unknown"`；或更新文档允许 `other`

#### L4. NETWORK_ERROR / UNKNOWN_ERROR 在 Go 中定义但属于前端传输层错误码

- **描述**：`CodeUnknownError`（`codes.go:8`）和 `CodeNetworkError`（`codes.go:9`）定义在 Go 代码中，但没有任何后端 middlewares/handler 产生它们。`front-api.md:72` 将 UNKNOWN_ERROR 标注为"未知异常兜底"（客户端侧），NETWORK_ERROR 也是前端传输层产物。后端定义这些常量属于死代码。
- **建议修复**：从 Go 代码中移除这两个常量（保留在前端），或添加超时中间件等生产路径使用它们

#### L5. /api/health 健康检查端点未在文档中记录且响应格式不统一

- **描述**：`router.go:56-58` 注册了 `GET /api/health`，返回 `{"status":"ok"}` 而非标准信封格式。这是运维 endpoint，负载均衡器可正常解析当前格式。
- **建议修复**：可选择性地在文档中记录作为 devops 端点；可选地统一为信封格式

#### L6. 缺乏验证错误码生产正确性的功能测试

- **描述**：`errors_test.go:197-230` 仅验证所有常量字符串非空。没有测试验证在特定条件下 handler/service 返回正确的错误码（如 `SuspendVisit` 返回 `INVALID_STATE`、LLM 失败时返回什么等）。
- **建议修复**：在 handler/service 层增加集成测试，使用 mock 依赖来触发特定错误条件并验证错误码

---

## 修复路线图

### 第一批次：CRITICAL（应立即修复，预计工时：1-2 天）

| 优先级 | 问题 | 建议修复 | 预估工时 |
|--------|------|---------|---------|
| P0 | C1. SSE Heartbeat 未启动 | 在 3 个 SSE handler 中启动心跳 goroutine | 0.5 天 |
| P0 | C2. RecoveryMiddleware 绕过统一信封 | 替换为 `apperrors.WriteError` | 0.5 天 |

### 第二批次：HIGH（应在下一迭代修复，预计工时：3-5 天）

| 优先级 | 问题 | 建议修复 | 预估工时 |
|--------|------|---------|---------|
| P1 | H1. JWT 密钥共用 | 新增 `AdminJWTSecret` 配置字段，路由层分离 | 1 天 |
| P1 | H2. admin session 缺 timeline | 在 GetSessionDetail 中增加 timeline 查询与组装 | 0.5 天 |
| P1 | H3. admin 患者列表 birthDate 为空 | 替换 SQL 占位符为实际列查询 | 0.5 天 |
| P1 | H4. CodeTimeout 不可达 | 添加超时中间件或从文档中移除 | 0.5 天 |
| P1 | H5. TITLE_ALREADY_EXISTS / LLM_UNAVAILABLE 不可达 | 修改服务层传播错误或更新文档 | 0.5 天 |
| P1 | H6. INVALID_STATE 缺失于文档 | 添加到 front-api.md §2.2 错误码表 | 0.5 天 |

### 第三批次：MEDIUM（应在当前迭代完成，预计工时：3-5 天）

| 优先级 | 问题 | 建议修复 | 预估工时 |
|--------|------|---------|---------|
| P2 | M1-M5. v9 文档缺失（suspend endpoint + 3 枚举 + lastActivityAt） | 将 `docs/front-api-v9.md` 内容合并入 `front-api.md`，更新对应章节（§3.1, §3.2, §3.10, §4, §5.2） | 1 天 |
| P2 | M6. Readonly bool + omitempty | 修改模型和序列化层 | 0.5 天 |
| P2 | M7-M9. schema 文档缺失（deliveryAddress, title, updatedAt） | 更新 front-api.md 对应章节 | 0.5 天 |
| P2 | M10. Token 撤销物理删除 | 改为软删除，区分撤销范围 | 1 天 |
| P2 | M11. admin 患者详情缺地址列表 | 扩展 DTO，附加地址数据 | 0.5 天 |
| P2 | M12. pageSize 无上限校验 | 扩展 ParseQueryInt 支持 max 参数 | 0.5 天 |
| P2 | M13-M14. emergency severity / suggestedDepartment | 扩展 medAgent 合约或补充文档说明 | 1 天 |
| P2 | M15-M16. 支付卡 / quantity 文档不精确 | 更新文档类型描述 | 0.5 天 |

### 第四批次：LOW（在以上问题解决后处理，预计工时：1-2 天）

| 优先级 | 问题 | 建议修复 | 预估工时 |
|--------|------|---------|---------|
| P3 | L1. InterruptedBy 缺 idle | 更新 front-api.md §6.1 | 0.25 天 |
| P3 | L2. 速率限制覆盖范围过大 | 重构路由组隔离 | 0.5 天 |
| P3 | L3. AdminPatientItem.gender 无约束 | 添加转换映射 | 0.25 天 |
| P3 | L4. 客户端错误码在 Go 中为死代码 | 清理 Go 代码 | 0.25 天 |
| P3 | L5. /api/health 未记录 | 可选更新文档 | 0.25 天 |
| P3 | L6. 错误码生产缺乏测试 | 增加集成/契约测试 | 0.5 天 |

---

**总预估工时**：约 8-14 天（取决于并发工作量和资源分配）

**关键依赖**：
- M13/M14 涉及 medAgent 接口扩展，需协调 medAgent 团队
- H1（JWT 密钥分离）需确认部署环境变量和密钥轮换策略
- M1-M5（v9 文档合并）应安排在第一批次之后立即完成
