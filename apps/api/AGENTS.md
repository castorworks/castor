# AGENTS.md — Castor API

Go REST API（Gin + GORM + PostgreSQL + Redis + S3），DDD 分层 + Wire 编译期注入 + RBAC3。
仓库级原则、验证命令、跨栈清单与安全不变量见 [../../AGENTS.md](../../AGENTS.md)，本文件只写后端细节。

## 1. 命令

```bash
go run ./cmd/castor init-db        # 迁移 + 默认资源/角色/菜单/管理员（带 advisory lock，可重复执行）
go run ./cmd/castor                # 启动 API（不做迁移）
go test -race -count=1 ./...       # 测试
go vet ./...
gofmt -w . && goimports -w .       # 格式化
cd cmd/castor && wire              # 修改 Provider 后重新生成 wire_gen.go
../../scripts/check.sh api         # 完整后端验证（与 CI 一致）
```

配置文件路径：`CASTOR_CONFIG_FILE` 环境变量，默认 `configs/app/config.toml`（已忽略，从 `configs/app/example.config.toml` 复制）。

## 2. 目录与分层

依赖方向：`interfaces → application → domain ← infrastructure`，由 `internal/architecture_test.go` 强制校验：domain 不得引入 gorm/gin/redis 与上层包；application 不得引入 infrastructure、interfaces、gorm、gin、gin-jwt；persistence 不得引入 application。

```
cmd/castor/                  main.go（serve / init-db）、wire.go、wire_gen.go（生成）、providers.go（配置 → 应用层策略）
configs/app/                 example.config.toml
configs/i18n/                {zh,en,ja,ko}.toml — 后端消息与种子数据翻译
internal/
├── domain/{module}/         实体 + Repository 接口 + 默认种子（var Default…），只依赖标准库与 domain/shared、pkg/query
│   └── shared/              BaseModel、ErrNotFound
├── application/
│   ├── apperror/            业务哨兵错误（唯一定义处）
│   ├── dto/                 请求/响应 DTO（binding tag、FromEntity）
│   └── service/             服务接口 + 实现；接口在同文件导出
├── infrastructure/
│   ├── persistence/         Repository 实现、PaginatedQuery、translateError
│   │   └── models/          GORM Model（tag、TableName、钩子、ToEntity/FromEntity）
│   ├── database/            连接、migration.go（版本化迁移）
│   ├── bootstrap/           init-db 编排、menu.go（默认菜单）
│   ├── authorization/       默认资源/角色对账
│   ├── config/              配置结构、默认值、环境变量覆盖、校验
│   ├── cache/ storage/      Redis、S3
│   ├── external/            短信、邮件、图形验证码、验证码存储
│   └── i18n/                通知模板渲染
├── interfaces/api/
│   ├── handler/             Gin handler；common.go 提供分页/筛选/CRUD 泛型助手
│   ├── middleware/          auth(JWT/CSRF)、admin(RBAC)、ratelimit、bodylimit、i18n、log、requestid
│   ├── response/            响应助手、错误码、i18n key、错误映射
│   ├── server/              Gin 引擎、安全头、生命周期
│   └── router.go            全部路由（AdminHandlers / UserHandlers）
└── pkg/                     query（筛选/排序）、ratelimit 等通用包
migrations/README.md         迁移规范
```

现有模块：`user`、`asset`、`audit_log`、`login_history`、`permission`（资源/角色/约束/授权会话）、`menu`、`notification`、`dictionary`、`setting`。

**参考模板**：分页 + 筛选列表抄 `audit_log`（handler/service/repository 各一个文件，结构最简单）；带创建/更新/删除抄 `user` 或 `notification`。
**不要照抄**：`dictionary`、`setting` 列表不分页。

## 3. 各层写法

### Domain

```go
// internal/domain/audit_log/repository.go
type Repository interface {
    Gets(ctx context.Context, page, size int, order string, opts ...query.Option) ([]AuditLog, int64, error)
    Get(ctx context.Context, id uint) (*AuditLog, error)
    Create(ctx context.Context, item *AuditLog) error
}
```

- 实体只有 `json` tag；敏感字段用 `json:"-"`（同时自动排除出筛选/排序白名单）。
- 审计字段 `ID/CreatedAt/CreatedBy/UpdatedAt/UpdatedBy` 可内联或嵌入 `shared.BaseModel`。
- “未找到”统一为 `shared.ErrNotFound`；domain 不得 import gorm、gin、redis。

### Persistence

- Model 放 `persistence/models/models.go`（大模块可单独文件），提供 `TableName()`、`ToEntity()`、`XxxModelFromEntity()`；需要审计人时实现 `BeforeCreate/BeforeUpdate` 调 `ExtractUserIDFromContext(tx)`。
- 列表用 `PaginatedQuery[models.XxxModel](ctx, r.db, page, size, order, opts...)`，它会再次按模型列白名单校验 `order`。
- 单条查询包 `translateError(...)`，把 `gorm.ErrRecordNotFound` 转为 `shared.ErrNotFound`。
- 永远 `r.db.WithContext(ctx)`；不要把客户端字符串拼进 `Where`/`Order`/`Raw`。

### Migration

`internal/infrastructure/database/migration.go`：

```go
const CurrentSchemaVersion uint = 2
var migrations = []migration{
    {Version: 1, Name: "initial_schema", Up: AutoMigrate},
    {Version: 2, Name: "create_widgets", Up: func(db *gorm.DB) error {
        return db.Migrator().CreateTable(&models.WidgetModel{})
    }},
}
```

- 只追加新版本并递增 `CurrentSchemaVersion`；**不修改版本 1**，不在启动时 AutoMigrate。
- 迁移只在 `init-db` 中执行，须兼容正在运行的旧镜像；破坏性变更单独评审。详见 `migrations/README.md`。

### Application

- DTO：`XxxPostReq`（`binding:"required,..."`）、`XxxPutReq`（指针字段表示可选）、`XxxResp` + `FromEntity(e) error`。
- Service：导出接口 `XxxService` + 私有实现 + `NewXxxService(依赖...) XxxService`。
- 错误：在 `apperror/errors.go` 定义 `ErrXxx = errors.New("...")`；判断未找到用 `errors.Is(err, shared.ErrNotFound)`，不得 import gorm。
- 涉及账号安全状态（密码、启用、锁定、过期）变化时调用 `RBACService.RevokeUserSessions`。
- **不读全局配置**：需要配置时在 `service/policy.go` 定义策略结构体（如 `RuntimePolicy`、`AssetPolicy`、`RsaKeyConfig`）作为构造参数，在 `cmd/castor/providers.go` 从 `config.C` 构造并加入 `wire.Build`；测试直接传入策略值，不修改全局状态。
- **不依赖 HTTP**：服务方法接收 `context.Context` 与普通参数；请求解析、客户端 IP/UA 提取、响应写出留在 handler/middleware（参考 `LoginService.Login(ctx, method, req, LoginClient)`、`UserAvatarService.Download(ctx, key, io.Writer)`）。日志用 `log.ErrCtx/WarnCtx(ctx)`（gin.Context 传入时自动带请求 ID，引擎已开启 `ContextWithFallback`）。
- 异步 goroutine 不得持有请求 context，使用 `context.Background()` + 超时。

### Handler 与路由

```go
func (h *AdminAuditLogHandler) Gets(c *gin.Context) {
    GenericGets(c, reflect.TypeOf(audit_log.AuditLog{}), func(ctx *gin.Context, page, size int, order string, opts ...query.Option) (interface{}, int64, error) {
        return h.AuditLogService.Gets(ctx, page, size, order, opts...)
    })
}
```

- `handler/common.go`：`GenericGets / GenericGet / GenericPost / GenericPut / GenericPatch / GenericDelete / GenericBatchDelete`、`ParsePageParams`、`ValidateOrderParam`、`parseOrderParam`（非泛型列表的排序校验）。
- 响应只用 `response.Success / SuccessListPaged / SuccessI18n / BadRequestErr / BadRequestI18n / HandleError`；错误统一 `response.HandleError(c, err)`。
- 路由：在 `router.go` 的 `AdminHandlers`（或 `UserHandlers`）加字段，在 `With()` 中 `adminGroup.Group("/widgets")` 下注册。路径必须是字面量（`router_rbac_test.go` 解析源码）。
- 公开路由必须挂 `middleware.RemoteIpPathRateLimit(r.rateLimiter, 时长, 次数)`。
- 关键写操作记录审计：`AuditLogService.LogAsync(&audit_log.AuditLog{...})`，参考 `handler/account.go` 的 `logAudit`；新类型加到 `domain/audit_log/entity.go` 并在 `domain/dictionary/entity.go` 的 `audit_log_type` 字典中补项。`Details` 不得包含密码、验证码、原始错误。

### 错误码链路（缺一环测试即失败）

1. `application/apperror/errors.go`：`ErrWidgetNotFound`
2. `interfaces/api/response/messages.go`：i18n key 常量 + `msgToCode` 条目
3. `interfaces/api/response/errcode.go`：`CodeWidgetNotFound = ErrCode{Code, HTTPStatus, MsgKey}`（数字段见 `response/errors.go`）
4. `interfaces/api/response/handle_error.go`：`errorMappings` 加 `{apperror.ErrWidgetNotFound, CodeWidgetNotFound}`
5. `configs/i18n/{zh,en,ja,ko}.toml`：翻译

### Wire

`cmd/castor/wire.go` 的 `wire.Build` 中按注释分组添加 `persistence.NewXxxRepository`、`service.NewXxxService`、`handler.NewAdminXxxHandler`；`AdminHandlers/UserHandlers` 通过 `wire.Struct(..., "*")` 自动注入。然后 `cd cmd/castor && wire`。

## 4. RBAC 与菜单

- **授权**：`middleware/admin.go` 调 `RBACService.AuthorizeSession(ctx, sessionID, userID, c.FullPath(), method)`，资源路径必须与 Gin 路由模板完全一致（如 `/api/v1/admin/widgets/:id`）。
- **资源登记**：每个受保护路由在 `internal/domain/permission/entity.go` 的 `DefaultResources` 中声明：

  ```go
  {Code: "admin:widgets:list", Name: "seedResources.admin.widgets.list", Path: "/api/v1/admin/widgets",
   Actions: StringSlice{ActionGET}, Category: CategoryAdmin, Module: "widgets", SortOrder: 300, IsSystem: true, IsEnabled: true},
  ```

  `Name` 的翻译同时加到 `configs/i18n/*.toml` 与 `apps/web/messages/*.json`（`seedResources.*`），前端另需 `resources.modules.widgets`。
- **默认角色**（`infrastructure/authorization/defaults.go`，init-db 对账）：`admin` 拥有全部系统资源；`auditor` 拥有管理类 GET；`user` 拥有用户类资源。
- **菜单**（`infrastructure/bootstrap/menu.go`，仅在菜单表为空时播种）：
  - `add(menu.Menu{Code: "page_widgets", Kind: menu.Page, Titles: menu.Titles{En, Zh, Ja, Ko}, Path: "/dashboard/widgets", Icon: <Icons 的 key>, AccessMode: "permission", Permissions: []menu.Permission{{ResourceID: byPath["/api/v1/admin/widgets"].ID, Action: "GET"}}}, "<父目录 code>")`
  - 在 `modules` 映射中加 `"widgets": "widgets"`，按钮级 action 节点标题来自 `defaultActionTitles`。
  - 已有数据库不会自动出现新菜单，需在「菜单管理」添加或写迁移。

## 5. 配置

- 结构与默认值：`infrastructure/config/config.go`（`default:"..."` tag，由 creasty/defaults 填充）。
- 加载顺序：`Load → OverrideFromEnv → Validate`（`config/load.go`）。
- 环境变量覆盖是**固定列表**：`CASTOR_JWT_KEY`、`CASTOR_RSA_SECRET`、`CASTOR_DB_{HOST,PORT,USER,PASSWORD,NAME}`、`CASTOR_REDIS_{ADDR,PASSWORD}`、`CASTOR_S3_{ENDPOINT,ACCESS_KEY,SECRET_KEY,BUCKET}`、`CASTOR_MAIL_*`、`CASTOR_SMS_*`、`CASTOR_CORS_ORIGINS`；另有 `CASTOR_CONFIG_FILE`、`CASTOR_DEFAULT_ADMIN_PASSWORD`（≥12 位，非开发模式 init-db 必填）。
- 始终校验：JwtKey ≥32 字节、`JwtTimeoutHours ≤ JwtMaxRefreshHours`、必填连接信息；非开发模式另外要求 RSA 密钥恰为 32 字节、拒绝占位符密钥、CORS 不允许 `*`。
- 新增配置项：同时更新 `config.go`、`configs/app/example.config.toml`、`deploy/config/api.config.example.toml`，必要时更新 `scripts/dev.py` 生成的开发配置。

## 6. 测试约定

- 标准库 `testing`，table-driven + `t.Run`；**手写 mock**（`service/test_helpers_test.go`、`handler/handler_test.go`），不引入 mock 框架。
- Redis 用 `miniredis`；属性测试用 `testing/quick`。
- `gin.SetMode` 只在 `TestMain` 中调用（见 `service/main_test.go`、`middleware/main_test.go`）；修改 `config.C`、调用 `t.Setenv` 或 `gin.SetMode` 的测试**不得** `t.Parallel()`。application 层测试通过策略结构体注入配置，无需触碰 `config.C`。
- 真实 PostgreSQL 测试读取 `CASTOR_TEST_POSTGRES_DSN`，未设置时 `t.Skip`。
- 修复缺陷时先写复现测试；安全相关修复必须带回归测试（参考 `handler/order_security_test.go`、`middleware/auth_refresh_test.go`）。

## 7. 代码风格

- 包名小写无下划线（现有 `audit_log`、`login_history` 为历史例外，不新增）。
- 错误包装 `fmt.Errorf("context: %w", err)`；忽略错误必须有理由，清理类失败至少记日志。
- import 分组：标准库 / 第三方 / 本项目。
- 日志用 `github.com/hyperits/gosuite/logger`，禁止记录密码、token、验证码。
- 注释与现有代码一致使用中文，导出符号写 godoc。
