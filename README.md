# Castor

Castor 是一个全栈管理后台系统，提供用户管理、权限控制、文件管理、审计日志等开箱即用的功能模块。

## 项目结构

```
apps/
├── api/                     # Go API
└── web/                     # Next.js Web

deploy/
├── config/                  # 共用 API 配置模板
├── compose/                 # 单机部署 / 本地开发基础设施
└── k8s/                     # Kubernetes Kustomize base + overlays

scripts/
├── check.sh                 # 统一验证入口（本地 / CI / AI 代理）
├── dev.py                   # 本地开发环境准备（F5 调用）
├── build-images.sh          # 构建、推送、导出镜像
├── deploy-compose.sh        # 本地和远程 Compose 发布
├── deploy-k8s.sh             # Kubernetes 初始化与发布
└── backup.sh                # Compose 自托管数据备份 / 恢复
```

## 技术栈

### 后端

| 组件 | 技术选型 |
|------|----------|
| 语言 | Go 1.26.0+ |
| Web 框架 | Gin |
| ORM | GORM |
| 数据库 | PostgreSQL |
| 缓存 | Redis |
| 对象存储 | S3 (MinIO 兼容) |
| 权限控制 | RBAC3（角色层级、SSD、DSD、授权会话） |
| 认证 | JWT |
| 依赖注入 | Wire |

### 前端

| 组件 | 技术选型 |
|------|----------|
| 框架 | Next.js 16 (App Router) |
| UI | React 19 + shadcn/ui + Tailwind CSS 4 |
| 状态管理 | Zustand |
| 数据请求 | TanStack Query |
| 表单 | TanStack Form |
| 表格 | TanStack Table |

## 快速开始

### VS Code 一键调试（F5）

用 VS Code 打开仓库根目录，安装推荐的 Go 扩展，按 **F5**（调试配置选择 `Castor`）。需要预先安装 Docker Desktop、Go、Node.js 22+、Bun、Python 3.11+ 和 Chrome；数据库等服务无需手动安装。

首次启动会自动创建 PostgreSQL、Redis、MinIO、存储 bucket 和开发配置，安装依赖并初始化数据库，再启动 Go / Next.js 调试器并打开浏览器。后续启动复用配置、密钥和数据，数据库初始化可重复执行。macOS 上 Docker Desktop 未运行时会自动启动。

- 前端默认 `http://127.0.0.1:3000`，API 默认 `http://127.0.0.1:1234`；端口占用时自动选择可用端口，以准备任务输出为准。
- 默认管理员为 `system`，初始密码保存在 `.local/dev/settings.json` 的 `admin_password` 字段；已有账号的密码不会被重置。
- `.local/dev/` 已忽略提交，包含开发密钥、API 配置和调试环境变量。脚本会更新连接地址等托管字段，其他 API 配置可在 `api.config.toml` 中调整。
- **Shift+F5** 停止前后端调试。基础设施保留运行以便再次启动；通过“任务：运行任务 → Castor: Stop services”或 `python3 scripts/dev.py stop` 停止容器，数据卷保留。
- 仅准备环境可运行 `python3 scripts/dev.py prepare`。启动脚本支持中、英、日、韩文，使用 `CASTOR_DEV_LANG` 指定语言。

每个仓库路径使用独立的 Compose 项目和数据卷，基础设施端口仅绑定本机并自动分配。Go 调试使用 `CASTOR_CONFIG_FILE` 指向生成的配置文件。

### 环境要求

- Go 1.26.0+
- Node.js 22+ / Bun
- PostgreSQL
- Redis

### 启动基础设施

```bash
cp deploy/compose/.env.example deploy/compose/.env  # 填写密钥
scripts/deploy-compose.sh up dev
```

### 启动后端

```bash
cd apps/api
cp configs/app/example.config.toml configs/app/config.toml  # 修改配置
go mod tidy
# 数据库地址、数据库名、凭据和 bucket 与部署配置保持一致
go run ./cmd/castor init-db
go run ./cmd/castor
```

`init-db` 创建默认管理员账号 `system`。初始密码通过 `CASTOR_DEFAULT_ADMIN_PASSWORD` 指定（≥12 位，非开发模式必填）；开发模式下未指定时随机生成，仅由 `init-db` 在 stderr 输出一次。普通 API 启动不执行数据库迁移。

### 启动前端

```bash
cd apps/web
bun install
bun run dev
```

## 开发规范

开发指引集中在 [AGENTS.md](AGENTS.md)，人类贡献者与 AI 编码代理（Claude Code、Codex、Cursor 等）共用同一份规范：

- [AGENTS.md](AGENTS.md)：工作原则、端到端新增模块清单、安全不变量、前后端契约、提交规范
- [apps/api/AGENTS.md](apps/api/AGENTS.md)：后端分层、RBAC/菜单、迁移、测试约定
- [apps/web/AGENTS.md](apps/web/AGENTS.md)：前端数据层、页面、权限、i18n、测试约定

提交前运行统一验证（CI 执行同一脚本）：

```bash
scripts/check.sh          # 或 scripts/check.sh api | web | scripts
```

首次克隆后在项目根目录执行一次 `bun install` 启用 Git hooks：pre-commit 格式化暂存文件，pre-push 构建前端并测试改动的 Go 包。

## 核心功能

- 多方式认证（密码 / 邮箱 / 手机号）
- RBAC3 权限管理（多角色继承、SSD/DSD、会话角色激活）
- 文件/资产管理（S3 兼容存储）
- 审计日志
- 动态系统配置
- 数据字典
- 通知系统
- 国际化（中/英/日/韩）

## 常用命令

```bash
# 后端
cd apps/api && go test ./...          # 运行测试
cd apps/api && go vet ./...           # 静态检查
cd apps/api/cmd/castor && wire        # 重新生成依赖注入

# 前端
cd apps/web && bun run build          # 构建
cd apps/web && bun run lint           # Lint
cd apps/web && bun run test           # 测试

# 发布（准备配置后执行，详见部署指南）
scripts/build-images.sh load castor local
scripts/deploy-compose.sh up full
scripts/deploy-k8s.sh apply production your-kube-context

# Compose 自托管备份 / 恢复
scripts/backup.sh create full
scripts/backup.sh list
RESTORE_CONFIRM=snapshot-20260909_120000 scripts/backup.sh restore full snapshot-20260909_120000
```

## 文档

- [部署指南](deploy/README.md)
- [后端开发参考](apps/api/AGENTS.md)
- [前端开发参考](apps/web/AGENTS.md)
- [安全策略](SECURITY.md)
- [数据库迁移](apps/api/migrations/README.md)

## 许可证

[MIT](LICENSE)。前端基于 [next-shadcn-dashboard-starter](https://github.com/Kiranism/next-shadcn-dashboard-starter)（MIT）改造，第三方声明见 [NOTICE](NOTICE)。
