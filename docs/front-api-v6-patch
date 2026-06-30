# REST API Patch v6 — 账单记录查询

日期：2026-06-30

## 变更概述

新增 `GET /billing/records` 端点，聚合所有历史支付记录。

## 新增端点

### `GET /billing/records`

聚合当前患者所有会话中的支付卡片记录。

#### 请求

无请求体。鉴权通过 Bearer token 识别当前患者。

#### 响应体

| 字段 | 类型 | 说明 |
|------|------|------|
| items | BillingRecord[] | 按 createdAt 降序 |

#### BillingRecord 字段

| 字段 | 类型 | 说明 |
|------|------|------|
| paymentId | string | 支付 ID |
| sessionId | string | 关联会话 ID |
| sessionTitle | string | 会话标题（chiefComplaint/diagnosis） |
| purpose | "lab" \| "medication" | 缴费用途 |
| items | {name, amount, quantity?}[] | 缴费项目 |
| totalAmount | number | 总金额 |
| insuranceAmount | number | 医保 |
| selfPayAmount | number | 自费 |
| paymentStatus | PaymentStatus | 支付状态 |
| createdAt | string (ISO8601) | 创建时间 |

## Zod Schema

```ts
const billingRecordSchema = z.object({
  paymentId: z.string().trim().min(1),
  sessionId: sessionIdSchema,
  sessionTitle: z.string().trim().min(1),
  purpose: z.enum(["lab", "medication"]),
  items: z.array(z.object({
    name: z.string().trim().min(1),
    amount: z.number().min(0),
    quantity: z.number().int().positive().optional(),
  })),
  totalAmount: z.number().min(0),
  insuranceAmount: z.number().min(0),
  selfPayAmount: z.number().min(0),
  paymentStatus: paymentStatusSchema,
  createdAt: z.string().datetime(),
})

const listBillingRecordsResultSchema = z.object({
  items: z.array(billingRecordSchema),
})
```

## Mock 层变更

- `mock-db.ts`: 新增 `listBillingRecords()` 方法
- `handlers/billing-handlers.ts`: 新增文件
- `mock-transport.ts`: 注册 `GET /billing/records`
- `fixtures/timeline.ts`: `mockCompletedTimeline` 新增已支付 payment card

## 前端集成

- 新增 feature: `src/features/billing/`
- 新增页面: `src/pages/home/BillingPage.tsx`
- 路由: `/billing` (HomeLayout child)
- 导航: ProfilePage → 账单记录行

## 验证清单

- [x] pnpm build
- [x] pnpm lint（无新增 lint 错误）
- [x] 页面展示 mock 数据
- [x] 筛选 tab 切换正常
- [x] ProfilePage 入口可点击跳转
