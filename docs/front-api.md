# NEUHIS Agent 结项 REST / SSE API 文档

更新时间：2026-06-29

> 本文档由本前端工程**已实现并通过 mock/契约测试的 Zod schema 与 facade** 自动梳理而成，是结项交付的**权威 REST/SSE 合约**。所有 endpoint、请求/响应字段、枚举取值、SSE 事件均取自源码 schema，未作发明。后端实现、联调与验收以本文档为基线；若需调整字段或枚举，必须同步回改 `src/**` schema、mock、契约测试与本文档。
>
> 主要来源：`src/lib/api/{types,errors,config,transport}.ts`、`src/lib/ui-message.ts`、`src/features/patient/api/{index,schemas}.ts`、`src/features/visits/api/{index,schemas}.ts`、`src/features/workbench/api/{index,schemas,timeline-schemas}.ts`，对齐 `special-designs/api.md` 的 medAgent 映射与边界。

---

## 1. 概述

NEUHIS Agent（产品名「东软云脑智能医疗」）是面向患者的「AI+诊疗」Agentic 聊天前端。本 API 支撑：身份核验、问诊会话生命周期、聊天与流式 AI 回复、流程卡片（检验/缴费/诊断/处置/用药/治疗执行/仅医嘱/完成）交互、急症守护、整次导诊计时与主动退出结算。

### 1.1 鉴权与患者身份上下文

- 系统采用 **JWT 双令牌认证**：**accessToken**（15 分钟有效期）携带于 `Authorization: Bearer <token>` header；**refreshToken**（7 天有效期）为不透明字符串，服务端存储，单次使用后轮换（rotation）。
- 用户通过 `POST /auth/register` 或 `POST /auth/login` 获取令牌对；accessToken 过期后通过 `POST /auth/refresh` 静默换取新令牌对。
- 患者身份通过 `POST /patients/verify` 核验后建立。核验返回患者摘要 `patient` 与可读范围 `readableScopes`（`profile` / `history` / `allergies` / `medications`）。
- 后续请求以 `patientId` / `sessionId` 作为路径或入参标识资源，并在 `Authorization` header 中携带有效 accessToken。
- 患者只能访问归属于自身的会话；越权访问应返回 HTTP `403`（见 §2.2 错误码）。

### 1.2 环境与运行模式

来源 `src/lib/api/config.ts`：

| 配置项 | 环境变量 | 默认值 | 说明 |
| --- | --- | --- | --- |
| 模式 | `VITE_API_MODE` | 开发 `mock` / 生产 `http` | `mock` 走内存 mock transport；`http` 走真实后端 |
| 基础前缀 | `VITE_API_BASE_URL` | `/api` | 所有 endpoint 在此前缀之下，如 `/api/visits` |
| mock 延迟 | `VITE_MOCK_DELAY_MS` | `400`（ms） | 仅 mock 模式生效，用于暴露 loading 态 |

- 所有 endpoint 路径**相对 `baseUrl`**。本文档中 `POST /visits` 实际请求为 `POST {baseUrl}/visits`，默认 `POST /api/visits`。
- 传输层接口（`src/lib/api/transport.ts`）：`get/post/patch/delete/stream`。普通请求返回 JSON 并经对应 Zod schema 校验；`stream` 走 SSE，事件经 `assistantStreamEventSchema` 校验后回调。
- `RequestOptions` 支持 `searchParams`、`headers`、`signal`（AbortSignal）。流式 `StreamHandlers` 提供 `onOpen` / `onEvent` / `onError` / `onDone` 与 `signal`。

---

## 2. 通用约定

### 2.1 ApiError 模型

来源 `src/lib/api/types.ts` (`apiErrorSchema`) 与 `src/lib/api/errors.ts`：

```jsonc
{
  "code": "SESSION_NOT_FOUND", // string，必填，错误码
  "message": "找不到这次就诊记录",  // string，必填，开发者可读说明
  "status": 404,                  // number，可选，HTTP 状态码
  "details": { },                 // unknown，可选，附加信息（如 Zod issues）
  "retriable": false              // boolean，可选，是否值得重试
}
```

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `code` | string(min 1) | 是 | 错误码，见 §2.2 |
| `message` | string(min 1) | 是 | 开发者可读说明（**不直接展示给患者**） |
| `status` | int>0 | 否 | HTTP 状态码 |
| `details` | unknown | 否 | 附加细节，Zod 校验失败时为 `error.issues` |
| `retriable` | boolean | 否 | 是否可重试；缺省时由 UI 兜底规则推断 |

> UI 不直接展示 `message`/`code`/内部状态名。`src/lib/ui-message.ts` 的 `toUiMessage()` 负责把 `ApiError` 转换成患者语言的 `UiMessage { title, description?, retriable }`。

### 2.2 错误码清单

来源 `src/lib/api/errors.ts` 与 `src/lib/ui-message.ts`：

| code | 来源 | retriable | 含义 |
| --- | --- | --- | --- |
| `VALIDATION_ERROR` | Zod 响应解析失败 (`createValidationApiError`) | false | 返回数据不符合前端契约，`details` 为 Zod issues |
| `UNKNOWN_ERROR` | 未知异常兜底 (`toApiError`) | true | 未识别异常，默认可重试 |
| `NETWORK_ERROR` | 网络层 | true | 网络连接不稳定 |
| `UNAUTHORIZED` | 认证失败 | false | 未提供有效认证凭据 |
| `FORBIDDEN` | 权限不足 | false | 无权访问指定资源 |
| `NOT_FOUND` | 资源未找到 | false | 端点或资源不存在 |
| `TIMEOUT` | 请求超时 | false | 请求处理超时 |
| `INTERNAL_ERROR` | 服务器内部错误 | true | 服务器异常 |
| `SESSION_NOT_FOUND` | 业务 | false | 找不到该就诊会话 |
| `PATIENT_NOT_FOUND` | 业务 | false | 找不到患者信息 |
| `CARD_NOT_FOUND` | 业务 | true | 流程卡已更新/失效，提示刷新 |
| `AUTH_PHONE_EXISTS` | 业务 | false | 注册时手机号已存在 |
| `AUTH_INVALID_CREDENTIALS` | 业务 | false | 登录时手机号或密码不匹配 |
| `AUTH_TOKEN_EXPIRED` | 业务 | false | accessToken 过期（JWT exp 校验失败） |
| `AUTH_REFRESH_INVALID` | 业务 | false | refreshToken 无效、已被使用或已被撤销 |
| `AUTH_REFRESH_EXPIRED` | 业务 | false | refreshToken 超过 7 天有效期 |
| `RATE_LIMITED` | 业务 | false | 超出速率限制 |
| `TITLE_ALREADY_EXISTS` | 业务 | false | 会话已有标题（幂等保护） |
| `LLM_UNAVAILABLE` | 业务 | true | 大模型服务不可用 |
| `ADDRESS_NOT_FOUND` | 业务 | false | 收货地址不存在 |
| `ADDRESS_LIMIT_EXCEEDED` | 业务 | false | 地址数量已达上限（最多 10 条） |
| `ADDRESS_REQUIRED` | 业务 | false | 当前操作需先添加收货地址 |

UI 文案命中的 HTTP 状态（`MESSAGE_BY_HTTP_STATUS`）：`401`（登录失效，不可重试）、`403`（无法访问该记录，不可重试）、`404`（找不到内容，不可重试）、`408`（请求超时，可重试）。其余 HTTP 错误按 `status >= 500` 可重试、`4xx` 不可重试兜底。

### 2.3 分页（cursor）

来源 `pageResultSchema`（`src/lib/api/types.ts`）。游标式分页，响应统一形如：

```jsonc
{
  "items": [ /* TItem[] */ ],
  "nextCursor": "opaque-cursor", // string，可选；无更多数据时缺省
  "hasMore": true                 // boolean，必填
}
```

- 请求侧传 `cursor`（可选）与 `pageSize`；用 `nextCursor` 拉取下一页（时间线场景为更早数据）。
- `TimelineItem.id` 在 mock 与 HTTP 下都必须稳定，避免虚拟列表重挂载。

### 2.4 时间戳格式

所有时间字段均为 **ISO 8601 字符串**（Zod `z.string().datetime()`），如 `2026-06-29T08:30:00.000Z`。

### 2.5 ID 类型

来源 `src/lib/api/types.ts`，均为 `z.string().trim().min(1)`（非空字符串）：

| 类型 | schema | 用途 |
| --- | --- | --- |
| `PatientId` | `patientIdSchema` | 患者标识 |
| `SessionId` | `sessionIdSchema` | 会话标识 |
| `TimelineItemId` | `timelineItemIdSchema` | 时间线条目标识 |
| `FlowCardId` | `flowCardIdSchema` | 流程卡标识 |

---

## 3. 状态枚举附录

以下取值逐字取自 schema，顺序与源码一致。

### 3.1 `VisitStatus`（`visitStatusSchema`）

会话对外状态（10 值）：`loading_context`、`chatting`、`analyzing`、`blocked`、`diagnosis`、`treatment`、`completed`、`transferred`、`emergency_terminated`、`exited`。

### 3.2 `VisitMachineState`（`visitMachineStateSchema`）

前端状态机内部态（17 值）：`loadingContext`、`chatting`、`analyzing`、`labDecision`、`labPayment`、`labExecution`、`diagnosis`、`treatmentDecision`、`medicationPayment`、`medicationFulfillment`、`treatmentExecution`、`adviceOnly`、`completed`、`emergencyPending`、`terminated`、`exitSettlement`、`exited`。

### 3.3 `TerminalReason`（`terminalReasonSchema`）

终止原因（7 值）：`emergency`、`timeout`、`ask_limit_reached`、`lab_limit_reached`、`referral`、`capability_insufficient`、`exited`。

### 3.4 `PaymentStatus`（`paymentStatusSchema`）

支付状态（5 值）：`unpaid`、`pending`、`paid`、`failed`、`refunded`。

### 3.5 `VisitEntryType`（`visitEntryTypeSchema`）

会话入口类型：`new`、`follow_up`。

### 3.6 `FlowCardKind`（`flowCardKindSchema`）

流程卡类型（9 值）：`lab_decision`、`payment`、`lab_execution`、`diagnosis`、`treatment_plan`、`medication_fulfillment`、`treatment_execution`、`advice_only`、`completed_visit`。

### 3.7 `FlowCardStatus`（`flowCardStatusSchema`）

流程卡状态（9 值）：`pending`、`accepted`、`skipped`、`vetoed`、`paid`、`processing`、`completed`、`failed`、`invalidated`。

### 3.8 `TimelineItem.kind`（`timelineItemSchema` 判别字段）

时间线条目类型（4 值）：`message`、`flow_card`、`system_event`、`terminal`。

### 3.9 `TimelineItemStatus`（`timelineItemStatusSchema`）

时间线条目状态（5 值）：`pending`、`streaming`、`done`、`failed`、`invalidated`。

### 3.10 `system_event` 的 `eventType`（`systemEventTimelineItemSchema`）

系统事件类型（8 值）：`context_loaded`、`agent_thinking`、`lab_result_received`、`payment_succeeded`、`drug_purchased`、`follow_up_started`、`emergency_dismissed`、`exit_settled`。

### 3.11 `AssistantStreamEvent.type`（`assistantStreamEventSchema` 判别字段）

SSE 事件类型（7 值）：`delta`、`message_final`、`card`、`state`、`emergency`、`done`、`error`。详见 §6。

---

## 4. Endpoint 清单

> 「写入时间线?」指该操作是否产出/追加 `TimelineItem`（直接返回或经 SSE 下发）。

| Method | Path | Facade 方法 | 用途 | 写入时间线? |
| --- | --- | --- | --- | --- |
| `POST` | `/auth/register` | `authApi.register` | 注册新用户，签发令牌对 | 否 |
| `POST` | `/auth/login` | `authApi.login` | 手机号+密码登录，签发令牌对 | 否 |
| `POST` | `/auth/refresh` | `authApi.refresh` | 使用 refreshToken 换取新令牌对 | 否 |
| `POST` | `/auth/logout` | `authApi.logout` | 注销当前会话，使 refreshToken 失效 | 否 |
| `POST` | `/patients/verify` | `patientApi.verifyIdentity` | 身份核验，返回患者摘要与可读范围 | 否 |
| `GET` | `/patients/:patientId/context` | `patientApi.getPatientContext` | 读取问诊上下文（病史/过敏史/上次诊断） | 否 |
| `PATCH` | `/patients/:patientId/profile` | `patientApi.updatePatientProfile` | 更新患者过敏史/慢病/长期用药 | 否 |
| `POST` | `/visits` | `visitsApi.createSession` | 创建新出诊会话 | 是（initialTimeline） |
| `POST` | `/visits/:sessionId/follow-up` | `visitsApi.createFollowUp` | 由父会话创建复诊会话 | 是（initialTimeline） |
| `GET` | `/visits` | `visitsApi.listSessions` | 分页查询历史就诊列表 | 否 |
| `GET` | `/visits/:sessionId` | `visitsApi.getSession` / `workbenchApi.getSession` | 查询会话详情与当前状态 | 否 |
| `GET` | `/visits/:sessionId/snapshot` | `visitsApi.getReadonlySnapshot` | 只读回看完整快照 | 否（返回完整时间线） |
| `GET` | `/visits/:sessionId/timeline` | `workbenchApi.listTimeline` | 分页查询消息+卡片混排时间线 | 否（读取） |
| `POST` | `/visits/:sessionId/messages` | `workbenchApi.sendMessage` | 发送患者消息，返回占位与会话状态 | 是 |
| `POST` | `/visits/:sessionId/assistant-stream` | `workbenchApi.streamAssistantMessage` | **SSE** 流式生成 AI 回复与卡片 | 是（流式） |
| `POST` | `/visits/:sessionId/lab-decision` | `workbenchApi.submitLabDecision` | 提交检验决定（同意/不查/暂不决定） | 是 |
| `POST` | `/visits/:sessionId/payments` | `workbenchApi.submitPayment` | 创建/确认检验或药品支付 | 是 |
| `POST` | `/visits/:sessionId/fulfillment` | `workbenchApi.submitFulfillment` | 提交取药/配送方式确认 | 是 |
| `POST` | `/visits/:sessionId/treatment-execution` | `workbenchApi.submitTreatmentExecution` | 自动化治疗预约/到号/开始/完成/取消 | 是 |
| `POST` | `/visits/:sessionId/advice-ack` | `workbenchApi.ackAdvice` | 确认仅医嘱处置并留痕 | 是 |
| `POST` | `/visits/:sessionId/lock-question` | `workbenchApi.askLockedQuestion` | **SSE** 锁定态卡片旁路疑问 | 是（流式） |
| `POST` | `/visits/:sessionId/classify-intent` | `workbenchApi.classifyFollowUpIntent` | 完成态输入意图分类 | 否 |
| `POST` | `/visits/:sessionId/consult` | `workbenchApi.streamConsultationReply` | **SSE** 完成态咨询问答（不触发复诊） | 是（流式） |
| `POST` | `/visits/:sessionId/vitals` | `workbenchApi.reportVitals` | 上报体征触发急症复检 | 否（命中后经 emergency） |
| `POST` | `/visits/:sessionId/exit` | `workbenchApi.exitVisit` | 主动退出并生成结算结果 | 是 |
| `POST` | `/visits/:sessionId/timer` | `workbenchApi.pauseVisitTimer` / `resumeVisitTimer` | 暂停/恢复整次导诊总计时 | 否 |
| `POST` | `/visits/:sessionId/generate-title` | `workbenchApi.generateTitle` | 调用 LLM 生成会话标题 | 否 |
| `POST` | `/visits/:sessionId/dismiss-emergency` | `workbenchApi.dismissEmergency` | 误报申诉，解除急症态 | 是 |
| `GET` | `/patients/:patientId/addresses` | `addressApi.listAddresses` | 查询患者收货地址列表 | 否 |
| `POST` | `/patients/:patientId/addresses` | `addressApi.createAddress` | 新增收货地址 | 否 |
| `PATCH` | `/patients/:patientId/addresses/:addressId` | `addressApi.updateAddress` | 修改收货地址 | 否 |
| `DELETE` | `/patients/:patientId/addresses/:addressId` | `addressApi.deleteAddress` | 删除收货地址 | 否 |
| `PUT` | `/patients/:patientId/addresses/:addressId/default` | `addressApi.setDefaultAddress` | 设置默认收货地址 | 否 |
| `GET` | `/billing/records` | `billingApi.listBillingRecords` | 查询患者历史账单汇总 | 否 |

> 检验结果回填 `POST /visits/:sessionId/lab-results`（`special-designs/api.md` 列出）由 mock/后端内部驱动，前端 facade 不直接暴露方法，故不在本表 facade 列展开。

---

## 5. Endpoint 详解

### 5.1 patient 域

#### `POST /patients/verify` — 身份核验

请求体（`verifyIdentityInputSchema`）：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `credentialType` | enum `id_card` \| `phone` | 是 | 凭证类型 |
| `credential` | string(min 4) | 是 | 凭证号 |
| `name` | string(min 1) | 否 | 姓名 |

响应（`verifyIdentityResultSchema`）：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `patient` | `PatientProfile` | 是 | 患者摘要，见下 |
| `readableScopes` | enum 数组 `profile`\|`history`\|`allergies`\|`medications` | 是 | 可读范围 |
| `verifiedAt` | ISO8601 | 是 | 核验时间 |

`PatientProfile`（`patientProfileSchema`）：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `id` | PatientId | 是 | 患者 ID |
| `name` | string(min 1) | 是 | 姓名 |
| `gender` | enum `male`\|`female`\|`other`\|`unknown` | 是 | 性别 |
| `age` | int 0–130 | 是 | 年龄 |
| `phoneMasked` | string | 否 | 脱敏手机号 |
| `idCardMasked` | string | 否 | 脱敏身份证 |
| `allergies` | string[] | 是 | 过敏史 |
| `chronicDiseases` | string[] | 是 | 慢性病 |
| `longTermMedications` | string[] | 是 | 长期用药 |
| `updatedAt` | ISO8601 | 是 | 更新时间 |

注意错误：`PATIENT_NOT_FOUND`、`VALIDATION_ERROR`、`UNAUTHORIZED`。

#### `GET /patients/:patientId/context` — 问诊上下文

路径参数：`patientId`（PatientId）。

响应（`patientContextSchema`）：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `patient` | `PatientProfile` | 是 | 患者摘要 |
| `chiefComplaint` | string | 否 | 主诉 |
| `medicalHistory` | string[] | 是 | 病史 |
| `allergies` | string[] | 是 | 过敏史 |
| `longTermMedications` | string[] | 是 | 长期用药 |
| `priorVisit` | `PatientPriorVisit` | 否 | 上次就诊纪要 |

`PatientPriorVisit`（`patientPriorVisitSchema`）：`sessionId`(必)、`completedAt`(ISO8601, 必)、`diagnosis`(必)、`labResultSummary`(可选)、`treatmentSummary`(必)。

注意错误：`PATIENT_NOT_FOUND`、`FORBIDDEN`、`NOT_FOUND`。

#### `PATCH /patients/:patientId/profile` — 更新患者资料

路径参数：`patientId`。请求体（`updatePatientProfileInputSchema`）：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `patientId` | PatientId | 是 | 患者 ID（与路径一致） |
| `allergies` | string[] | 否 | 过敏史（整体替换） |
| `chronicDiseases` | string[] | 否 | 慢性病 |
| `longTermMedications` | string[] | 否 | 长期用药 |
| `medicalHistory` | string[] | 否 | 既往病史（整体替换）；不传则不修改，传 `[]` 则清空 |

响应：更新后的 `PatientProfile`。注意错误：`PATIENT_NOT_FOUND`、`VALIDATION_ERROR`。

#### 地址簿（v5）

地址簿接口管理患者的收货地址，上限 10 条，首条自动设为默认。

##### `GET /patients/:patientId/addresses` — 地址列表

路径参数：`patientId`。无需请求体。

响应：`AddressListResponse`：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `addresses` | `Address[]` | 是 | 地址列表，按创建时间倒序 |

其中 `Address` 字段：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `id` | AddressId | 是 | 地址 ID（UUID） |
| `patientId` | PatientId | 是 | 所属患者 ID |
| `name` | string(1–20) | 是 | 收件人姓名 |
| `phone` | string(11) | 是 | 大陆手机号（1 开头 11 位） |
| `province` | string | 是 | 省 |
| `city` | string | 是 | 市 |
| `district` | string | 是 | 区 |
| `detail` | string(1–200) | 是 | 详细地址 |
| `isDefault` | boolean | 是 | 是否默认地址 |
| `tag` | string | 是 | 标签：`家` \| `公司` \| `医院` \| `其他` \| `""` |
| `createdAt` | ISO8601 | 是 | 创建时间 |
| `updatedAt` | ISO8601 | 是 | 更新时间 |

##### `POST /patients/:patientId/addresses` — 新增地址

路径参数：`patientId`。请求体（`CreateAddressInput`）：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `patientId` | PatientId | 是 | 患者 ID（与路径一致） |
| `name` | string(1–20) | 是 | 收件人姓名 |
| `phone` | string(11) | 是 | 大陆手机号 |
| `province` | string | 是 | 省 |
| `city` | string | 是 | 市 |
| `district` | string | 是 | 区 |
| `detail` | string(1–200) | 是 | 详细地址 |
| `isDefault` | boolean | 否 | 是否设为默认（首条强制 true） |
| `tag` | string | 否 | 标签，默认 `""` |

响应：`201 Created`，返回创建的 `Address`。错误：`ADDRESS_LIMIT_EXCEEDED`（已达 10 条上限）、`VALIDATION_ERROR`。

> 首条地址自动设为默认（`isDefault: true`）。若 `isDefault: true`，会清除其他地址的默认标记。

##### `PATCH /patients/:patientId/addresses/:addressId` — 修改地址

路径参数：`patientId`、`addressId`。请求体（`UpdateAddressInput`，全部可选，仅更新非 `nil` 字段）：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `name` | string(1–20) \| null | 否 | 收件人姓名 |
| `phone` | string(11) \| null | 否 | 大陆手机号 |
| `province` | string \| null | 否 | 省 |
| `city` | string \| null | 否 | 市 |
| `district` | string \| null | 否 | 区 |
| `detail` | string(1–200) \| null | 否 | 详细地址 |
| `isDefault` | boolean \| null | 否 | 是否设为默认 |
| `tag` | string \| null | 否 | 标签 |

响应：更新后的 `Address`。错误：`ADDRESS_NOT_FOUND`（地址不存在或不属于该患者）、`VALIDATION_ERROR`。

##### `DELETE /patients/:patientId/addresses/:addressId` — 删除地址

路径参数：`patientId`、`addressId`。无需请求体。

响应：`DeleteAddressResponse`：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `success` | boolean | 是 | 固定 `true` |

> 若删除的是默认地址，则自动将剩余第一条地址提升为默认。

错误：`ADDRESS_NOT_FOUND`。

##### `PUT /patients/:patientId/addresses/:addressId/default` — 设置默认

路径参数：`patientId`、`addressId`。无需请求体。

响应：更新后的 `Address`（`isDefault: true`）。错误：`ADDRESS_NOT_FOUND`。

### 5.2 visits 域

#### `POST /visits` — 创建新出诊

请求体（`createSessionInputSchema`，`.strict()` 禁止额外字段）：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `patientId` | PatientId | 是 | 患者 ID |
| `entryType` | literal `"new"` | 是 | 固定为新出诊 |
| `chiefComplaint` | string(1–2000) | 否 | 主诉 |

响应（`createSessionResultSchema`）：`session`（`VisitSession`）+ `initialTimeline`（`TimelineItem[]`）。

#### `POST /visits/:sessionId/follow-up` — 创建复诊

路径参数：`sessionId`（父会话 ID，等于 body 的 `parentSessionId`）。请求体（`createFollowUpInputSchema`）：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `patientId` | PatientId | 是 | 患者 ID |
| `parentSessionId` | SessionId | 是 | 父会话 ID |
| `chiefComplaint` | string(1–2000) | 否 | 主诉 |

响应：同 `createSessionResultSchema`。

#### `GET /visits` — 历史就诊列表

查询参数（`listSessionsInputSchema`）：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `patientId` | PatientId | 否 | 按患者过滤 |
| `status` | `VisitStatus` | 否 | 按状态过滤 |
| `cursor` | string | 否 | 分页游标 |
| `pageSize` | int 1–50（默认 20） | 否 | 每页条数 |

响应（`listSessionsResultSchema` = `PageResult<VisitSessionSummary>`）：`items` 为 `VisitSessionSummary`，含 `id`、`patientId`、`entryType`、`status`、`startedAt`、`updatedAt`、`endedAt?`、`parentSessionId?`、`terminalReason?`、`summary`。

#### `GET /visits/:sessionId` — 会话详情

路径参数：`sessionId`。响应为完整 `VisitSession`（`visitSessionSchema`）：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `id` | SessionId | 是 | 会话 ID |
| `patientId` | PatientId | 是 | 患者 ID |
| `entryType` | `VisitEntryType` | 是 | `new` / `follow_up` |
| `status` | `VisitStatus` | 是 | 会话状态 |
| `startedAt` | ISO8601 | 是 | 开始时间 |
| `updatedAt` | ISO8601 | 是 | 更新时间 |
| `endedAt` | ISO8601 | 否 | 结束时间 |
| `timeoutAt` | ISO8601 | 否 | 总计时截止时间 |
| `pausedAt` | ISO8601 | 否 | 当前暂停起点 |
| `askRound` | int≥0 | 是 | 已用追问轮次 |
| `askRoundLimit` | int>0 | 是 | 追问轮次上限 |
| `labRound` | int≥0 | 是 | 已用检验轮次 |
| `labRoundLimit` | int>0 | 是 | 检验轮次上限 |
| `parentSessionId` | SessionId | 否 | 父会话（复诊时） |
| `terminalReason` | `TerminalReason` | 否 | 终止原因 |
| `activeCardId` | string | 否 | 当前阻塞/活动卡 ID |
| `timerPaused` | boolean | 是 | 计时是否暂停 |
| `summary` | `VisitSummary` | 是 | 摘要 |

校验约束（`superRefine`）：`entryType=new` 不得带 `parentSessionId`；`status=blocked` 必须带 `activeCardId`。

`VisitSummary`（`visitSummarySchema`，全部可选）：`title?`、`chiefComplaint?`、`diagnosis?`、`treatmentSummary?`、`lastMessage?`。

> `title`：AI 生成的问诊记录标题（由 `POST /visits/:sessionId/generate-title` 生成）。前端展示优先级：`title > chiefComplaint > "未命名问诊"`。

#### `GET /visits/:sessionId/snapshot` — 只读快照

路径参数：`sessionId`。响应（`visitSnapshotSchema`）：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `session` | `VisitSession` | 是 | 会话 |
| `timeline` | `TimelineItem[]` | 是 | 完整时间线 |
| `readonly` | literal `true` | 是 | 只读标记 |
| `terminalReason` | `TerminalReason` | 否 | 终止原因 |

注意错误：`SESSION_NOT_FOUND`。

### 5.3 workbench 域 — 聊天（chat）

#### `GET /visits/:sessionId/timeline` — 时间线分页

查询参数（`listTimelineInputSchema`）：`sessionId`(必)、`cursor`(可选)、`pageSize`(int 1–100，默认 50)。响应为 `PageResult<TimelineItem>`。`TimelineItem` 各形态见 §6.1。

#### `POST /visits/:sessionId/messages` — 发送患者消息

请求体（`sendMessageInputSchema`）：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `sessionId` | SessionId | 是 | 会话 ID |
| `content` | string(1–2000) | 是 | 消息内容 |
| `clientMessageId` | string(min 1) | 是 | 客户端幂等 ID |

响应（`sendMessageResultSchema`）：`session`（`VisitSession`）、`patientMessage`（`TimelineItem`）、`assistantPlaceholder?`（`TimelineItem`，可选占位）。

#### `POST /visits/:sessionId/assistant-stream` — 流式 AI 回复（SSE）

请求体（`streamAssistantInputSchema`）：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `sessionId` | SessionId | 是 | 会话 ID |
| `requestId` | string(min 1) | 是 | 本轮流式请求 ID |
| `clientMessageId` | string(min 1) | 否 | 关联的患者消息 ID |

响应：SSE 事件流，事件类型见 §6.2。每个事件经 `assistantStreamEventSchema` 校验。客户端应支持 `AbortSignal` 以便退出/急症时中断。

### 5.4 workbench 域 — 检验（lab）

#### `POST /visits/:sessionId/lab-decision` — 检验决定

请求体（`submitLabDecisionInputSchema`）：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `sessionId` | SessionId | 是 | 会话 ID |
| `cardId` | FlowCardId | 是 | 检验决定卡 ID |
| `decision` | enum `accepted`\|`skipped`\|`vetoed` | 是 | 同意检验 / 不查 / 暂不决定 |

响应：`FlowActionResult`（见 §5.10）。`accepted` 驱动产出检验缴费卡；`skipped` 走诊断（证据不含 lab_result）；`vetoed` 解除阻塞回到 `chatting`。注意错误：`CARD_NOT_FOUND`。

### 5.5 workbench 域 — 支付（payment）

#### `POST /visits/:sessionId/payments` — 创建/确认支付

请求体（`submitPaymentInputSchema`）：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `sessionId` | SessionId | 是 | 会话 ID |
| `cardId` | FlowCardId | 是 | 支付卡 ID |
| `purpose` | enum `lab`\|`medication` | 是 | 检验费 / 药品费 |
| `paymentMethodId` | string(min 1) | 否 | 支付方式 ID |
| `simulateStatus` | `PaymentStatus` | 否 | mock 模拟支付结果 |
| `defer` | boolean | 否 | 暂缓支付 |

响应：`FlowActionResult`。检验费支付成功后由后端回填检验结果驱动续跑；药品费支付成功后产出取药卡。注意错误：`CARD_NOT_FOUND`。

### 5.6 workbench 域 — 用药/治疗执行（treatment）

#### `POST /visits/:sessionId/fulfillment` — 取药/配送确认

请求体（`submitFulfillmentInputSchema`）：`sessionId`(必)、`cardId`(必)、`mode`(enum `pickup`\|`delivery`，必)。响应：`FlowActionResult`。

#### `POST /visits/:sessionId/treatment-execution` — 自动化治疗推进

请求体（`submitTreatmentExecutionInputSchema`）：`sessionId`(必)、`cardId`(必)、`action`(enum `schedule`\|`confirm_arrival`\|`start`\|`complete`\|`cancel`，必)。响应：`FlowActionResult`。

> 自动化治疗为前端/mock 语义，medAgent 无对应（见 §8）。

#### `POST /visits/:sessionId/advice-ack` — 仅医嘱确认

请求体（`ackAdviceInputSchema`）：`sessionId`(必)、`cardId`(必)。响应：`FlowActionResult`，确认后进入完成态。

### 5.7 workbench 域 — 咨询/锁定问答（consult）

#### `POST /visits/:sessionId/lock-question` — 锁定态卡片疑问（SSE）

请求体（`askLockedQuestionInputSchema`）：`sessionId`(必)、`cardId`(必)、`content`(string 1–1000, 必)、`requestId`(必)。响应：SSE 事件流（旁路问答，不推进主流程）。

#### `POST /visits/:sessionId/consult` — 完成态咨询（SSE）

请求体（`consultationInputSchema`）：`sessionId`(必)、`content`(string 1–1000, 必)、`requestId`(必)。响应：SSE 事件流，基于本次记录作答，不创建复诊会话。

### 5.8 workbench 域 — 意图分类（intent）

#### `POST /visits/:sessionId/classify-intent` — 完成态意图分类

请求体（`classifyIntentInputSchema`）：`sessionId`(必)、`content`(string 1–1000, 必)。

响应（`classifyIntentResultSchema`）：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `intent` | enum `consultation`\|`follow_up`\|`uncertain` | 是 | 意图分类 |
| `confidence` | number 0–1 | 是 | 置信度 |
| `reason` | string | 否 | 判定理由 |

### 5.9 workbench 域 — 会话控制（visit-control）

#### `POST /visits/:sessionId/vitals` — 上报体征（急症复检）

请求体（`reportVitalsInputSchema`）：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `sessionId` | SessionId | 是 | 会话 ID |
| `source` | enum `patient_report`\|`device`\|`manual` | 是 | 体征来源 |
| `symptoms` | string[] | 是 | 症状列表 |
| `vitals` | object | 否 | 体征值，见下 |

`vitals`（全部可选）：`temperature`(number)、`heartRate`(int>0)、`systolicPressure`(int>0)、`diastolicPressure`(int>0)、`spo2`(number 0–100)。

响应（`emergencyRecheckResultSchema`）：`emergency`(boolean, 必)、`severity?`(enum `suspected`\|`critical`)、`message?`(string)。

#### `POST /visits/:sessionId/exit` — 主动退出结算

请求体（`exitVisitInputSchema`）：`sessionId`(必)、`reason`(enum `patient_request`\|`timeout`\|`emergency`\|`other`，必)。

响应（`exitSettlementResultSchema`）：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `sessionId` | SessionId | 是 | 会话 ID |
| `terminalReason` | `TerminalReason` | 是 | 终止原因 |
| `refundAmount` | number≥0 | 是 | 退款金额 |
| `payableAmount` | number≥0 | 是 | 应付金额 |
| `timelineItem` | `TimelineItem` | 是 | 终止时间线项 |
| `consequence` | `ExitConsequence` | 否 | 结算后果 |

`ExitConsequence`（`exitConsequenceSchema`）：`kind`(enum `no_fee`\|`refundable`\|`executed_no_refund`\|`medication_dispensed`，必)、`amount?`(number≥0)、`text`(string min 1, 必)。四档语义见 §7.4。

#### `POST /visits/:sessionId/timer` — 暂停/恢复总计时

facade `pauseVisitTimer` / `resumeVisitTimer` 共用此 endpoint。请求体为 `{ sessionId }`，transport 自动注入 `action: "pause"` 或 `action: "resume"`。响应：更新后的 `VisitSession`（`timerPaused`、`pausedAt`、`timeoutAt` 相应变化）。

#### `POST /visits/:sessionId/dismiss-emergency` — 解除急症（误报申诉）

请求体（`dismissEmergencyInputSchema`）：`{ sessionId }`。响应（`dismissEmergencyResultSchema`）：`session`（`VisitSession`）+ `timelineItem`（`TimelineItem`，通常为 `system_event: emergency_dismissed`）。

> 急症恢复为前端/mock 语义（见 §8）。

### 5.10 公共响应：`FlowActionResult`

多数卡片动作（lab-decision / payments / fulfillment / treatment-execution / advice-ack）返回统一结构（`flowActionResultSchema`）：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `sessionId` | SessionId | 是 | 会话 ID |
| `status` | `VisitStatus` | 是 | 动作后会话状态 |
| `activeCardId` | FlowCardId | 否 | 新的活动卡 ID |
| `card` | `FlowCard` | 否 | 新增/更新的流程卡 |
| `timelineItems` | `TimelineItem[]` | 是 | 本次追加的时间线项 |
| `message` | string | 否 | 附加说明 |

### 5.11 auth 域 — JWT 认证

#### `POST /auth/register` — 注册

请求体：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `phone` | string | 是 | 手机号，11 位中国大陆号码 |
| `password` | string | 是 | 密码，最少 8 字符 |
| `realName` | string | 否 | 真实姓名；不传则留空 |

响应（201 Created）：

```jsonc
{
  "accessToken": "eyJhbGciOiJIUzI1NiIs...",
  "refreshToken": "dGhpcyBpcyBhIHJlZnJlc2g...",
  "expiresIn": 900,
  "user": {
    "userId": "u_abc123",
    "patientId": "p_xyz789",
    "phone": "13800138000",
    "realName": "张三"
  }
}
```

注意错误：`AUTH_PHONE_EXISTS`（409）、`VALIDATION_ERROR`（422）。

#### `POST /auth/login` — 登录

请求体：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `phone` | string | 是 | 手机号 |
| `password` | string | 是 | 密码 |

响应（200 OK）：同 register 响应结构。

注意错误：`AUTH_INVALID_CREDENTIALS`（401）、`RATE_LIMITED`（429）。

#### `POST /auth/refresh` — 刷新令牌

请求体：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `refreshToken` | string | 是 | 当前持有的 refreshToken |

响应（200 OK）：

```jsonc
{
  "accessToken": "eyJhbGciOiJIUzI1NiIs...",
  "refreshToken": "bmV3IHJlZnJlc2ggdG9rZW4...",
  "expiresIn": 900
}
```

注意错误：`AUTH_REFRESH_INVALID`（401）、`AUTH_REFRESH_EXPIRED`（401）。

#### `POST /auth/logout` — 注销

请求体：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `refreshToken` | string | 是 | 需要失效的 refreshToken |

响应：204 No Content（无响应体）。幂等处理——即使 token 已失效也返回 204。

#### Token 规格

**accessToken**：
- 格式：JWT（HS256）
- 传输：`Authorization: Bearer <accessToken>`
- 有效期：900 秒（15 分钟）
- Payload：`{ "sub": "userId", "patientId": "...", "phone": "...", "iat": ..., "exp": ... }`

**refreshToken**：
- 格式：不透明字符串（≥ 32 字节 Base64 URL-safe）
- 存储：服务端持久化，关联 userId
- 有效期：604800 秒（7 天）
- 单次使用后轮换（rotation）

#### 安全要求

| 要求 | 说明 |
| --- | --- |
| 密码存储 | bcrypt，cost factor ≥ 12 |
| refreshToken 单次使用 | 每次 refresh 调用后旧 token 立即失效 |
| Token theft 防护 | 已失效 refreshToken 被重用 → 撤销该用户全部 refreshToken |
| Rate limiting | `/auth/login` 和 `/auth/register` 限制 5 req/min/IP |
| JWT 签名密钥 | 至少 256-bit 随机密钥 |
| 传输安全 | 所有 auth 端点必须 HTTPS |

#### 数据库 schema

- `users` 表：`userId`、`phone`（唯一）、`passwordHash`、`realName`、`createdAt`
- `refresh_tokens` 表：`tokenHash`、`userId`、`expiresAt`、`usedAt`

### 5.12 workbench 域 — 会话标题生成

#### `POST /visits/:sessionId/generate-title` — 生成会话标题

调用后端 LLM 基于对话上下文生成简短问诊标题。

请求体：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `sessionId` | SessionId | 是 | 问诊会话 ID（需与路径参数一致） |

响应（200 OK）：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `sessionId` | SessionId | 回显的会话 ID |
| `title` | string | 生成的标题，1-50 字符 |

注意错误：`SESSION_NOT_FOUND`（404）、`VALIDATION_ERROR`（422）、`TITLE_ALREADY_EXISTS`（409，可选）、`LLM_UNAVAILABLE`（503）。

#### 标题生成规范

| 规则 | 说明 |
| --- | --- |
| 长度 | 1-50 字符 |
| 格式 | 简短中文短语，无标点结尾 |
| 内容 | 概括症状 + 时间线索，或诊断名称 |
| 示例 | "发热伴咳嗽3天"、"反复腹痛一周"、"上呼吸道感染" |

#### 后端实现要求

输入上下文：患者消息（`role: "patient"`）+ 助手前 2 条消息 + 已有诊断（若有则优先）。

LLM 降级策略：大模型不可用时返回 503 或降级使用 `chiefComplaint` 截断（≤50 字符）。

### 5.13 billing 域 — 账单记录（v6）

#### `GET /billing/records` — 历史账单汇总

查询当前患者的全部已支付账单，按时间倒序排列，适合对账与费用回溯。

请求：无路径参数和请求体，需携带有效的 `Authorization` header（JWT accessToken），患者身份从 token 的 `patientId` claim 中提取。

响应（`BillingRecordsResponse`）：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `items` | `BillingRecord[]` | 是 | 账单记录列表，无记录时为空数组 `[]` |

其中 `BillingRecord`：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `paymentId` | string | 是 | 支付 ID（UUID） |
| `sessionId` | SessionId | 是 | 所属就诊会话 ID |
| `sessionTitle` | string | 是 | 就诊标题（`chiefComplaint` > `diagnosis` > `title` 降级，兜底 "未知就诊"） |
| `purpose` | string | 是 | 费用用途说明（如 "lab"、"medication"） |
| `items` | `BillingLineItem[]` | 是 | 费用明细行 |
| `totalAmount` | number | 是 | 总金额（元） |
| `insuranceAmount` | number | 是 | 医保报销金额（元） |
| `selfPayAmount` | number | 是 | 自付金额（元） |
| `paymentStatus` | PaymentStatus | 是 | 支付状态（固定 `"paid"`，仅展示已支付记录） |
| `createdAt` | ISO8601 | 是 | 支付完成时间 |

`BillingLineItem`：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `name` | string | 是 | 项目名称（如 "血常规"、"阿莫西林"） |
| `amount` | number | 是 | 单价（元） |
| `quantity` | number \| null | 否 | 数量（仅 >1 时返回） |

#### 后端实现要求

**数据来源**：遍历患者所有 visit session 的 `FlowCard`，仅取 `kind=payment` 且 `paymentStatus=paid` 的卡片。

**会话标题降级**：`chiefComplaint` → `diagnosis` → `title` → `"未知就诊"`（当前端未调用 `generate-title` 时 `title` 为空）。

**时间戳**：优先使用 `handledAt`（支付处理时间），降级使用 `createdAt`（卡片创建时间）。

错误：`UNAUTHORIZED`（未认证，401）。

---

## 6. 时间线条目与 SSE 事件目录

### 6.1 `TimelineItem` 形态（`timelineItemSchema` 判别联合）

公共基字段（`timelineItemBaseSchema`）：`id`(TimelineItemId)、`sessionId`(SessionId)、`createdAt`(ISO8601)、`status`(`TimelineItemStatus`)。

| kind | 专属字段 |
| --- | --- |
| `message` | `role`(`patient`\|`assistant`)、`content`(string)、`localKey?`、`interruptedBy?`(`emergency`\|`timeout`\|`exit`) |
| `flow_card` | `card`（`FlowCard`，见 §6.3） |
| `system_event` | `eventType`（见 §3.10）、`title`、`description?` |
| `terminal` | `reason`(`TerminalReason`)、`title`、`description?`、`suggestedDepartment?` |

### 6.2 `AssistantStreamEvent` 目录（`assistantStreamEventSchema`）

SSE 流中每个事件的 payload：

| type | payload 字段 |
| --- | --- |
| `delta` | `sessionId`、`requestId`、`content`(string，文本增量) |
| `message_final` | `sessionId`、`requestId`、`item`（`message` 形态 `TimelineItem`） |
| `card` | `sessionId`、`requestId`、`card`（`FlowCard`）、`timelineItem?`（`flow_card` 形态 `TimelineItem`） |
| `state` | `sessionId`、`state`(`VisitMachineState`)、`status?`(`VisitStatus`)、`activeCardId?`(FlowCardId) |
| `emergency` | `sessionId`、`severity`(`suspected`\|`critical`)、`message`(string min 1) |
| `done` | `sessionId`、`requestId` |
| `error` | `sessionId?`、`requestId?`、`error`（`ApiError`） |

示例事件序列（一次追问）：

```jsonc
{ "type": "delta", "sessionId": "s1", "requestId": "r1", "content": "您好，" }
{ "type": "delta", "sessionId": "s1", "requestId": "r1", "content": "请问发热多久了？" }
{ "type": "message_final", "sessionId": "s1", "requestId": "r1",
  "item": { "id": "m9", "sessionId": "s1", "kind": "message", "role": "assistant",
            "status": "done", "createdAt": "2026-06-29T08:30:00.000Z",
            "content": "您好，请问发热多久了？" } }
{ "type": "done", "sessionId": "s1", "requestId": "r1" }
```

### 6.3 `FlowCard` 各类型字段（`flowCardSchema` 判别联合）

公共基字段（`flowCardBaseSchema`）：`id`(FlowCardId)、`sessionId`、`status`(`FlowCardStatus`)、`blocking`(boolean)、`title`(string)、`createdAt`(ISO8601)、`handledAt?`(ISO8601)、`lockReason?`(string)。各 kind 专属字段：

- `lab_decision`：`testItems[]`(`{ code, name, sampleType? }`)、`reason`、`differentialTargets[]`、`estimatedFee`(≥0)。
- `payment`：`paymentId`、`purpose`(`lab`\|`medication`)、`items[]`(`{ name, amount, quantity? }`)、`totalAmount`、`insuranceAmount`、`selfPayAmount`、`paymentStatus`(`PaymentStatus`)。
- `lab_execution`：`labOrderId`、`executionStatus`(`waiting_payment`\|`queued`\|`collecting`\|`testing`\|`result_ready`\|`completed`)、`resultSummary?`、`resultReturnedAt?`。
- `diagnosis`：`diagnosis`、`confidence`(`low`\|`medium`\|`high`)、`evidence[]`、`evidenceSources[]`(`history`\|`answer`\|`lab_result`)、`riskSignals[]`。
- `treatment_plan`：`plan`(`medication`\|`treatment`\|`advice_only`\|`referral`)、`capability`(`available`\|`limited`\|`unavailable`)、`summary`、`actions[]`。
- `medication_fulfillment`：`medications[]`(`{ name, spec, quantity, dosage, days, price }`)、`availableModes[]`(`pickup`\|`delivery`)、`selectedMode?`、`fulfillmentStatus`(`pending`\|`confirmed`\|`completed`)。
- `treatment_execution`：`treatmentName`、`capability`、`executionStatus`(`pending`\|`scheduled`\|`arrived`\|`in_progress`\|`completed`\|`canceled`)、`appointmentAt?`、`queueNo?`、`notices[]`、`availableActions[]`(`schedule`\|`confirm_arrival`\|`start`\|`complete`\|`cancel`)。
- `advice_only`：`advices[]`、`watchItems[]`、`followUpRecommendation`。
- `completed_visit`：`diagnosis`、`treatmentSummary`、`followUpSuggestion`、`completedAt`(ISO8601)。

### 6.4 medAgent `Step.kind` → SSE 映射

对齐 `special-designs/api.md`。HTTP 模式由后端 adapter 完成，mock 模式直接产出同样事件：

| medAgent `Step.kind` | SSE 产出 | 前端表现 |
| --- | --- | --- |
| `ASK` | `delta` × n + `message_final` | AI 追问气泡 |
| `NEED_TESTS`（恒血常规） | `card(lab_decision)` + `state` | 是否检验阻塞卡 |
| `DRUG_QUERY` | 仅 `state`（对用户透明） | 不渲染卡片，可选「正在核对药品规格」系统事件 |
| `PURCHASE` | `card(medication_fulfillment)` + `state` | 购药/取药确认卡（含盒数） |
| `EMERGENCY` | `emergency` | 急症 Overlay |
| `DONE` | `card(diagnosis)` + `card(completed_visit)` 或 `card(advice_only)` + `done` | 诊断卡 + 完成/医嘱卡 |

---

## 7. 典型时序

### 7.1 主流程（新建→发消息→流式→检验→缴费→结果回填→确诊→处置→完成）

1. `POST /patients/verify` → `POST /visits`（`entryType:new`）得到 `session` + `initialTimeline`。
2. `POST /visits/:id/messages`（患者主诉）→ 返回 `patientMessage` + `assistantPlaceholder?`。
3. `POST /visits/:id/assistant-stream`（SSE）→ `delta`×n + `message_final`（AI 追问），必要时 `card(lab_decision)` + `state`。
4. `POST /visits/:id/lab-decision`（`accepted`）→ `FlowActionResult` 含 `card(payment, purpose:lab)`。
5. `POST /visits/:id/payments`（`purpose:lab`）支付成功 → 后端回填检验结果（`lab_execution` 推进至 `result_ready`，`system_event: lab_result_received`）。
6. Agent 续跑 → `card(diagnosis)` + `card(treatment_plan)`。
7. 处置分流：
   - **用药**：`card(payment, purpose:medication)` → `POST /payments(medication)` → `card(medication_fulfillment)` → `POST /fulfillment(mode)` → 完成（`card(completed_visit)`）。
   - **仅医嘱**：`card(advice_only)` → `POST /advice-ack` → 完成。
   - **自动化治疗（mock）**：`card(treatment_execution)` → `POST /treatment-execution` 依次 `schedule`→`confirm_arrival`→`start`→`complete` → 完成。
8. 完成态产出 `card(completed_visit)`，`state(completed)` / `status=completed`。

### 7.2 急症

任意阶段流式命中 `emergency` 事件，或 `POST /vitals` 返回 `emergency:true`（`severity` `suspected`/`critical`）→ 前端弹 EmergencyOverlay。急症确认后会话终止（`status=emergency_terminated`，`terminalReason=emergency`）。误报可 `POST /dismiss-emergency` 解除（前端/mock 语义，见 §8）。

### 7.3 超时

前端总计时（`timeoutAt`）耗尽 → 前端发起 `POST /visits/:id/exit`（`reason:timeout`）收口，`terminalReason=timeout`。计时可经 `POST /timer`（pause/resume）暂停恢复。medAgent 无总超时（见 §8）。

### 7.4 主动退出结算（四档后果）

`POST /visits/:id/exit`（`reason:patient_request`）返回 `consequence.kind`：

| kind | 含义 | refund/payable 倾向 |
| --- | --- | --- |
| `no_fee` | 未产生任何费用，直接退出 | refund=0, payable=0 |
| `refundable` | 已付未执行，可退款 | refund>0 |
| `executed_no_refund` | 服务已执行，不可退款 | payable 已结算，refund=0 |
| `medication_dispensed` | 药品已发出，按已购计费 | payable 含药费，refund=0 |

### 7.5 完成后咨询 / 复诊

- 完成态输入先经 `POST /classify-intent` → `consultation` / `follow_up` / `uncertain`。
- `consultation` → `POST /consult`（SSE），基于本次记录作答，不创建会话。
- `follow_up` → `POST /visits/:id/follow-up` 创建复诊会话（携父会话纪要）。

---

## 8. 边界与未实现

对齐 `special-designs/api.md` 的 medAgent 边界，须在 contract 与 mock 中体现：

1. **无院内治疗执行（暂）**：medAgent 处置只有 `MEDICATION` / `ADVICE_ONLY` / `REFERRAL`，需院内操作/手术直接 `REFERRAL`。前端 contract 预留 `/treatment-execution` 与 `treatment_execution` 卡，由前端/mock 演示；仅接 medAgent 三分类的 HTTP 后端下，治疗类情形按 `REFERRAL` 映射为终止卡（`reason: referral` 或 `capability_insufficient`），该分支不出现。
2. **无总计时**：medAgent 无总超时强制转诊。前端总计时（`timeoutAt` / `/timer` / `timerPaused`）与超时退出是纯前端/mock 机制；HTTP 模式由前端发起 `exitVisit` 或后端转诊收口。
3. **急症会话即关闭**：medAgent 命中急症后会话关闭。前端「误报申诉恢复」（`emergencyPending` / `POST /dismiss-emergency`）只在前端/mock 成立；HTTP 模式若需可恢复语义，须后端 contract 显式支持。
4. **检验固定血常规**：当前 `NEED_TESTS` 恒为血常规。检验卡可据此简化，但 schema 仍保留 `testItems[]` 数组以备扩展。

补充：「暂不决定 / 不查」（`lab-decision` 的 `vetoed` / `skipped`）是前端 + 后端业务层语义，medAgent 不感知，调用 medAgent 前消化。自动化治疗、总计时、急症恢复均为**前端或 mock 语义**，后端业务层补能力后再接入。

---

## 附：来源核对

本文档的 endpoint、请求/响应字段、枚举与 SSE 事件均取自以下已实现源文件（结项时逐字核对）：

- `src/lib/api/types.ts`、`src/lib/api/errors.ts`、`src/lib/api/config.ts`、`src/lib/api/transport.ts`、`src/lib/ui-message.ts`
- `src/features/patient/api/{index,schemas}.ts`
- `src/features/visits/api/{index,schemas}.ts`
- `src/features/workbench/api/{index,schemas,timeline-schemas}.ts`
- 对齐文档：`agent-workspace/special-designs/api.md`
