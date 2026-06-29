# REST API Patch v3 — JWT 认证体系

日期：2026-06-29

## 变更概述

引入基于 JWT 的认证体系，采用 **accessToken + refreshToken 双令牌模式**：

- **accessToken**：短期令牌（15 分钟），携带于每次请求的 `Authorization` header。
- **refreshToken**：长期令牌（7 天），不透明字符串，服务端存储，**单次使用后轮换（rotation）**。

此变更为所有需鉴权的端点提供统一身份认证基础。现有无需鉴权的端点（如健康检查）不受影响。

---

## 新增端点

### `POST /auth/register`

注册新用户，成功后直接签发令牌对（免二次登录）。

#### 请求体

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `phone` | `string` | ✅ | 手机号，11 位中国大陆号码 |
| `password` | `string` | ✅ | 密码，最少 8 字符 |
| `realName` | `string` | ❌ | 真实姓名；不传则留空，后续可补填 |

#### 响应体

```jsonc
// 201 Created
{
  "accessToken": "eyJhbGciOiJIUzI1NiIs...",
  "refreshToken": "dGhpcyBpcyBhIHJlZnJlc2g...",
  "expiresIn": 900,           // accessToken 有效期（秒）
  "user": {
    "userId": "u_abc123",
    "patientId": "p_xyz789",  // 注册时自动创建关联患者档案
    "phone": "13800138000",
    "realName": "张三"         // 若未传则为 null
  }
}
```

#### 错误码

| HTTP 状态码 | 错误码 | 说明 |
|-------------|--------|------|
| 409 | `AUTH_PHONE_EXISTS` | 手机号已注册 |
| 422 | `VALIDATION_ERROR` | 请求体校验失败（密码过短、手机号格式错误等） |

---

### `POST /auth/login`

使用手机号 + 密码登录，签发新令牌对。

#### 请求体

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `phone` | `string` | ✅ | 手机号 |
| `password` | `string` | ✅ | 密码 |

#### 响应体

```jsonc
// 200 OK
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

#### 错误码

| HTTP 状态码 | 错误码 | 说明 |
|-------------|--------|------|
| 401 | `AUTH_INVALID_CREDENTIALS` | 手机号或密码错误 |
| 429 | `RATE_LIMITED` | 请求频率超限 |

---

### `POST /auth/refresh`

使用 refreshToken 换取新的令牌对。旧 refreshToken 在使用后**立即失效**（rotation）。

#### 请求体

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `refreshToken` | `string` | ✅ | 当前持有的 refreshToken |

#### 响应体

```jsonc
// 200 OK
{
  "accessToken": "eyJhbGciOiJIUzI1NiIs...",   // 新 accessToken
  "refreshToken": "bmV3IHJlZnJlc2ggdG9rZW4...", // 新 refreshToken（旧的已失效）
  "expiresIn": 900
}
```

#### 错误码

| HTTP 状态码 | 错误码 | 说明 |
|-------------|--------|------|
| 401 | `AUTH_REFRESH_INVALID` | refreshToken 无效或已被使用（触发 token theft 防护） |
| 401 | `AUTH_REFRESH_EXPIRED` | refreshToken 已过期（超过 7 天） |

---

### `POST /auth/logout`

注销当前会话，使服务端持有的 refreshToken 失效。

#### 请求体

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `refreshToken` | `string` | ✅ | 需要失效的 refreshToken |

#### 响应体

```jsonc
// 204 No Content
// （无响应体）
```

#### 错误码

| HTTP 状态码 | 错误码 | 说明 |
|-------------|--------|------|
| 401 | `AUTH_REFRESH_INVALID` | refreshToken 不存在或已失效（幂等处理，仍返回 204 亦可） |

> 设计建议：logout 端点建议做**幂等处理**——即使 token 已失效也返回 204，避免前端退出流程异常。

---

## Token 规格

### accessToken

- **格式**：JWT（HS256 或 RS256，由后端决定签名算法）
- **传输方式**：HTTP header `Authorization: Bearer <accessToken>`
- **有效期**：900 秒（15 分钟）

#### JWT Payload 结构

```jsonc
{
  "sub": "u_abc123",        // userId
  "patientId": "p_xyz789",
  "phone": "13800138000",
  "iat": 1751155200,        // 签发时间（Unix 秒）
  "exp": 1751156100         // 过期时间（Unix 秒），iat + 900
}
```

### refreshToken

- **格式**：不透明字符串（opaque），建议 ≥ 32 字节 Base64 URL-safe 编码
- **存储**：服务端持久化（数据库/Redis），关联 userId
- **有效期**：604800 秒（7 天）
- **使用规则**：单次使用，使用后签发新 token 并使旧 token 失效

---

## 错误码定义

统一错误响应格式：

```jsonc
{
  "error": {
    "code": "AUTH_INVALID_CREDENTIALS",
    "message": "手机号或密码错误"
  }
}
```

| 错误码 | HTTP 状态码 | 触发场景 |
|--------|-------------|----------|
| `AUTH_PHONE_EXISTS` | 409 | 注册时手机号已存在 |
| `AUTH_INVALID_CREDENTIALS` | 401 | 登录时手机号或密码不匹配 |
| `AUTH_TOKEN_EXPIRED` | 401 | accessToken 过期（JWT exp 校验失败） |
| `AUTH_REFRESH_INVALID` | 401 | refreshToken 无效、已被使用或已被撤销 |
| `AUTH_REFRESH_EXPIRED` | 401 | refreshToken 超过 7 天有效期 |
| `RATE_LIMITED` | 429 | 超出速率限制 |
| `VALIDATION_ERROR` | 422 | 请求体字段校验失败 |

---

## 后端安全要求

| 要求 | 说明 |
|------|------|
| 密码存储 | bcrypt，cost factor ≥ 12 |
| refreshToken 单次使用 | 每次 `/auth/refresh` 调用后，旧 token 立即标记为已使用 |
| Token theft 防护 | 若检测到**已失效** refreshToken 被重复使用，立即撤销该用户名下**全部** refreshToken，强制重新登录 |
| Rate limiting | `/auth/login` 和 `/auth/register` 端点限制 **5 req/min/IP** |
| JWT 签名密钥 | 至少 256-bit 随机密钥；生产环境建议 RS256 + 密钥轮换 |
| 传输安全 | 所有 auth 端点必须 HTTPS |

### Token theft 防护流程

```text
客户端 A（合法）使用 refreshToken_1 → 签发 refreshToken_2，refreshToken_1 标记已用
攻击者 B（窃取）使用 refreshToken_1 → 服务端检测到已失效 token 被重用
  → 撤销该用户全部 refreshToken（包括 refreshToken_2）
  → 合法客户端下次 refresh 失败，强制重新登录
```

---

## 前端集成

### Token 注入（ky beforeRequest hook）

```ts
// src/api/client.ts
import ky from "ky"
import { useAuthStore } from "@/stores/auth"

export const apiClient = ky.create({
  prefixUrl: import.meta.env.VITE_API_BASE_URL,
  hooks: {
    beforeRequest: [
      (request) => {
        const { accessToken } = useAuthStore.getState()
        if (accessToken) {
          request.headers.set("Authorization", `Bearer ${accessToken}`)
        }
      },
    ],
  },
})
```

### 401 静默刷新 + 重试

```ts
// src/api/client.ts（续）
import ky from "ky"

export const apiClient = ky.create({
  // ...
  hooks: {
    afterResponse: [
      async (request, options, response) => {
        if (response.status === 401) {
          const { refreshToken, setTokens, clearAuth } = useAuthStore.getState()
          if (!refreshToken) {
            clearAuth()
            window.location.href = "/login"
            return response
          }

          try {
            const res = await ky
              .post("auth/refresh", {
                prefixUrl: import.meta.env.VITE_API_BASE_URL,
                json: { refreshToken },
              })
              .json<{ accessToken: string; refreshToken: string; expiresIn: number }>()

            setTokens(res.accessToken, res.refreshToken)

            // 用新 token 重试原请求
            request.headers.set("Authorization", `Bearer ${res.accessToken}`)
            return ky(request, options)
          } catch {
            clearAuth()
            window.location.href = "/login"
            return response
          }
        }
      },
    ],
  },
})
```

### SSE 流式请求

SSE 连接建立时需在请求头中携带 Bearer token：

```ts
// src/api/sse.ts
const { accessToken } = useAuthStore.getState()

const eventSource = new EventSource(url, {
  headers: {
    Authorization: `Bearer ${accessToken}`,
  },
})
```

> 注：原生 `EventSource` 不支持自定义 header，前端使用 `eventsource-parser` + `fetch` 实现 SSE，因此可正常注入 Authorization header。

---

## 兼容性

| 维度 | 评估 |
|------|------|
| 已有无鉴权端点 | 不受影响；后端按路由粒度配置鉴权中间件 ✅ |
| Mock 层 | Mock server 跳过 JWT 校验，直接信任请求中的 userId；生产环境由中间件统一校验 ✅ |
| 数据库 schema | 新增 `users` 表（userId, phone, passwordHash, realName, createdAt）和 `refresh_tokens` 表（tokenHash, userId, expiresAt, usedAt） |
| 前端路由守卫 | 未登录状态访问需鉴权页面 → 重定向至 `/login`；登录后回跳原路径 |
| SSE 兼容 | 使用 fetch-based SSE 实现，支持自定义 header ✅ |
| 版本协商 | 无需额外 header；accessToken 过期由 401 + silent refresh 机制透明处理 |

---

## 验证

- [ ] `POST /auth/register` — 新用户注册成功，返回 token 对 + user 信息
- [ ] `POST /auth/register` — 重复手机号返回 409 `AUTH_PHONE_EXISTS`
- [ ] `POST /auth/login` — 正确凭据返回 200 + token 对
- [ ] `POST /auth/login` — 错误密码返回 401 `AUTH_INVALID_CREDENTIALS`
- [ ] `POST /auth/refresh` — 有效 refreshToken 返回新 token 对
- [ ] `POST /auth/refresh` — 已使用的 refreshToken 返回 401 `AUTH_REFRESH_INVALID`
- [ ] `POST /auth/refresh` — Token theft 检测：已失效 token 重用后，该用户全部 refresh token 被撤销
- [ ] `POST /auth/refresh` — 过期 refreshToken 返回 401 `AUTH_REFRESH_EXPIRED`
- [ ] `POST /auth/logout` — 成功后 refreshToken 不可再次使用
- [ ] 携带有效 accessToken 的请求正常通过鉴权中间件
- [ ] accessToken 过期后，前端自动 silent refresh 并重试原请求
- [ ] silent refresh 失败时，前端清除状态并跳转 `/login`
- [ ] SSE 流式请求正确携带 Bearer token
- [ ] Rate limiting：同一 IP 对 login/register 超过 5 次/分钟后返回 429
- [ ] 密码存储为 bcrypt hash，无明文存储
