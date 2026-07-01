# REST API Patch v7：医嘱记录查询（GET /medical-orders）

日期：2026-06-30

## 变更概述

新增 `GET /medical-orders` 端点，聚合所有历史问诊中已完成/已确认的医嘱记录。

## 新增端点

### `GET /medical-orders`

聚合当前患者所有历史问诊中已完成（completed/confirmed）的医嘱和用药记录，按 `handledAt` 倒序排列。

#### 请求

无请求体。鉴权通过 Bearer token 识别当前患者。

#### 响应体

| 字段 | 类型 | 说明 |
|------|------|------|
| items | MedicalOrderRecord[] | 按 handledAt 降序 |

#### MedicalOrderRecord 字段

| 字段 | 类型 | 说明 |
|------|------|------|
| recordId | string | 记录 ID（对应卡片 ID） |
| sessionId | string | 关联会话 ID |
| sessionTitle | string | 会话标题 |
| kind | "advice" \| "medication" | 记录类型 |
| advices | string[] | （kind=advice）医嘱建议 |
| watchItems | string[] | （kind=advice）需观察项目 |
| followUpRecommendation | string | （kind=advice）随访建议 |
| medications | MedicationItem[] | （kind=medication）药品列表 |
| fulfillmentStatus | "pending" \| "confirmed" \| "completed" | （kind=medication）履行状态 |
| deliveryAddress | DeliveryAddressSummary | （kind=medication）配送地址 |
| handledAt | string (ISO8601) | 处理时间 |
| createdAt | string (ISO8601) | 创建时间 |

## Zod Schema

```ts
export const medicalOrderKindSchema = z.enum(["advice", "medication"])

export const medicationItemSchema = z.object({
  name: z.string().trim().min(1),
  spec: z.string().trim().min(1),
  quantity: z.number().int().positive(),
  dosage: z.string().trim().min(1),
  days: z.number().int().positive(),
  price: z.number().min(0),
})

export const deliveryAddressSummarySchema = z.object({
  name: z.string().trim().min(1),
  phone: z.string().trim().min(1),
  fullAddress: z.string().trim().min(1),
})

export const medicalOrderRecordSchema = z.object({
  recordId: z.string().trim().min(1),
  sessionId: sessionIdSchema,
  sessionTitle: z.string().trim().min(1),
  kind: medicalOrderKindSchema,
  // advice 专属字段
  advices: z.array(z.string().trim().min(1)).optional(),
  watchItems: z.array(z.string().trim().min(1)).optional(),
  followUpRecommendation: z.string().trim().min(1).optional(),
  // medication 专属字段
  medications: z.array(medicationItemSchema).optional(),
  fulfillmentStatus: z.enum(["pending", "confirmed", "completed"]).optional(),
  deliveryAddress: deliveryAddressSummarySchema.optional(),
  // 通用
  handledAt: z.string().datetime(),
  createdAt: z.string().datetime(),
})

export const listMedicalOrdersResultSchema = z.object({
  items: z.array(medicalOrderRecordSchema),
})
```

## 数据来源

从会话 timeline 中聚合以下卡片：

- `advice_only`（status=completed，即患者已确认已知晓）
- `medication_fulfillment`（fulfillmentStatus=completed 或 confirmed）

## 前端实现

- API 层：`src/features/medical-orders/api/`（types.ts、schemas.ts、index.ts）
- 页面：`/medical-orders`（MedicalOrdersPage）
- 路由：HomeLayout child
- 导航：ProfilePage 功能入口 → 医嘱记录

## 验证清单

- [ ] pnpm build
- [ ] pnpm lint
- [ ] 页面展示 mock 数据
- [ ] ProfilePage 入口可点击跳转
- [ ] /medical-orders 路由访问正常
