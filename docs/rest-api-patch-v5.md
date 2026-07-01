# REST API Patch v5 — 地址簿与药品配送地址

日期：2026-06-30

## 变更概述

为药品配送补充患者收货地址簿：

- 患者可维护最多 10 条收货地址，支持新增、编辑、删除、设置默认。
- `POST /visits/:sessionId/fulfillment` 在 `mode=delivery` 时必须提交 `addressId`。
- `medication_fulfillment` 卡片可返回 `deliveryAddress` 摘要，用于确认态和回看态展示。

## 新增端点

### `GET /patients/:patientId/addresses`

查询患者收货地址列表。

响应体：

```jsonc
{
  "addresses": [
    {
      "id": "addr-1",
      "patientId": "patient-mock-001",
      "name": "李明",
      "phone": "13800002468",
      "province": "辽宁省",
      "city": "沈阳市",
      "district": "浑南区",
      "detail": "创新路195号东软软件园B4座3楼",
      "isDefault": true,
      "tag": "公司",
      "createdAt": "2026-06-01T00:00:00.000Z",
      "updatedAt": "2026-06-01T00:00:00.000Z"
    }
  ]
}
```

### `POST /patients/:patientId/addresses`

新增收货地址。若这是患者第一条地址，服务端自动设为默认；若请求 `isDefault=true`，服务端取消其他默认地址。

请求体：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `patientId` | `string` | 是 | 患者 ID，应与路径一致 |
| `name` | `string` | 是 | 收件人，1-20 字 |
| `phone` | `string` | 是 | 11 位大陆手机号 |
| `province` | `string` | 是 | 省份 |
| `city` | `string` | 是 | 城市 |
| `district` | `string` | 是 | 区县 |
| `detail` | `string` | 是 | 详细地址，1-200 字 |
| `isDefault` | `boolean` | 否 | 是否设为默认，默认 `false` |
| `tag` | `"家" | "公司" | "医院" | "其他"` | 否 | 地址标签 |

响应体：`Address`。

错误码：

| HTTP 状态码 | 错误码 | 说明 |
| --- | --- | --- |
| 400 | `ADDRESS_LIMIT_EXCEEDED` | 地址数量已达 10 条 |
| 422 | `VALIDATION_ERROR` | 请求体校验失败 |

### `PATCH /patients/:patientId/addresses/:addressId`

更新收货地址。字段均可选，提交 `isDefault=true` 时取消其他默认地址。

响应体：更新后的 `Address`。

错误码：

| HTTP 状态码 | 错误码 | 说明 |
| --- | --- | --- |
| 404 | `ADDRESS_NOT_FOUND` | 地址不存在或不属于该患者 |
| 422 | `VALIDATION_ERROR` | 请求体校验失败 |

### `DELETE /patients/:patientId/addresses/:addressId`

删除收货地址。若删除的是默认地址，且仍有地址剩余，服务端自动把列表第一条设为默认。

响应体：

```jsonc
{
  "success": true
}
```

错误码：

| HTTP 状态码 | 错误码 | 说明 |
| --- | --- | --- |
| 404 | `ADDRESS_NOT_FOUND` | 地址不存在或不属于该患者 |

### `PUT /patients/:patientId/addresses/:addressId/default`

设置默认收货地址。

响应体：更新后的 `Address`。

错误码：

| HTTP 状态码 | 错误码 | 说明 |
| --- | --- | --- |
| 404 | `ADDRESS_NOT_FOUND` | 地址不存在或不属于该患者 |

## 已有端点变更

### `POST /visits/:sessionId/fulfillment`

请求体新增 `addressId`：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `sessionId` | `string` | 是 | 会话 ID |
| `cardId` | `string` | 是 | 取药卡 ID |
| `mode` | `"pickup" | "delivery"` | 是 | 自取或配送 |
| `addressId` | `string` | 条件必填 | `mode=delivery` 时必填 |

新增错误码：

| HTTP 状态码 | 错误码 | 说明 |
| --- | --- | --- |
| 400 | `ADDRESS_REQUIRED` | 配送模式未提交 `addressId` |
| 404 | `ADDRESS_NOT_FOUND` | 地址不存在或不属于会话患者 |

## 卡片 Schema 变更

`medication_fulfillment` 卡片新增可选字段：

```ts
deliveryAddress?: {
  name: string
  phone: string
  fullAddress: string
}
```

该字段只保存配送地址摘要，不替代地址簿主数据。患者后续修改地址簿不会回写历史取药卡。

## Mock 层摘要

- `mock-db` 新增 `addresses: Record<PatientId, Address[]>` 状态和 5 个地址方法。
- `mock-transport` 新增 `PUT` 方法与 5 条地址路由。
- 配送确认时，mock 校验 `addressId` 并把地址摘要写入取药卡。

## 兼容性

- 自取流程无行为变化。
- 旧的配送提交若不带 `addressId` 将被拒绝；前端已改为先弹出地址选择器，确认后再提交。
- HTTP 后端需要同步实现 `PUT` transport 对应端点；mock 与真实 HTTP 共用同一套 Zod schema。
