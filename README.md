# 东软云医院 HIS 门诊管理系统 — 后端

门诊全流程信息化系统后端，覆盖 **挂号 → 看病 → 缴费 → 发药** 的闭环。基于
**Go 1.26 + Gin + GORM + MySQL 8**，分层架构（handler → service → repository →
model），JWT 鉴权，按科室类型（`dept_type`）做 RBAC。

> 规格见 [`docs/SPEC.md`](docs/SPEC.md)，实现方案见 [`docs/PLAN.md`](docs/PLAN.md)。
> 前端为独立仓库（`his-frontend`）。

## 技术栈

| 层 | 选型 |
| --- | --- |
| Web | Gin（单体）、统一响应封装 `{success,data,error,meta}` |
| ORM | GORM + MySQL 8 |
| 迁移 | golang-migrate（SQL 版本化，内嵌于二进制） |
| 鉴权 | JWT（HS256）+ bcrypt 口令哈希 |
| 文档 | swaggo / Swagger |
| 测试 | `go test` + testify；集成测试默认 SQLite（纯 Go），CI 另跑真实 MySQL |
| CI | GitHub Actions：lint + 真库迁移/种子 + race 测试 + 覆盖率门控 ≥80% |

## 目录结构

```
cmd/            server / migrate / seed 三个入口
internal/
  config/       环境变量配置加载
  model/        15 业务表 + 2 财务流水表的 GORM 模型
  repository/   数据访问层（接口 + GORM 实现，含泛型请求仓储）
  service/      业务逻辑（状态机、金额计算、事务）
  handler/      Gin handlers
  middleware/   JWT 鉴权 / RBAC / CORS / Recovery / Logger
  router/       路由注册与分组
  pkg/          response / apperr / jwt / hash / constant / database
  seed/         演示数据
migrations/     golang-migrate 的 *.up.sql / *.down.sql
test/           黑盒集成测试（驱动整机 HTTP）
```

## 快速开始

### 方式一：Docker Compose（推荐）

```bash
cp .env.example .env        # 可按需修改
docker compose up -d --build
# 后端自动执行迁移 + 种子，监听 http://localhost:8080
curl http://localhost:8080/api/health
```

### 方式二：本地运行

```bash
# 1) 启动 MySQL 8（示例用 Docker）
docker run -d --name his-mysql -e MYSQL_ROOT_PASSWORD=rootpw \
  -e MYSQL_DATABASE=his -e MYSQL_USER=his -e MYSQL_PASSWORD=hispw \
  -p 3306:3306 mysql:8.0

# 2) 配置环境
cp .env.example .env         # 确认 DATABASE_DSN / JWT_SECRET

# 3) 迁移 + 种子 + 启动
make migrate
make seed
make run                     # http://localhost:8080
```

## 环境变量

见 [`.env.example`](.env.example)。关键项：

| 变量 | 说明 |
| --- | --- |
| `DATABASE_DSN` | GORM MySQL DSN，须含 `parseTime=True` |
| `JWT_SECRET` | JWT 签名密钥（必填，无默认，禁止硬编码） |
| `JWT_TTL_HOURS` | Token 有效期（默认 12） |
| `SEED_DEFAULT_PASSWORD` | 种子账号统一口令（默认 `Passw0rd!`） |

## 演示账号

种子后可用以下账号登录（统一口令 `SEED_DEFAULT_PASSWORD`）：

| 用户名 | 角色 | `dept_type` |
| --- | --- | --- |
| `finance` | 挂号收费员 | 财务 |
| `doctor` | 门诊医生 | 门诊 |
| `checker` | 检查医生 | 检查 |
| `inspector` | 检验医生 | 检验 |
| `pharmacist` | 药房管理员 | 药房 |
| `disposer` | 处置医生 | 处置 |
| `root` | 系统管理员（只读） | root |

## 常用命令

```bash
make help        # 列出全部目标
make run         # 启动服务
make migrate     # 应用迁移   / make migrate-down 回滚
make seed        # 灌入演示数据
make test        # 运行测试
make cover       # 测试 + 覆盖率门控（≥80%）
make lint        # golangci-lint
make fmt         # gofmt
make swag        # 生成 Swagger 文档到 internal/swagger
```

## API 与文档

- 统一前缀 `/api`，鉴权头 `Authorization: Bearer <jwt>`。
- 统一响应：`{ "success": true, "data": {}, "error": null, "meta": {...} }`。
- Swagger：`make swag` 生成后，访问 `/swagger/index.html`。

## 测试策略

- **单元**：service 用 mock 仓储测业务逻辑；pkg/middleware/model 独立单测。
- **集成**：`test/` 驱动整机 HTTP，默认连内存 SQLite（纯 Go、隔离、可并行）。
- **真库校验**：CI 用 MySQL 8 service 跑 `make migrate && make seed`，确保生产
  schema 与种子在真实 MySQL 上可用。
- 覆盖率门控 ≥80%（入口与活库适配器除外，详见 `scripts/coverage.sh`）。

```bash
go test -race ./...
make cover
```
