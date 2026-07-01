# API 实现与文档漂移审计报告

> **审计日期**: 2026-07-01  
> **审计范围**: `docs/rest-api.md` + patches v2-v9 vs Go 后端实现  
> **审计方法**: 多 Agent 交叉比对 + 对抗性验证  
> **基线文档**: `docs/rest-api.md` (639 行, 23 端点) + 8 个增量 patch (v2-v9, 合计 23 端点), 合并共 46 端点  
> **实现代码**: `internal/handler/`, `internal/model/`, `internal/service/`, `internal/middleware/`, `internal/errors/`

---

## 执行摘要

对 46 个文档化 API 端点与 Go 后端实现进行了 6 维度全面比对。经对抗性验证，共发现 **30 个确认真实漂移**，其中：

| 严重程度 | 数量 | 关键影响 |
|----------|------|----------|
| **CRITICAL** | 2 | 会导致前端功能直接崩溃 |
| **HIGH** | 5 | 可能导致重要功能异常或安全隐患 |
| **MEDIUM** | 19 | 字段约束缺失、错误码遗漏 |
| **LOW** | 4 | 命名不一致、文档过时 |

**消除误报**: 7 个初始发现经对抗性验证后被标记为 FALSE_POSITIVE，已从报告中移除。

**端点对齐**: 所有 46 个文档化端点与代码路由完全匹配，无缺失端点、无路径/方法不匹配。唯一未文档化端点是 `GET /api/health`（运维探活）。

---

## 1. CRITICAL 漂移 — 将导致前端功能崩溃

### C1. `SendMessageResult` 缺少 JSON Tags — 聊天响应字段名不匹配

- **端点**: `POST /api/visits/:sessionId/messages`
- **文件**: `internal/service/workbench/chat.go:27-31`
- **漂移类型**: NAME_MISMATCH

**问题**: `SendMessageResult` 结构体三个字段均无 `json` tag。Go 的 `encoding/json` 将字段序列化为大写开头 (`Session`, `PatientMessage`, `AssistantPlaceholder`)，而非文档规定的 camelCase (`session`, `patientMessage`, `assistantPlaceholder`)。

```go
// chat.go:27-31 — 缺少 json tags
type SendMessageResult struct {
    Session              *model.VisitSession
    PatientMessage       *model.TimelineItem
    AssistantPlaceholder *model.TimelineItem
}
```

**影响**: 前端 Zod schema 校验 `session`/`patientMessage`/`assistantPlaceholder` 将因字段名不匹配而失败，**每次消息发送的响应都会被拒绝，聊天功能完全不可用**。

### C2. `RecoveryMiddleware` 返回裸 ApiError — Panic 恢复破坏标准信封

- **端点**: 全局 (panic 恢复)
- **文件**: `internal/middleware/recovery.go:21-25`
- **漂移类型**: FORMAT_MISMATCH

**问题**: RecoveryMiddleware 在 panic 恢复后直接调用 `c.AbortWithStatusJSON(500, NewApiError(...))`，产生裸 JSON:
```json
{"code":"INTERNAL_ERROR","message":"internal server error","status":500}
```
而标准信封应为:
```json
{"success":false,"data":null,"error":{"code":"INTERNAL_ERROR","message":"internal server error","status":500}}
```

**影响**: 前端统一错误处理逻辑（期望 `response.error.code`）无法解析 panic 场景下的错误，可能导致未预期的客户端崩溃。

---

## 2. HIGH 漂移 — 重要功能异常或安全隐患

### H1. 患者/管理员 JWT 共享签名密钥

- **端点**: 全局 (认证系统)
- **文件**: `internal/handler/router.go:77,137`, `internal/config/config.go:17`
- **漂移类型**: SECURITY_DRIFT

**问题**: 文档明确要求「密钥、载荷、过期策略与患者端完全分离」。但代码中患者 `AuthMiddleware(cfg.JWTSecret)` 和管理员 `AdminAuthMiddleware(cfg.JWTSecret)` 使用**同一个** `JWTSecret`。`Config` 结构体没有 `AdminJWTSecret` 字段。

**影响**: HMAC 对称签名下，持有密钥的任一方可伪造另一方 Token。若任一系统被攻破，两个系统全部沦陷。

### H2. `GET /visits` 状态筛选参数未实现

- **端点**: `GET /api/visits`
- **文件**: `internal/handler/visit_handler.go:74-88`
- **漂移类型**: FIELD_IN_DOC_NOT_IN_CODE

**问题**: 文档注明 `status` 查询参数（可选，用于按会话状态筛选），但 Handler 仅解析 `patientId`、`cursor`、`pageSize`。Service 层和 Repository 层接口均无 `status` 参数。

**影响**: 前端依赖服务端按状态筛选时，将收到未筛选的全部会话，被迫在客户端自行过滤，分页逻辑变不正确。

### H3. `TITLE_ALREADY_EXISTS` 错误码定义了但从未返回

- **端点**: `POST /api/visits/:sessionId/generate-title`
- **文件**: `internal/service/workbench/title.go:35-37`
- **漂移类型**: IN_DOC_NOT_IN_CODE

**问题**: 文档规定当会话已有标题时应返回 `TITLE_ALREADY_EXISTS` (409)。但 `GenerateTitle` 服务在标题已存在时静默返回已有标题，不产生任何错误。

**影响**: 前端依赖此错误码的幂等性判断逻辑无法触发，静默覆盖可能导致预期外行为。

### H4. `LLM_UNAVAILABLE` 错误码定义了但从未返回

- **端点**: `POST /api/visits/:sessionId/generate-title`
- **文件**: `internal/service/workbench/title.go:44-51`
- **漂移类型**: IN_DOC_NOT_IN_CODE

**问题**: 文档规定 LLM 不可用时应返回 `LLM_UNAVAILABLE` (503)。但服务静默降级为 `chiefComplaint` 截断，不传播此错误。

**影响**: 前端无法区分「LLM 生成的标题」和「降级标题」，也无法提示用户后端服务异常。

### H5. `TIMEOUT` 错误码定义了但无超时中间件

- **端点**: 全局
- **文件**: `internal/errors/codes.go:13`
- **漂移类型**: IN_DOC_NOT_IN_CODE (死代码)

**问题**: 文档列出 `TIMEOUT` (408) 为预期错误码。代码定义了常量 `CodeTimeout`，但**没有任何超时中间件**，也没有任何代码路径产生此错误。

**影响**: 请求超时场景下，客户端不会收到 `TIMEOUT` 错误码，而是等待 TCP 连接超时或收到 `INTERNAL_ERROR`。

---

## 3. MEDIUM 漂移 — 字段约束缺失与错误码遗漏

### 字段约束缺失 (10 项)

所有以下请求结构体字段缺乏文档规定的长度/非空约束：

| # | 端点 | 字段 | 文档约束 | 代码状态 | 文件 |
|---|------|------|----------|----------|------|
| M1 | POST /visits/:sessionId/messages | content | string(1-2000) | 无约束 | workbench_requests.go:6 |
| M2 | POST /visits/:sessionId/messages | clientMessageId | string(min 1) | 无约束 | workbench_requests.go:7 |
| M3 | POST /visits/:sessionId/assistant-stream | requestId | string(min 1) | 无约束 | workbench_requests.go:13 |
| M4 | POST /visits | chiefComplaint | string(1-2000) | 无约束 | model/visit.go:69 |
| M5 | POST /visits/:sessionId/follow-up | chiefComplaint | string(1-2000) | 无约束 | model/visit.go:86 |
| M6 | POST /visits/:sessionId/lock-question | content | string(1-1000) | 无约束 | workbench_requests.go:54 |
| M7 | POST /visits/:sessionId/consult | content | string(1-1000) | 无约束 | workbench_requests.go:61 |
| M8 | POST /visits/:sessionId/classify-intent | content | string(1-1000) | 无约束 | workbench_requests.go:33 |
| M9 | POST /patients/verify | credential | string(min 4) | 无约束 | model/patient.go:61 |
| M10 | POST /visits/:sessionId/vitals | vitals | 结构化对象 | map[string]interface{} | workbench_requests.go:40 |

**影响**: 超长输入不会被边界拒绝，可能穿透到 Service 层甚至数据库，引发不可预期的行为或存储异常。`map[string]interface{}` 接受任意键值对，前端传入的非预期字段会被静默接受。

### SSE 流处理缺陷 (3 项)

| # | 问题 | 文件 | 影响 |
|---|------|------|------|
| M11 | **27 处 callback 错误返回值被忽略** — 所有 SSE 流回调的 error 返回值使用了 `_ = callback(...)`，客户端断开连接时服务继续处理，无法产生 mid-stream error 事件 | chat.go:201-521, consult.go | 流中断时前端收不到 error 事件，资源浪费 |
| M12 | **2/3 SSE Handler 忽略 NewSSEWriter 错误** — `AskLockedQuestion` 和 `StreamConsultationReply` 使用 `writer, _ := NewSSEWriter(c)` 丢弃错误，若 ResponseWriter 不支持 Flusher，nil writer 引发空指针 panic | workbench_handler.go:439,470 | 非 SSE 兼容客户端可触发服务器崩溃 |
| M13 | **Emergency 事件后缺少 done 事件** — 文档规定 emergency 后必须跟随 done 事件终结流，但 `handleEmergency` 发送 emergency 后直接返回，无 done | chat.go:371-398 | 前端可能认为流未正常结束，轮询/重连 |

### 错误码遗漏 (3 项)

| # | 错误码 | HTTP | 使用位置 | 文档状态 |
|---|--------|------|----------|----------|
| M14 | `FORBIDDEN` | 403 | workbench_handler.go:38, middleware.go:73, admin_auth.go:87 | 未在 errorCodes 列表中 |
| M15 | `NOT_FOUND` | 404 | router.go:151 (NoRoute handler) | 未在 errorCodes 列表中 |
| M16 | `INTERNAL_ERROR` | 500 | 30+ handler 位置 (全局 catch-all) | 未在 errorCodes 列表中 |

### 枚举漂移 (5 项)

| # | 枚举 | 漂移 | 详情 |
|---|------|------|------|
| M17 | VisitMachineState | code 多了 `transferred` | 文档 18 个值，代码 19 个值（enums.go:43） |
| M18 | MedicationFulfillmentStatus | doc 有 type，code 用 raw string | 无 Go type 定义 |
| M19 | FulfillmentStatus | doc 有 type，code 用 raw string | 无 Go type 定义 |
| M20 | TimerAction | doc 有 type，code 用 raw string | 无 Go type 定义 |
| M21 | AddressTag | doc 有 type，code 用 raw string | 无枚举常量和校验 |

### 文档遗漏 (1 项)

| # | 问题 | 详情 |
|---|------|------|
| M22 | rest-api.md 端点表缺失全部 7 个 auth 端点 | `/auth/register`, `/auth/login`, `/auth/refresh`, `/auth/logout`, `/admin/auth/login`, `/admin/auth/logout`, `/admin/auth/refresh` 在代码中实现但未出现在 rest-api.md 端点列表中（原 front-api.md 中有记录） |

---

## 4. LOW 漂移 — 命名不一致与文档过时

| # | 问题 | 影响 |
|---|------|------|
| L1 | `IntentType` (doc) vs `ConsultationIntent` (code) — 相同值，不同名 | 交叉引用混淆 |
| L2 | `TreatmentPlanType` (doc) vs `TreatmentPlan` (code) | 交叉引用混淆 |
| L3 | `Capability` (doc) vs `TreatmentCapability` (code) | 交叉引用混淆 |
| L4 | `TreatmentExecutionStatus` (doc) vs `ExecutionStatus` (code) | 交叉引用混淆 |

---

## 5. 误报说明 (7 项已验证为 FALSE POSITIVE)

以下初始发现经对抗性验证后被排除：

| 发现 | 排除原因 |
|------|----------|
| Heartbeat 未调用 | 文档从未提及 heartbeat/keepalive，实现是未使用的工具方法，不构成漂移 |
| admin refresh 错误粒度不匹配 | 文档对 admin 仅指定 INVALID_REFRESH_TOKEN，代码与之完全匹配 |
| /patients/verify auth 要求模糊 | 两份文档均明确 verify 在认证之前，代码实现正确 |
| Token 重用物理删除 | 文档明确要求「撤销全部 refreshToken」，代码行为与文档一致 |
| CreateAddressInput.isDefault | 非指针 bool 正确实现「可选，默认 false」的契约，无反序列化行为漂移 |
| UNKNOWN_ERROR 未产生 | 文档注明为前端 transport 概念，后端等效使用 INTERNAL_ERROR |
| NETWORK_ERROR 未产生 | 文档注明为前端传输层概念，后端不可能产生网络层错误 |

---

## 6. 漂移按 API 域分布

| API 域 | Critical | High | Medium | Low | 合计 |
|--------|----------|------|--------|-----|------|
| 通用/全局 | C2 | H1, H5 | M14-M16 | - | 6 |
| 认证 (/auth, /admin/auth) | - | - | M22 | - | 1 |
| 患者 (/patients) | - | - | M9-M10 | - | 2 |
| 问诊 (/visits) | C1 | H2 | M1-M8, M11-M13 | - | 14 |
| 工作台 (workbench) | C1 | H3, H4 | M1-M7, M11-M13 | - | 12 |
| 枚举/模型 | - | - | M17-M21 | L1-L4 | 9 |

---

## 7. 风险评估

| 风险 | 影响用户流程 | 严重度 |
|------|-------------|--------|
| C1 聊天响应字段名不匹配 | **发送消息 → 前端解析失败 → 聊天完全不可用** | 致命 |
| C2 非标准错误信封 | Panic 场景下前端无法正确展示错误 | 高 |
| H1 共享 JWT 密钥 | 安全边界模糊，一方沦陷双方遭殃 | 高 |
| M11-M13 SSE 缺陷 | 流式 AI 回复可靠性下降 | 中 |
| M1-M10 字段约束缺失 | 异常输入穿透到后端深层 | 中 |
| H2 状态筛选未实现 | 列表页 UI 状态过滤功能异常 | 中 |

---

## 8. 修复优先级

### P0 — 立即修复（预计 2-3 小时）

1. **C1**: 给 `SendMessageResult` 三个字段添加 `json:"session"` / `json:"patientMessage"` / `json:"assistantPlaceholder"` tag
2. **C2**: RecoveryMiddleware 改用 `WriteError` 或预序列化的标准信封 JSON

### P1 — 本周内修复（预计 5-8 小时）

3. **H1**: Config 新增 `AdminJWTSecret` 字段，admin 中间件使用独立密钥
4. **H3**: GenerateTitle 在标题已存在时返回 `TITLE_ALREADY_EXISTS` 或修改文档
5. **H4**: GenerateTitle 在 LLM 不可用时返回 `LLM_UNAVAILABLE` 或修改文档
6. **H5**: 添加超时中间件，或从错误码常量和文档中移除 TIMEOUT

### P2 — 下个迭代修复（预计 6-10 小时）

7. **M1-M9**: 使用 Gin binding tags (`binding:"required,min=1,max=2000"`) 添加字段约束
8. **M10**: VitalsRequest 改用结构化类型替代 `map[string]interface{}`
9. **M11**: 所有 callback 调用处理 error 返回值，断开时中止处理
10. **M12**: 修复 `AskLockedQuestion` / `StreamConsultationReply` 的 NewSSEWriter 错误处理
11. **M13**: `handleEmergency` 发送 emergency 后追加 done 事件

### P3 — 文档/清理（预计 3-5 小时）

12. **H2**: 实现 GET /visits 的 status 筛选或从文档移除
13. **M14-M16**: 文档中补充 FORBIDDEN/NOT_FOUND/INTERNAL_ERROR 错误码
14. **M17**: 文档补充 VisitMachineState `transferred` 或代码移除
15. **M18-M21**: 定义缺失的枚举 Go type，添加 Tag 校验
16. **M22**: rest-api.md 端点表补充 7 个 auth 端点
17. **L1-L4**: 统一枚举类型命名（doc ↔ code）

---

## 9. 预防建议

1. **契约驱动开发**: 将 `rest-api.md` 作为 CI 检查的一环。从 Go struct tags 自动生成 OpenAPI/Swagger spec，与前端 Zod schema 自动生成的 OpenAPI spec 进行 diff 比对
2. **JSON Tag 检查**: 添加 golangci-lint 规则强制所有 API response struct 有显式 `json` tag（使用 `musttag` linter）
3. **响应快照测试**: 对每个端点添加 golden-file 测试，确保 JSON 输出结构与文档完全一致
4. **Pre-commit 门控**: 扩展 pre-commit hook，当 handler/model 文件变更时自动对比文档字段表
5. **文档补丁合并**: patch v2-v9 的变更应合入 `rest-api.md` 主文档，避免多文档交叉引用产生的版本漂移

---

*报告由 Claude Code Workflow 自动生成 (workflow wf_ba7acfc2-7aa)*  
*审计覆盖: 25 agents, 1,657,784 tokens, 507 tool uses, 34 minutes*
