# REST API Patch v9 — 空闲挂起、传输层补充与字段补遗

日期：2026-07-01

## 变更概述

本文档是对 `rest-api.md` 及已有 Patch（v2–v8）的增量补充，记录已在源码中实现但未在任何设计文档中体现的功能与约束：

1. **空闲挂起机制**：`VisitStatus.suspended` / `VisitMachineState.suspended` 值及 `POST /visits/:sessionId/suspend` 端点。
2. **`idle` 中断原因**：`MessageTimelineItem.interruptedBy` 新增 `"idle"` 值。
3. **`session_suspended` 系统事件**：`system_event` 的 `eventType` 新增 `"session_suspended"`。
4. **`lastActivityAt` 字段**：`VisitSession` 新增空闲计时基准字段。
5. **`ApiTransport.put()` 方法**：传输层接口补充 `put` 方法。
6. **地址标签约束**：`Address.tag` 的 `min(1)` / `max(20)` 约束与不可清空设计。
7. **配送地址电话宽松校验**：`deliveryAddress.phone` 与 `Address.phone` 的校验差异。
8. **SSE 流事件类型详述**：对 `assistantStreamEventSchema` 各事件的详细说明。

> 本 patch 不修改已有文档或端点；所有内容为纯新增记录。

---

## 1. 空闲挂起状态

### 1.1 `VisitStatus.suspended`

来源 `visitStatusSchema`（`src/lib/api/types.ts`）。

在 `completed` 与 `transferred` 之间新增 `"suspended"` 值。该状态**不是终态**——会话不写 `endedAt` 或 `terminalReason`，患者可按复诊流程以本会话为父会话继续问诊。

完整 `VisitStatus` 枚举（11 值）：

| 值 | 说明 |
| --- | --- |
| `loading_context` | 加载问诊上下文中 |
| `chatting` | 自由对话中 |
| `analyzing` | AI 正在分析 |
| `blocked` | 被流程卡阻塞，等待患者操作 |
| `diagnosis` | 诊断中 |
| `treatment` | 处置中 |
| `completed` | 完成 |
| **`suspended`** | **空闲挂起（非终态）** |
| `transferred` | 转诊 |
| `emergency_terminated` | 急症终止 |
| `exited` | 主动退出 |

### 1.2 `VisitMachineState.suspended`

来源 `visitMachineStateSchema`（`src/lib/api/types.ts`）。

在 `completed` 与 `emergencyPending` 之间新增 `"suspended"` 值。

完整 `VisitMachineState` 枚举（18 值）：

| 值 | 说明 |
| --- | --- |
| `loadingContext` | 加载上下文中 |
| `chatting` | 对话中 |
| `analyzing` | 分析中 |
| `labDecision` | 等待检验决策 |
| `labPayment` | 检验缴费 |
| `labExecution` | 检验执行中 |
| `diagnosis` | 诊断中 |
| `treatmentDecision` | 处置决策中 |
| `medicationPayment` | 药品缴费 |
| `medicationFulfillment` | 取药/配送确认 |
| `treatmentExecution` | 治疗执行中 |
| `adviceOnly` | 仅医嘱 |
| `completed` | 完成 |
| **`suspended`** | **空闲挂起** |
| `emergencyPending` | 急症待处理 |
| `terminated` | 已终止 |
| `exitSettlement` | 退出结算 |
| `exited` | 已退出 |

---

## 2. 会话挂起端点

### `POST /visits/:sessionId/suspend`

路径参数：`sessionId`（SessionId）。

#### 用途

当空闲计时（距 `lastActivityAt` 超过空闲阈值）触发时，前端调用此端点将会话挂起。非终态——不写 `endedAt` / `terminalReason`，仅将 `status` 置为 `"suspended"` 并清除 `activeCardId`。

#### 请求

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `sessionId` | SessionId | 是 | 会话 ID |

#### 响应（`suspendVisitResultSchema`）

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `session` | `VisitSession` | 是 | 更新后的会话（status=suspended） |
| `timelineItem` | `TimelineItem` | 是 | `system_event` 类型，eventType=`session_suspended` |

#### Zod Schema

```ts
export const suspendVisitInputSchema = z.object({
  sessionId: sessionIdSchema,
})

export const suspendVisitResultSchema = z.object({
  session: visitSessionSchema,
  timelineItem: timelineItemSchema,
})
```

#### 错误码

| HTTP 状态码 | 错误码 | 说明 |
| --- | --- | --- |
| 404 | `SESSION_NOT_FOUND` | 会话不存在 |
| 422 | `INVALID_STATE` | 会话不处于可挂起状态（非 chatting/analyzing/blocked 等可对话态） |

#### 实现参考

- 定义：`src/features/workbench/api/schemas.ts` — `suspendVisitInputSchema`、`suspendVisitResultSchema`
- Mock handler：`src/mocks/api/handlers/chat-handlers.ts` — `handleSuspendVisit`
- Mock DB：`src/mocks/api/mock-db.ts` — `suspendVisit()` 方法
- 路由：`src/mocks/api/mock-transport.ts` — `POST /visits/:sessionId/suspend`

---

## 3. `interruptedBy` 新增 `"idle"` 值

来源 `messageTimelineItemSchema`（`src/features/workbench/api/timeline-schemas.ts`）。

`MessageTimelineItem.interruptedBy` 原枚举（3 值）：`"emergency"`、`"timeout"`、`"exit"`。

新增值 `"idle"`，表示该消息因空闲超时触发挂起而被中断。

完整枚举（4 值）：

| 值 | 触发场景 |
| --- | --- |
| `emergency` | 急症中断 |
| `timeout` | 全局超时中断 |
| `exit` | 主动退出中断 |
| **`idle`** | **空闲超时挂起中断** |

当流式 AI 回复尚未完成时触发空闲挂起，当前正在生成的消息 timeline item 的 `interruptedBy` 标记为 `"idle"`。

---

## 4. 系统事件 `session_suspended`

来源 `systemEventTimelineItemSchema`（`src/features/workbench/api/timeline-schemas.ts`）。

`system_event` 类型的 `eventType` 新增 `"session_suspended"` 值。

完整 `eventType` 枚举（9 值）：

| 值 | 说明 |
| --- | --- |
| `context_loaded` | 上下文加载完成 |
| `agent_thinking` | AI 正在思考 |
| `lab_result_received` | 检验结果回填 |
| `payment_succeeded` | 支付成功 |
| `drug_purchased` | 药品已购买 |
| `follow_up_started` | 复诊开始 |
| `emergency_dismissed` | 急症解除 |
| `exit_settled` | 退出结算完成 |
| **`session_suspended`** | **会话因空闲超时被挂起** |

`session_suspended` 事件由 `POST /visits/:sessionId/suspend` 产出。其 `title` 通常为"会话已暂停"，`description` 提示患者可直接输入或按复诊流程继续。

---

## 5. `lastActivityAt` 字段

来源 `visitSessionSchema`（`src/features/visits/api/schemas.ts`）。

`VisitSession` 新增 `lastActivityAt` 字段，用于空闲计时和挂起检测。

### 字段定义

| 字段 | 类型 | 必填 | 位置 | 说明 |
| --- | --- | --- | --- | --- |
| `lastActivityAt` | ISO8601 | 否 | `visitSessionBaseSchema` | 最后一次操作时间 |

### 与 `timeoutAt` 的区别

| 字段 | 含义 | 刷新时机 |
| --- | --- | --- |
| `timeoutAt` | 总计时截止时间（绝对 deadline） | 暂停时冻结，恢复时顺延 |
| `lastActivityAt` | 最后一次活动时间（相对空闲基准） | 发消息、提交流程卡、暂停/恢复计时时刷新 |

### 使用方式

空闲计时以 `lastActivityAt` 为基准：`(当前时间 - lastActivityAt) > 空闲阈值` 触发自动挂起。

- 挂起操作**不**刷新 `lastActivityAt`（挂起是空闲的结果，不应重置计时基准）。
- 暂停计时期间 `lastActivityAt` 不变，恢复后当次操作刷新。

### 完整 `VisitSession` 字段表（补充）

（以下字段已列于 `rest-api.md` §5.2，此处仅记录被遗漏的字段）

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `lastActivityAt` | ISO8601 | 否 | 最后一次操作时间，用于空闲计时检测 |
| `timeoutAt` | ISO8601 | 否 | 总计时截止时间（已列于 rest-api.md） |
| `pausedAt` | ISO8601 | 否 | 当前暂停起点（已列于 rest-api.md） |

---

## 6. `ApiTransport.put()` 方法

来源 `ApiTransport` 接口（`src/lib/api/transport.ts`）。

### 接口定义

传输层接口原文档（rest-api.md §1.2）列出 `get/post/patch/delete/stream`，遗漏了 `put` 方法：

```ts
export interface ApiTransport {
  get<T>(path: string, options?: RequestOptions): Promise<T>
  post<T>(path: string, body?: unknown, options?: RequestOptions): Promise<T>
  put<T>(path: string, body?: unknown, options?: RequestOptions): Promise<T>
  patch<T>(path: string, body?: unknown, options?: RequestOptions): Promise<T>
  delete<T>(path: string, options?: RequestOptions): Promise<T>
  stream<TEvent>(
    path: string,
    body: unknown,
    handlers: StreamHandlers<TEvent>,
  ): Promise<void>
}
```

`put` 方法的签名与 `post` / `patch` 一致：`path`、`body?`、`options?`，返回 `Promise<T>`。

### 使用点

| 端点 | 用途 | 引入 Patch |
| --- | --- | --- |
| `PUT /patients/:patientId/addresses/:addressId/default` | 设置默认收货地址 | v5 |
| `PUT /admin/settings` | 更新系统设置 | v8 |

### 实现参考

- 接口定义：`src/lib/api/transport.ts`
- Mock 实现：`src/mocks/api/mock-transport.ts` — `createMockTransport().put`

---

## 7. 地址标签约束

来源 `addressTagSchema`（`src/features/patient/api/address-schemas.ts`）。

### Schema 定义

```ts
export const addressTagSchema = z.string().trim().min(1).max(20)
```

### 约束含义

| 约束 | 说明 |
| --- | --- |
| `min(1)` | 标签不能为空字符串。设置空字符串会触发 `VALIDATION_ERROR`。 |
| `max(20)` | 标签最长 20 个字符（trim 后）。 |

### 设计说明

- 标签为可选字段（`tag: addressTagSchema.optional()`），创建地址时可不传。
- 标签一旦设置，不可通过清空（传空字符串）来移除——由于 `min(1)` 约束，空字符串会校验失败。
- 要修改标签，需提供一个合法的非空新值。

### 使用场景

常见预设标签值：`"家"`、`"公司"`、`"病房"`，但标签为自由文本，不限值集。

### 相关 Schema

| Schema | 字段 | 规则 |
| --- | --- | --- |
| `createAddressInputSchema` | `tag?: addressTagSchema` | 可选，设置时需满足 min(1)/max(20) |
| `updateAddressInputSchema` | `tag?: addressTagSchema` | 可选，设置时需满足 min(1)/max(20) |

---

## 8. 配送地址电话宽松校验

来源 `deliveryAddressSummarySchema`（`src/features/workbench/api/timeline-schemas.ts`）与 `addressSchema`（`src/features/patient/api/address-schemas.ts`）。

### 校验差异

| 字段 | Schema | 校验规则 |
| --- | --- | --- |
| `Address.phone` | `addressSchema.phone` | `z.string().regex(/^1\d{10}$/)` — 严格 11 位大陆手机号 |
| `deliveryAddress.phone` | `deliveryAddressSummarySchema.phone` | `z.string().trim().min(1)` — 仅要求非空字符串 |

### 说明

- `deliveryAddress` 是配送确认时从地址簿**拷贝**的快照摘要，存储在 `medication_fulfillment` 卡片和 `medical-order` 记录中。
- 使用宽松校验是为了保留原始录入值。历史数据中可能存在非常规格式的号码（如含分机号、国际号码等），快照应如实保存源地址的 phone 值，而不应因格式校验失败而拒绝存储。
- 该字段只读不校验；修改地址簿不会回写历史取药卡的 `deliveryAddress`。

### 相关 Schema

```ts
// 地址簿主数据 — 严格校验
export const addressSchema = z.object({
  // ...
  phone: z.string().regex(/^1\d{10}$/, "手机号格式不正确"),
  // ...
})

// 配送地址快照 — 宽松校验（仅非空）
export const deliveryAddressSummarySchema = z.object({
  name: z.string().trim().min(1),
  phone: z.string().trim().min(1),   // 仅需非空，无手机号格式限制
  fullAddress: z.string().trim().min(1),
})

// medication_fulfillment 卡片引用
export const medicationFulfillmentCardSchema = flowCardBaseSchema.extend({
  kind: z.literal("medication_fulfillment"),
  // ...
  deliveryAddress: deliveryAddressSummarySchema.optional(),
})
```

---

## 9. SSE 流事件类型详述

来源 `assistantStreamEventSchema`（`src/features/workbench/api/timeline-schemas.ts`）。

`rest-api.md` §6.2 列出了 7 种 SSE 事件类型及其 payload 字段。本节补充各事件的触发时机与语义，以及典型序列。

### 9.1 事件类型总表

| type | 触发时机 | 语义 |
| --- | --- | --- |
| `delta` | AI 生成回复过程中 | 增量文本块，客户端逐块拼接成完整回复 |
| `message_final` | 一条消息生成完毕 | 完整的 `message` 形态 `TimelineItem`，含 id、role、timestamp |
| `card` | 需要插入流程卡时 | 一条或多条 `FlowCard`，后接 `state` 事件指示状态转移 |
| `state` | 会话状态变化时 | 状态机转移信号，含新 state、status 和当前活动卡 ID |
| `emergency` | 分析中检测到急症风险 | 急症告警，分 `suspected`（疑诊）和 `critical`（危急）两级 |
| `done` | 流式完成 | 发生在 `message_final` 或尾部卡片、状态事件之后 |
| `error` | 流式处理出错 | 流异常终止，含错误码和说明 |

### 9.2 各事件详情

#### `delta`

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `type` | literal `"delta"` | 是 | 事件类型标识 |
| `sessionId` | SessionId | 是 | 会话 ID |
| `requestId` | string(min 1) | 是 | 本轮流式请求 ID |
| `content` | string | 是 | 文本增量片段 |

客户端将所有 `delta` 按序拼接为完整消息内容。`delta` 事件之间无顺序保证之外的关联；每个 `delta` 是独立的、直接可展示的文本片段。

#### `message_final`

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `type` | literal `"message_final"` | 是 | 事件类型标识 |
| `sessionId` | SessionId | 是 | 会话 ID |
| `requestId` | string(min 1) | 是 | 本轮流式请求 ID |
| `item` | `MessageTimelineItem` | 是 | 完整的消息时间线条目 |

`item.role` 为 `"assistant"`，`item.status` 为 `"done"` 表示正文完整，或 `"failed"` 表示正文段异常。`item.content` 为所有 `delta` 的拼接结果，与逐个 `delta` 渲染应一致。若消息被中断，`item.interruptedBy` 携带中断原因。

#### `card`

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `type` | literal `"card"` | 是 | 事件类型标识 |
| `sessionId` | SessionId | 是 | 会话 ID |
| `requestId` | string(min 1) | 是 | 本轮流式请求 ID |
| `card` | `FlowCard` | 是 | 流程卡（9 种 kind 之一） |
| `timelineItem` | `FlowCardTimelineItem` | 否 | 卡片对应的时间线条目 |

`card` 事件通常后跟一个 `state` 事件，指示新卡片导致的状态转移（如进入 `blocked` 状态等待患者操作）。

#### `state`

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `type` | literal `"state"` | 是 | 事件类型标识 |
| `sessionId` | SessionId | 是 | 会话 ID |
| `state` | `VisitMachineState` | 是 | 状态机的内部状态 |
| `status` | `VisitStatus` | 否 | 会话对外状态 |
| `activeCardId` | FlowCardId | 否 | 当前活动/阻塞卡 ID |

当 `status` 变为 `"blocked"` 时，`activeCardId` 应同时出现。`state` 事件可单独出现（如状态转移但不挂卡），也可在 `card` 事件后紧随。

#### `emergency`

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `type` | literal `"emergency"` | 是 | 事件类型标识 |
| `sessionId` | SessionId | 是 | 会话 ID |
| `severity` | `"suspected"` / `"critical"` | 是 | 急症级别 |
| `message` | string(min 1) | 是 | 告警描述（可直接展示） |

`emergency` 事件发出后流通常不会立即终止，但 `done` 可能仍会到达。前端应优先展示急症 Overlay，中止当前操作流。`severity` 两级：

- `suspected`：系统怀疑但不确定，建议提示患者确认。
- `critical`：系统判定危急，必须立即干预。

#### `done`

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `type` | literal `"done"` | 是 | 事件类型标识 |
| `sessionId` | SessionId | 是 | 会话 ID |
| `requestId` | string(min 1) | 是 | 本轮流式请求 ID |

流式正常结束的标记。收到 `done` 后客户端可关闭流连接。`done` 是流中最后一个事件（除非被 `error` 替代）。

#### `error`

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `type` | literal `"error"` | 是 | 事件类型标识 |
| `sessionId` | SessionId | 否 | 会话 ID |
| `requestId` | string | 否 | 本轮流式请求 ID |
| `error` | `ApiError` | 是 | 错误详情，含 code、message |

`error` 事件表示流式处理异常终止，取代 `done` 成为最终事件。客户端应读取 `error.code` 做相应处理（如 `retriable` 为 true 时可重试）。

### 9.3 典型事件序列

**正常一次 AI 回复：**

```
delta × n → message_final → state → done
```

**带流程卡：**

```
delta × n → message_final → card → state → done
```

**多卡（检验决定 + 缴费）：**

```
delta × n → message_final → card(lab_decision) → state → card(payment, lab) → state → done
```

**命中急症：**

```
emergency → done
```

**流式出错：**

```
delta × 2 → error
```

---

## 相关 Patch 引用

| Patch | 关联内容 |
| --- | --- |
| v5 | 地址簿 CRUD 端点、地址标签、配送地址快照。本 patch §7（标签约束）与 §8（配送电话校验）为其补充说明。 |
| v6 | 账单记录查询。本 patch §9（SSE 事件详述）为通用的流式规范补充。 |
| v8 | 管理后台系统设置端点（`PUT /admin/settings`）。本 patch §6（put 方法）为其传输层说明。 |

---

## 验证清单

- [ ] 源码核对：`src/lib/api/types.ts` 的 `visitStatusSchema` 包含 `"suspended"`
- [ ] 源码核对：`src/lib/api/types.ts` 的 `visitMachineStateSchema` 包含 `"suspended"`
- [ ] 源码核对：`src/features/workbench/api/timeline-schemas.ts` 的 `messageTimelineItemSchema.interruptedBy` 包含 `"idle"`
- [ ] 源码核对：`src/features/workbench/api/timeline-schemas.ts` 的 `systemEventTimelineItemSchema.eventType` 包含 `"session_suspended"`
- [ ] 源码核对：`src/features/workbench/api/schemas.ts` 存在 `suspendVisitInputSchema` / `suspendVisitResultSchema` 与对应 parse 函数
- [ ] 源码核对：`src/features/visits/api/schemas.ts` 的 `visitSessionBaseSchema` 包含 `lastActivityAt`
- [ ] 源码核对：`src/lib/api/transport.ts` 的 `ApiTransport` 接口包含 `put` 方法
- [ ] 源码核对：`src/features/patient/api/address-schemas.ts` 的 `addressTagSchema` 为 `z.string().trim().min(1).max(20)`
- [ ] 源码核对：`src/features/workbench/api/timeline-schemas.ts` 的 `deliveryAddressSummarySchema.phone` 为 `z.string().trim().min(1)`
- [ ] Mock 核对：`src/mocks/api/mock-transport.ts` 存在 `PUT /patients/:patientId/addresses/:addressId/default` 路由
- [ ] Mock 核对：`src/mocks/api/mock-transport.ts` 存在 `POST /visits/:sessionId/suspend` 路由
- [ ] Mock 核对：`src/mocks/api/stream-simulator.ts` 的 `assistantStreamEventSchema` 涵盖全部 7 种事件类型
