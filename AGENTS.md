# AGENTS.md

本文件是 Castor 仓库面向 AI 编码代理（Claude Code、Codex、Cursor 等）与人类贡献者的**权威开发指引**。子目录的 AGENTS.md 提供该应用的详细参考：

- [apps/api/AGENTS.md](apps/api/AGENTS.md) — Go 后端：分层、模式、新增模块、RBAC/菜单、迁移、测试
- [apps/web/AGENTS.md](apps/web/AGENTS.md) — Next.js 前端：数据层、页面、表格/表单、权限、i18n、测试
- [deploy/README.md](deploy/README.md) — 镜像构建、Compose / Kubernetes 部署、备份恢复

文档与代码冲突时**以代码和测试为准**，并在同一改动中修正文档。

## 1. 仓库地图

| 路径 | 内容 |
|------|------|
| `apps/api/` | Go 1.26 后端：Gin、GORM、PostgreSQL、Redis、S3、Wire、RBAC3 |
| `apps/web/` | Next.js 16 前端：React 19、shadcn/ui、TanStack Query/Form/Table、next-intl、Bun |
| `deploy/` | Compose、Kustomize、共用 API 配置模板 |
| `scripts/` | `check.sh`（统一验证）、`dev.py`（本地环境）、镜像构建/部署/备份脚本及其测试 |
| `.github/workflows/ci.yml` | CI：调用 `scripts/check.sh` + govulncheck |
| `.vscode/` | F5 一键调试配置（调用 `scripts/dev.py`） |
| `.claude/settings.json` | Claude Code 共享权限（验证类命令免确认）；个人设置放 `settings.local.json`（已忽略） |

业务模块：用户、认证（密码/邮箱/短信）、登录历史、RBAC3 权限（角色/资源/授权会话）、菜单、资产、通知、字典、系统设置、审计日志、仪表盘。

## 2. 工作原则

1. **直达目标设计**：项目处于开发阶段，不保留兼容层、废弃字段或半成品折中。
2. **在 `main` 上开发**，不开分支；只在用户要求时提交。
3. **端到端完整**：一个功能涉及的所有层（迁移、领域、服务、接口、RBAC 资源、菜单、前端、i18n、测试、文档）在同一改动中落地。
4. **四语言 i18n**：所有用户可见文案进入翻译文件，`zh / en / ja / ko` 同步，禁止硬编码。
5. **先读后写**：修改前先阅读同类现有实现并沿用其模式；参考模块见子目录 AGENTS.md。
6. **不手改生成物**：`apps/api/cmd/castor/wire_gen.go`、`apps/web/.next/`、lockfile 只能通过工具生成。
7. **测试随改动**：修 bug 先写能复现的测试；新增逻辑补对应单测。
8. **不削弱安全不变量**（见第 6 节），不为让测试通过而放宽校验。

## 3. 验证：完成的定义

```bash
scripts/check.sh            # 全部（api + web + scripts）
scripts/check.sh api        # gofmt、go vet、go test -race
scripts/check.sh web        # oxfmt、oxlint、tsc、vitest、next build
scripts/check.sh scripts    # 部署/开发脚本单测
```

改动完成前必须让受影响目标通过；CI 执行同一脚本。以下测试是仓库的“护栏”，失败说明遗漏了某一层，**修复遗漏而不是修改测试**：

| 测试 | 防止 |
|------|------|
| `apps/api/internal/architecture_test.go` | DDD 分层依赖方向被破坏（domain/application 引入 gorm、gin 或下层包） |
| `apps/api/internal/interfaces/api/router_rbac_test.go` | 受保护路由未登记为 RBAC 资源，或资源已失效 |
| `apps/api/internal/interfaces/api/response/i18n_parity_test.go` | 后端四语言 key 不一致、`messages.go` 常量缺翻译 |
| `apps/api/internal/interfaces/api/response/i18n_keys_test.go` | 业务消息缺 HTTP 状态映射 |
| `apps/api/internal/interfaces/api/handler/order_security_test.go` | 排序参数注入 |
| `apps/web/src/i18n/messages.test.ts` | 前端四语言 key 不一致 |
| `apps/web/src/features/resources/i18n.test.ts` | 后端资源种子 `seedResources.*` 缺前端翻译 |

数据库集成测试需设置 `CASTOR_TEST_POSTGRES_DSN`（指向独立空库），未设置时自动跳过。

## 4. 本地环境

```bash
python3 scripts/dev.py prepare   # 启动 PostgreSQL/Redis/MinIO，生成配置，init-db
python3 scripts/dev.py stop      # 停止基础设施（保留数据卷）
```

- 或在 VS Code 按 F5（配置 `Castor`），同时启动 Go 与 Next.js 调试器。
- 生成物位于 `.local/dev/`（已忽略）：`settings.json`（端口、`admin_password`）、`api.config.toml`、`debug.env`。
- 默认管理员 `system`；API 默认 `http://127.0.0.1:1234`，Web 默认 `http://127.0.0.1:3000`，端口冲突时自动改选，以 `settings.json` 为准。
- 手动运行：`cd apps/api && CASTOR_CONFIG_FILE=../../.local/dev/api.config.toml go run ./cmd/castor`；`cd apps/web && bun run dev`。

## 5. 端到端新增业务模块

以下为跨栈清单，细节与代码模板见子目录 AGENTS.md。分页 CRUD 参考后端 `audit_log`、前端 `resources`。

**后端（apps/api）**
1. 领域：`internal/domain/{name}/entity.go`、`repository.go`（纯 Go，无 GORM）。
2. 持久化：`persistence/models/` 中的 Model（GORM tag、`ToEntity`/`FromEntity`）+ `persistence/{name}_repository.go`（`PaginatedQuery`、`translateError`）。
3. 迁移：在 `internal/infrastructure/database/migration.go` 追加新版本并递增 `CurrentSchemaVersion`，不改历史版本。
4. 应用层：`application/dto/{name}.go`、`application/service/{name}.go`、`application/apperror/errors.go` 中的哨兵错误。
5. 接口层：`interfaces/api/handler/{name}.go`；在 `router.go` 注册到 `AdminHandlers`/`UserHandlers`；错误码链路 `messages.go → errcode.go → handle_error.go → configs/i18n/*.toml`。
6. RBAC：每个 `/api/v1/admin/*` 路由在 `internal/domain/permission/entity.go` 的 `DefaultResources` 中登记（路径 = Gin 路由模板，方法一致）。
7. 菜单：在 `internal/infrastructure/bootstrap/menu.go` 添加页面节点及其权限（仅新库生效；已有库通过「菜单管理」或迁移添加）。
8. DI：`cmd/castor/wire.go` 加 Provider，执行 `cd apps/api/cmd/castor && wire`。

**前端（apps/web）**
9. 数据层：`src/features/{name}/api/{types,service,queries,mutations}.ts` + `service.test.ts`。
10. 组件：`schemas/{name}.ts`（zod）、`components/{name}-listing.tsx`、`{name}-form-sheet.tsx`、`{name}-table/{index,columns,cell-action}.tsx`。
11. 页面：`src/app/dashboard/{name}/page.tsx`（`PageContainer` + `serverHasPermission`）；新的 URL 筛选参数加到 `src/lib/searchparams.ts`。
12. i18n：`messages/{zh,en,ja,ko}.json` 中的 `nav.{name}`、`dashboard.{name}Description`、模块命名空间、`seedResources.*`、`resources.modules.{module}`；后端 `configs/i18n/*.toml` 中的 `seedResources.*`。

**收尾**
13. `scripts/check.sh` 全部通过；更新受影响的 AGENTS.md。

## 6. 安全不变量

| 规则 | 实现位置 |
|------|----------|
| 客户端 `order` 不得直接进入 `.Order()`；只允许白名单列 + `asc/desc` | `internal/pkg/query/order.go`，handler `ValidateOrderParam`，repo `PaginatedQuery` |
| 过滤/排序字段排除 `json:"-"` 与密码、密钥、token 类列 | `query.IsSensitiveColumn`，`handler/common.go` |
| 所有 `/admin/*` 路由经过 JWT + CSRF + RBAC 中间件，且登记为资源 | `router.go`，`middleware/admin.go`，`router_rbac_test.go` |
| 公开路由必须限流 | `middleware.RemoteIpPathRateLimit` |
| 修改/重置密码、禁用用户时撤销会话 | `RBACService.RevokeUserSessions` |
| Token 刷新作废旧 token；登出加入黑名单 | `middleware/auth.go`，`service/token.go` |
| 验证码限制失败次数、一次性、常量时间比较 | `service/verification_code.go`，`external/verification_code.go` |
| 非开发模式拒绝占位符密钥；`init-db` 必须提供管理员密码 | `infrastructure/config/load.go`，`service/user.go` |
| 敏感信息（密码、验证码、token）不写日志和审计详情 | 全局 |
| 前端登录态只在 httpOnly `jwt` cookie，浏览器端不读 token | `apps/web/src/lib/api-client.ts` |
| `redirect` 参数必须经 `safeRedirectPath` | `apps/web/src/lib/safe-redirect.ts` |
| 新增外部资源来源需同步更新 CSP | `apps/web/src/lib/security-headers.ts` |
| 前端权限判断仅用于显隐，后端 RBAC 为唯一安全边界 | — |

## 7. 前后端契约

- **路径**：API 前缀 `/api/v1`；浏览器经 Next.js rewrite 同源访问，服务端组件通过 `CASTOR_API_URL` 直连。
- **响应**：`{ code, data, message }`，`code === 200` 为成功；`message` 已按 `Accept-Language` 本地化，前端可直接展示。
- **分页**：请求 `page`、`pageSize`（默认 10，最大 500）；响应 `data: { total, list, page, pageSize, totalPages }`。
- **筛选/排序**：`{field}-{op}=value`（op：`like|eq|ne|gt|gte|lt|lte|in`），`searchText` + `searchFields`，`order=field asc|desc`（最多 3 项）；非法值返回 400。
- **认证**：登录前获取 RSA 公钥加密密码；后端设置 httpOnly `jwt` cookie 与 `csrf_token` cookie；写请求带 `X-CSRF-Token` 头；401 时前端单次刷新并重试。
- **权限标识**：`"{resourcePath}:{METHOD}"`，如 `/api/v1/admin/users/:id:PUT`，来自 `/api/v1/account/permissions`。
- **导航**：`/api/v1/account/navigation` 返回菜单树与 `routes[{path, allowed}]`，前端 `src/proxy.ts` 据此守卫 `/dashboard/*`。
- **错误码**：`0` 成功，`1xxx` 通用，`2xxx` 认证授权，`3xxx` 用户，`4xxx` 资产，`5xxx` 配置，`9xxx` 内部错误（定义于 `apps/api/internal/interfaces/api/response/errors.go`）。

## 8. 提交规范

格式：`<type>(<scope>): <描述>`，描述可用中文。type 取值：

`feat` 新特性 · `fix` 修复 · `perf` 性能 · `refactor` 重构 · `style` 无逻辑格式调整 · `test` 测试 · `docs` 文档 · `chore` 依赖/脚手架 · `ci` 持续集成 · `types` 类型定义 · `revert` 回滚

Git hooks（`bun install` 后启用）：pre-commit 格式化暂存文件并检查 gofmt；pre-push 构建前端并对改动的 Go 包执行 vet/test。

## 9. 文档维护

| 改动 | 同步更新 |
|------|----------|
| 新增/删除模块、路由分组、目录结构 | 本文件第 1、5 节，对应子目录 AGENTS.md |
| 新的安全机制或约束 | 本文件第 6 节 |
| 前后端契约变化 | 本文件第 7 节 |
| 配置项、环境变量 | `apps/api/AGENTS.md` 配置节、`configs/app/example.config.toml`、`deploy/config/api.config.example.toml` |
| 部署流程 | `deploy/README.md` |
| 验证步骤 | `scripts/check.sh`（CI 自动跟随） |
