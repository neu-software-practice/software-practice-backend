# API 漂移检测脚本

检测前端 Zod Schema（权威真相源）与后端 Go 实现之间的 API 契约漂移。

## 快速开始

```bash
# 1. 提取前端 Zod Schema（需要前端仓库可访问）
(cd ../neuhis-agent-front && node scripts/extract-zod-fields.mjs)

# 2. 提取后端 Go Struct
node scripts/extract-go-fields.mjs

# 3. 字段级对比
node scripts/compare-fields.mjs

# 4. 查看漂移报告
cat drift-report-fields.json | python3 -m json.tool

# 5. 一键编排（提取 → 对比 → 报告）
bash scripts/fix-drift-loop.sh
```

## 脚本说明

### 端点级检测（快速扫描）

| 脚本 | 输入 | 输出 | 用途 |
|------|------|------|------|
| `extract-frontend-api.mjs` | `../neuhis-agent-front/src/features/*/api/` | `api-contract.json` | 提取前端 API 端点列表（方法+路径） |
| `extract-backend-api.mjs` | `internal/handler/router.go` | `backend-api.json` | 提取后端路由表（方法+路径+handler） |
| `compare-api.mjs` | `api-contract.json` + `backend-api.json` | `drift-report.json` | 端点级对比：缺失端点、HTTP 方法错误 |

### 字段级检测（深度扫描） ⭐ 推荐

| 脚本 | 输入 | 输出 | 用途 |
|------|------|------|------|
| `extract-frontend-fields.mjs` | `../neuhis-agent-front/src/` 所有 schema 文件 | `frontend-fields.json` | 提取 Zod schema 字段级信息（required/optional/type/constraints） |
| `extract-go-fields.mjs` | `internal/model/`, `internal/handler/` | `backend-fields.json` | 提取 Go struct 字段级信息（pointer/omitempty/json tag） |
| `compare-fields.mjs` | `frontend-fields.json` + `backend-fields.json` | `drift-report-fields.json` | **字段级漂移检测** |

### 编排脚本

| 脚本 | 用途 |
|------|------|
| `fix-drift-loop.sh` | 循环编排：提取→对比→修复→验证→提交→重新扫描，直到零漂移 |

## 漂移检测算法

`compare-fields.mjs` 对每个非 SSE 端点的每个字段执行：

```
Zod required + Go *T + omitempty       → 🔴 HIGH   (nil 时字段静默消失)
Zod required + Go T + omitempty        → 🔴 HIGH   (零值时字段静默消失)
Zod required + Go *T (无 omitempty)    → ✅ OK     (null 可见，可区分)
Zod required + Go T (无 omitempty)     → ✅ OK
Zod optional + Go 无指针/omitempty     → 🟡 MEDIUM (无法区分"未设置"和零值)
Zod optional + Go *T + omitempty       → ✅ OK
```

## 依赖

- Node.js ≥ 22（使用 `fs.globSync`）
- 前端仓库位于 `../neuhis-agent-front/`（可修改为任意路径）
- 所有脚本为纯 JavaScript (`.mjs`)，无需额外依赖

## 自定义路径

通过环境变量或命令行参数指定前端路径：

```bash
# 方式 1: 环境变量
export FRONTEND_DIR=/path/to/neuhis-agent-front

# 方式 2: 脚本参数（部分脚本支持）
node scripts/compare-fields.mjs /custom/path/frontend-fields.json /custom/path/backend-fields.json
```

## 输出示例

### 端点级报告 (`drift-report.json`)
```json
{
  "totalDriftItems": 0,
  "bySeverity": { "CRITICAL": 0, "HIGH": 0, "MEDIUM": 0, "LOW": 0 },
  "items": []
}
```

### 字段级报告 (`drift-report-fields.json`)
```json
{
  "endpointsCompared": 45,
  "fieldsCompared": 345,
  "totalDriftItems": 0,
  "bySeverity": { "HIGH": 0, "MEDIUM": 0 },
  "items": [
    {
      "severity": "INFO",
      "category": "sse_endpoints",
      "description": "SSE event types verified"
    }
  ]
}
```
