# REST API Patch v7 — 管理后台（Admin Panel）

日期：2026-06-30

## 变更概述

新增管理后台完整 REST API，包括管理员认证（登录/登出/刷新）、仪表盘统计、患者管理、问诊记录查询和系统设置。管理员端点统一以 `/admin` 前缀区分，使用独立的 JWT 令牌体系，与患者端 token 互不影响。

## 新增端点

### `POST /admin/auth/login`

管理员登录，返回访问令牌和用户信息。

#### 请求

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| username | string | 是 | 管理员用户名 |
| password | string | 是 | 管理员密码 |

#### 响应体

| 字段 | 类型 | 说明 |
|------|------|------|
| tokens | AdminTokens | 令牌信息 |
| user | AdminUser | 当前管理员信息 |

#### AdminTokens 字段

| 字段 | 类型 | 说明 |
|------|------|------|
| accessToken | string | JWT 访问令牌 |
| refreshToken | string | 刷新令牌 |
| expiresIn | number | accessToken 有效期（秒） |

#### AdminUser 字段

| 字段 | 类型 | 说明 |
|------|------|------|
| id | string | 管理员 ID |
| username | string | 用户名 |
| role | "super_admin" \| "admin" \| "operator" | 角色 |
| displayName | string | 显示名称 |
| createdAt | string (ISO8601) | 创建时间 |

#### 错误码

| HTTP 状态码 | 错误码 | 说明 |
|-------------|--------|------|
| 401 | INVALID_CREDENTIALS | 用户名或密码错误 |

---

### `POST /admin/auth/logout`

管理员登出，使当前 refreshToken 失效。

#### 请求

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| refreshToken | string | 是 | 需要失效的刷新令牌 |

#### 响应体

| 字段 | 类型 | 说明 |
|------|------|------|
| success | boolean | 固定为 true |

#### 错误码

无额外错误码。即使 refreshToken 无效也返回 success。

---

### `POST /admin/auth/refresh`

刷新管理员访问令牌（rotation 机制，旧 refreshToken 立即失效）。

#### 请求

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| refreshToken | string | 是 | 当前有效的刷新令牌 |

#### 响应体

| 字段 | 类型 | 说明 |
|------|------|------|
| tokens | AdminTokens | 新的令牌信息 |

#### 错误码

| HTTP 状态码 | 错误码 | 说明 |
|-------------|--------|------|
| 401 | INVALID_REFRESH_TOKEN | 刷新令牌无效或已过期 |

---

### `GET /admin/dashboard/stats`

获取仪表盘统计数据。需管理员 Bearer token 鉴权。

#### 请求

无请求体。

#### 响应体

| 字段 | 类型 | 说明 |
|------|------|------|
| totalPatients | number | 患者总数 |
| totalSessions | number | 问诊记录总数 |
| activeSessions | number | 当前进行中的问诊数 |
| todayNewPatients | number | 今日新增患者数 |
| todayNewSessions | number | 今日新增问诊数 |

#### 错误码

| HTTP 状态码 | 错误码 | 说明 |
|-------------|--------|------|
| 401 | UNAUTHORIZED | 未提供有效管理员 token |

---

### `GET /admin/patients`

分页查询患者列表，支持搜索。需管理员 Bearer token 鉴权。

#### 请求

Query Parameters：

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| page | number | 1 | 页码（从 1 开始） |
| pageSize | number | 20 | 每页条数 |
| search | string | — | 可选，按 realName 或 phone 模糊匹配 |

#### 响应体

| 字段 | 类型 | 说明 |
|------|------|------|
| items | AdminPatientItem[] | 患者列表 |
| total | number | 总条数 |
| page | number | 当前页码 |
| pageSize | number | 每页条数 |

#### AdminPatientItem 字段

| 字段 | 类型 | 说明 |
|------|------|------|
| id | string | 患者 ID |
| realName | string | 真实姓名 |
| phone | string | 手机号 |
| gender | "male" \| "female" \| "unknown" | 性别 |
| birthDate | string (YYYY-MM-DD) | 出生日期 |
| createdAt | string (ISO8601) | 注册时间 |
| sessionCount | number | 历史问诊次数 |

#### 错误码

| HTTP 状态码 | 错误码 | 说明 |
|-------------|--------|------|
| 401 | UNAUTHORIZED | 未提供有效管理员 token |

---

### `GET /admin/patients/:id`

获取单个患者的完整 Profile 信息。需管理员 Bearer token 鉴权。

#### 请求

Path Parameters：

| 参数 | 类型 | 说明 |
|------|------|------|
| id | string | 患者 ID |

#### 响应体

返回完整 PatientProfile 对象（复用患者端 `GET /patient/profile` 的响应结构）。

#### 错误码

| HTTP 状态码 | 错误码 | 说明 |
|-------------|--------|------|
| 404 | PATIENT_NOT_FOUND | 患者不存在 |
| 401 | UNAUTHORIZED | 未提供有效管理员 token |

---

### `GET /admin/sessions`

分页查询问诊记录列表，支持按状态和患者筛选。需管理员 Bearer token 鉴权。

#### 请求

Query Parameters：

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| page | number | 1 | 页码（从 1 开始） |
| pageSize | number | 20 | 每页条数 |
| status | string | — | 可选，按问诊状态筛选 |
| patientId | string | — | 可选，按患者 ID 筛选 |

#### 响应体

| 字段 | 类型 | 说明 |
|------|------|------|
| items | AdminSessionItem[] | 问诊记录列表 |
| total | number | 总条数 |
| page | number | 当前页码 |
| pageSize | number | 每页条数 |

#### AdminSessionItem 字段

| 字段 | 类型 | 说明 |
|------|------|------|
| id | string | 会话 ID |
| patientId | string | 患者 ID |
| patientName | string | 患者姓名 |
| title | string | 问诊标题 |
| status | string | 会话状态 |
| createdAt | string (ISO8601) | 创建时间 |
| updatedAt | string (ISO8601) | 最后更新时间 |

#### 错误码

| HTTP 状态码 | 错误码 | 说明 |
|-------------|--------|------|
| 401 | UNAUTHORIZED | 未提供有效管理员 token |

---

### `GET /admin/sessions/:id`

获取单个问诊记录的完整详情。需管理员 Bearer token 鉴权。

#### 请求

Path Parameters：

| 参数 | 类型 | 说明 |
|------|------|------|
| id | string | 会话 ID |

#### 响应体

返回完整 VisitSession 对象（复用患者端 `GET /sessions/:id` 的响应结构，包含完整 timeline）。

#### 错误码

| HTTP 状态码 | 错误码 | 说明 |
|-------------|--------|------|
| 404 | SESSION_NOT_FOUND | 问诊记录不存在 |
| 401 | UNAUTHORIZED | 未提供有效管理员 token |

---

### `GET /admin/settings`

获取系统设置。需管理员 Bearer token 鉴权。

#### 请求

无请求体。

#### 响应体

| 字段 | 类型 | 说明 |
|------|------|------|
| siteName | string | 站点名称 |
| maxConcurrentSessions | number | 单患者最大并发问诊数 |
| sessionTimeoutMinutes | number | 问诊超时时间（分钟） |
| enableRegistration | boolean | 是否开放注册 |

#### 错误码

| HTTP 状态码 | 错误码 | 说明 |
|-------------|--------|------|
| 401 | UNAUTHORIZED | 未提供有效管理员 token |

---

### `PUT /admin/settings`

更新系统设置（支持部分更新）。需管理员 Bearer token 鉴权。

#### 请求

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| siteName | string | 否 | 站点名称 |
| maxConcurrentSessions | number | 否 | 单患者最大并发问诊数 |
| sessionTimeoutMinutes | number | 否 | 问诊超时时间（分钟） |
| enableRegistration | boolean | 否 | 是否开放注册 |

#### 响应体

返回完整的、更新后的 SystemSettings 对象（结构同 `GET /admin/settings` 响应）。

#### 错误码

| HTTP 状态码 | 错误码 | 说明 |
|-------------|--------|------|
| 400 | INVALID_SETTINGS | 设置值无效（如负数、空字符串等） |
| 401 | UNAUTHORIZED | 未提供有效管理员 token |

---

## Zod Schema

```ts
import { z } from "zod/v4"

// ─── 管理员认证 ───

export const adminLoginInputSchema = z.object({
  username: z.string().trim().min(1),
  password: z.string().trim().min(1),
})

export const adminRoleSchema = z.enum(["super_admin", "admin", "operator"])

export const adminUserSchema = z.object({
  id: z.string().trim().min(1),
  username: z.string().trim().min(1),
  role: adminRoleSchema,
  displayName: z.string().trim().min(1),
  createdAt: z.string().datetime(),
})

export const adminTokensSchema = z.object({
  accessToken: z.string().trim().min(1),
  refreshToken: z.string().trim().min(1),
  expiresIn: z.number().int().positive(),
})

export const adminLoginResultSchema = z.object({
  tokens: adminTokensSchema,
  user: adminUserSchema,
})

export const adminLogoutInputSchema = z.object({
  refreshToken: z.string().trim().min(1),
})

export const adminLogoutResultSchema = z.object({
  success: z.literal(true),
})

export const adminRefreshInputSchema = z.object({
  refreshToken: z.string().trim().min(1),
})

export const adminRefreshResultSchema = z.object({
  tokens: adminTokensSchema,
})

// ─── 仪表盘 ───

export const dashboardStatsSchema = z.object({
  totalPatients: z.number().int().min(0),
  totalSessions: z.number().int().min(0),
  activeSessions: z.number().int().min(0),
  todayNewPatients: z.number().int().min(0),
  todayNewSessions: z.number().int().min(0),
})

// ─── 患者管理 ───

export const adminPatientQuerySchema = z.object({
  page: z.coerce.number().int().positive().default(1),
  pageSize: z.coerce.number().int().positive().max(100).default(20),
  search: z.string().trim().optional(),
})

export const adminPatientGenderSchema = z.enum(["male", "female", "unknown"])

export const adminPatientItemSchema = z.object({
  id: z.string().trim().min(1),
  realName: z.string().trim().min(1),
  phone: z.string().trim().min(1),
  gender: adminPatientGenderSchema,
  birthDate: z.string().trim().min(1),
  createdAt: z.string().datetime(),
  sessionCount: z.number().int().min(0),
})

export const adminPatientListResultSchema = z.object({
  items: z.array(adminPatientItemSchema),
  total: z.number().int().min(0),
  page: z.number().int().positive(),
  pageSize: z.number().int().positive(),
})

// ─── 问诊记录管理 ───

export const adminSessionQuerySchema = z.object({
  page: z.coerce.number().int().positive().default(1),
  pageSize: z.coerce.number().int().positive().max(100).default(20),
  status: z.string().trim().optional(),
  patientId: z.string().trim().optional(),
})

export const adminSessionItemSchema = z.object({
  id: z.string().trim().min(1),
  patientId: z.string().trim().min(1),
  patientName: z.string().trim().min(1),
  title: z.string().trim().min(1),
  status: z.string().trim().min(1),
  createdAt: z.string().datetime(),
  updatedAt: z.string().datetime(),
})

export const adminSessionListResultSchema = z.object({
  items: z.array(adminSessionItemSchema),
  total: z.number().int().min(0),
  page: z.number().int().positive(),
  pageSize: z.number().int().positive(),
})

// ─── 系统设置 ───

export const systemSettingsSchema = z.object({
  siteName: z.string().trim().min(1),
  maxConcurrentSessions: z.number().int().positive(),
  sessionTimeoutMinutes: z.number().int().positive(),
  enableRegistration: z.boolean(),
})

export const updateSystemSettingsInputSchema = systemSettingsSchema.partial()

export const updateSystemSettingsResultSchema = systemSettingsSchema
```

## Mock 层变更

- `mock-db.ts`：新增 `adminUsers` 状态表；种子账号 `admin` / `admin123`（role: super_admin）；新增 6 个 admin 方法：`adminLogin()`、`adminLogout()`、`adminRefreshToken()`、`getDashboardStats()`、`listPatients(query)`、`listSessions(query)`
- `handlers/admin-handlers.ts`：新文件，10 个 handler 函数对应上述 10 个端点
- `mock-transport.ts`：新增 `/admin/*` 路由匹配规则，管理员端点的 Bearer token 校验独立于患者端

## 前端集成

- **API 层**：`src/features/admin/api/admin-api.ts` — 封装所有 admin 端点调用
- **Store**：`src/features/admin/store/admin-auth-store.ts` — Zustand store 管理管理员认证状态和 token 持久化
- **Guard**：`src/features/admin/components/AdminGuard.tsx` — 路由守卫，未登录重定向至 /admin/login
- **Layout**：`src/features/admin/components/AdminShell.tsx`、`AdminSidebar.tsx` — 管理后台整体布局和侧边栏导航
- **Pages**：`src/pages/admin/` 目录下：
  - `AdminLoginPage.tsx` — 管理员登录页
  - `DashboardPage.tsx` — 仪表盘（统计卡片）
  - `PatientListPage.tsx` — 患者列表（分页 + 搜索）
  - `SessionListPage.tsx` — 问诊记录列表（分页 + 状态筛选）
  - `SettingsPage.tsx` — 系统设置（读取 + 表单修改）
- **Routes**：`src/app/router.tsx` 新增 `/admin/*` 路由组，AdminGuard 包裹

## 验证清单

- [ ] pnpm build 无错误
- [ ] pnpm lint 通过
- [ ] /admin/login 页面可用 admin/admin123 登录
- [ ] 登录后 dashboard 展示统计数据
- [ ] 患者列表支持分页和搜索
- [ ] 问诊记录支持按状态筛选
- [ ] 系统设置可读取和修改
- [ ] 管理员 token 与患者 token 互不影响
