# REST API Patch v2 — `medicalHistory` 可编辑

日期：2026-06-29

## 变更概述

向 `PATCH /patients/:id/profile` 端点的请求体新增 **可选** 字段 `medicalHistory`，使患者可在个人中心编辑既往病史。

此变更为 **非破坏性扩展（backward compatible）**：现有调用方不传该字段，行为不变。

---

## 受影响端点

### `PATCH /patients/:id/profile`

#### 请求体（变更后）

```jsonc
{
  "patientId": "string (required)",
  "allergies": ["string"],          // optional
  "chronicDiseases": ["string"],    // optional
  "longTermMedications": ["string"],// optional
  "medicalHistory": ["string"]      // optional — 新增
}
```

#### 字段说明

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `patientId` | `string` | ✅ | 患者 ID |
| `allergies` | `string[]` | ❌ | 过敏史条目列表；不传则不修改 |
| `chronicDiseases` | `string[]` | ❌ | 慢性病列表；不传则不修改 |
| `longTermMedications` | `string[]` | ❌ | 长期用药列表；不传则不修改 |
| `medicalHistory` | `string[]` | ❌ | **新增** 既往病史列表；不传则不修改 |

每个数组元素：`string`，trim 后最少 1 字符，空字符串会被 schema 拒绝。

#### 响应体

不变。返回更新后的 `PatientProfile` 对象。

#### 行为语义

- 传入 `medicalHistory: []` → 清空既往病史。
- 传入 `medicalHistory: ["慢性咽炎病史 3 年", "2024年 阑尾炎手术"]` → 替换为新列表。
- 不传 `medicalHistory` 字段 → 保留原值，不做任何修改。

---

## Schema 变更

### `updatePatientProfileInputSchema`

```diff
 export const updatePatientProfileInputSchema = z.object({
   patientId: patientIdSchema,
   allergies: z.array(z.string().trim().min(1)).optional(),
   chronicDiseases: z.array(z.string().trim().min(1)).optional(),
   longTermMedications: z.array(z.string().trim().min(1)).optional(),
+  medicalHistory: z.array(z.string().trim().min(1)).optional(),
 })
```

### `UpdatePatientProfileInput` 类型

由 Zod schema 自动推导，无需手动维护：

```ts
type UpdatePatientProfileInput = z.infer<typeof updatePatientProfileInputSchema>
// { patientId: string; allergies?: string[]; chronicDiseases?: string[]; longTermMedications?: string[]; medicalHistory?: string[] }
```

---

## Mock 层变更

### `mock-db.ts` — `updatePatientProfile`

```diff
 updatePatientProfile(input: {
   patientId: PatientId
   allergies?: string[]
   chronicDiseases?: string[]
   longTermMedications?: string[]
+  medicalHistory?: string[]
 }) {
   // ...
   this.state.contexts[input.patientId] = {
     ...this.state.contexts[input.patientId],
     patient: updated,
     allergies: updated.allergies,
     longTermMedications: updated.longTermMedications,
+    medicalHistory:
+      input.medicalHistory ??
+      this.state.contexts[input.patientId].medicalHistory,
   }
 }
```

`medicalHistory` 存储于 `PatientContext` 级别（非 `PatientProfile`），因此在 context 层面单独合并。

---

## 前端集成

### `ProfilePage.tsx`

- `EditingSection` 联合类型新增 `"medicalHistory"`。
- `handleSave` 回调扩展字段类型，支持传入 `medicalHistory`。
- 原静态 `<ul>` 既往病史替换为 `<EditableChipList>` 组件，数据源为 `context.medicalHistory`。

---

## 兼容性

| 维度 | 评估 |
|------|------|
| 已有前端调用 | 不传 `medicalHistory`，行为不变 ✅ |
| 已有后端验证 | Zod `.optional()` 允许缺失 ✅ |
| 数据库 schema | 无 migration（mock 内存态），生产按实际 DB 补列/字段 |
| 版本协商 | 无需 header 协商，纯 additive 变更 |

---

## 验证

- `pnpm build`（tsc -b + vite）通过，无类型错误。
- Mock 层已支持 `medicalHistory` 的读/写/清空。
- ProfilePage 可进入编辑模式，添加/删除/保存/取消均正常。
