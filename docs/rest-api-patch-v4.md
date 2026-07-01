# REST API Patch v4 — 会话标题生成

日期：2026-06-29

## 变更概述

新增 **会话标题生成** 端点，由后端调用大模型基于对话内容总结出简短标题，替代之前直接使用患者第一句话（`chiefComplaint`）作为会话名称的做法。

变更涉及：

1. `VisitSummary` 对象新增 `title` 字段（AI 生成的标题）
2. 新增 `POST /visits/:sessionId/generate-title` 端点
3. 前端首轮 AI 回复完成后自动触发标题生成

---

## Schema 变更

### `VisitSummary` 新增字段

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `title` | `string` | ❌ | AI 生成的问诊记录标题，由后端调用大模型总结。前端展示优先级：`title > chiefComplaint > "未命名问诊"` |

完整 `VisitSummary` 结构：

```jsonc
{
  "title": "发热伴咳嗽3天",        // 新增：AI 生成标题
  "chiefComplaint": "我发烧了三天还一直咳嗽", // 原有：患者第一句原文
  "diagnosis": "上呼吸道感染",
  "treatmentSummary": "...",
  "lastMessage": "..."
}
```

---

## 新增端点

### `POST /visits/:sessionId/generate-title`

调用后端 LLM 基于对话上下文生成一个简短问诊标题。

#### 路径参数

| 参数 | 类型 | 说明 |
|------|------|------|
| `sessionId` | `string` | 问诊会话 ID |

#### 请求体

```jsonc
{
  "sessionId": "visit-abc123"  // 与路径参数一致，用于服务端二次校验
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `sessionId` | `string` | ✅ | 问诊会话 ID（需与路径参数一致） |

#### 响应体

```jsonc
// 200 OK
{
  "sessionId": "visit-abc123",
  "title": "发热伴咳嗽3天"     // 1-50 字符，已 trim
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| `sessionId` | `string` | 回显的会话 ID |
| `title` | `string` | 生成的标题，1-50 字符 |

#### 错误码

| HTTP 状态码 | 错误码 | 说明 |
|-------------|--------|------|
| 404 | `SESSION_NOT_FOUND` | 会话不存在 |
| 422 | `VALIDATION_ERROR` | 请求体校验失败 |
| 409 | `TITLE_ALREADY_EXISTS` | 会话已有标题（幂等保护，可选实现） |
| 503 | `LLM_UNAVAILABLE` | 大模型服务不可用（降级：返回空或使用 chiefComplaint 截断） |

#### 幂等性

- 若会话已有 `title`，后端可选择：
  - 返回已有 title（幂等）
  - 重新生成并覆盖（允许 regenerate 语义）
- 前端通过 `useSessionTitleGeneration` hook 保证同一 session 只触发一次

---

## 后端实现要求

### 标题生成策略

后端调用 LLM 时应提供以下上下文：

| 输入 | 来源 | 说明 |
|------|------|------|
| 患者消息 | timeline 中 `role: "patient"` 的消息 | 主要依据 |
| 助手消息 | timeline 中 `role: "assistant"` 的前 2 条 | 辅助理解语境 |
| 已有诊断 | `session.summary.diagnosis` | 若已有诊断，优先基于诊断生成 |

### 标题规范

| 规则 | 说明 |
|------|------|
| 长度 | 1-50 字符 |
| 格式 | 简短中文短语，无标点结尾 |
| 内容 | 概括症状 + 时间线索，或诊断名称 |
| 示例 | "发热伴咳嗽3天"、"反复腹痛一周"、"上呼吸道感染" |

### LLM Prompt 建议

```text
你是医疗问诊记录标题生成器。根据以下对话内容，生成一个简短的中文标题（不超过50字）。
标题应概括患者的主要症状和持续时间，或已确定的诊断。
格式示例：发热伴咳嗽3天、反复头痛一周、急性胃肠炎

对话内容：
{对话文本}

标题：
```

---

## 前端集成

### 触发时机

`useSessionTitleGeneration` hook 在以下条件**全部满足**时自动触发：

1. `isStreaming` 从 `true` → `false`（一轮 AI 回复刚完成）
2. 当前 session 的 `summary.title` 为空（尚未生成过标题）
3. `session.askRound >= 1`（至少有一轮对话）
4. 每个 `sessionId` 只触发一次（ref 防重）

### 展示优先级

```ts
const title = session.summary.title ?? session.summary.chiefComplaint ?? "未命名问诊"
```

### 缓存更新

标题生成成功后：

1. **乐观更新** session query cache 中的 `summary.title` 字段
2. **invalidate** visits list 缓存（历史列表同步刷新）

### Hook 签名

```ts
function useSessionTitleGeneration(sessionId: string, isStreaming: boolean): void
```

### 集成位置

在 `useWorkbenchSession` hook 中调用：

```ts
// ---- 会话标题生成 ----
useSessionTitleGeneration(sessionId, isStreaming)
```

---

## Mock 实现

Mock 层使用基于关键词的启发式算法模拟 LLM 标题生成：

1. **症状关键词映射**：扫描患者消息中的症状关键词（发烧→发热、咳嗽→咳嗽、头痛→头痛等）
2. **时间线索提取**：正则匹配 `N天/周/月/小时` 格式
3. **诊断优先**：若 `session.summary.diagnosis` 已有值，直接用诊断作为标题
4. **组合规则**：
   - 有症状 + 有时间 → `"发热、咳嗽3天"`
   - 有症状无时间 → `"发热、咳嗽问诊"`
   - 无症状 → 截断 chiefComplaint（≤15 字直接用，>15 字截断 13 字 + "…"）
   - 兜底 → `"问诊记录"`

Mock 生成后将 title 写入 session summary 并返回。

---

## 兼容性

| 维度 | 评估 |
|------|------|
| 已有端点 | `GET /visits` 和 `GET /visits/:sessionId` 返回的 summary 现包含可选 `title` 字段；不传时前端 fallback 到 chiefComplaint ✅ |
| 旧会话 | 历史会话 `title` 为 `undefined`，前端自动 fallback ✅ |
| Mock 层 | Mock 使用关键词启发式生成；生产环境由后端 LLM 生成 ✅ |
| 网络失败 | 标题生成失败不影响问诊流程，UI 继续展示 chiefComplaint ✅ |
| 数据库 | `sessions` 表 `summary` JSON 字段新增 `title` key，无需 migration（JSON 动态扩展） |

---

## 验证

- [ ] `POST /visits/:sessionId/generate-title` — 正常会话返回 200 + 生成的 title
- [ ] `POST /visits/:sessionId/generate-title` — 不存在的 sessionId 返回 404
- [ ] `POST /visits/:sessionId/generate-title` — 请求体缺少 sessionId 返回 422
- [ ] 生成的 title 长度在 1-50 字符范围内
- [ ] 生成后 `GET /visits/:sessionId` 返回的 summary 包含新 title
- [ ] 生成后 `GET /visits` 列表中对应会话的 summary 包含新 title
- [ ] 前端首轮 AI 回复完成后自动触发标题生成（无需手动操作）
- [ ] 前端同一 session 不会重复触发标题生成
- [ ] 已有 title 的 session 不会再次触发
- [ ] 标题生成失败时 UI 不受影响，继续展示 chiefComplaint
- [ ] 历史会话（无 title）正常展示 chiefComplaint 作为标题
- [ ] `SessionCard` 组件正确按优先级展示：title > chiefComplaint > "未命名问诊"
