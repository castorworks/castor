# Castor

[English](README.md) | **简体中文** | [日本語](README.ja.md) | [한국어](README.ko.md)

**面向 vibe coding 的全栈脚手架。**

Castor 基于 Go + Next.js，为你和 AI 编码代理提供可直接扩展的应用基础。用自然语言描述业务需求，在已有的认证、权限、文件管理、国际化和部署能力上构建功能，从想法走向可运行的全栈应用。

## 为什么适合 vibe coding

- **让 AI 有章可循**：根目录及前后端的 `AGENTS.md` 明确架构、代码模式、前后端契约和新增模块清单，供你和 AI 编码代理共同遵循。
- **从完整功能开始扩展**：内置公开站点与管理后台，提供用户、角色、菜单、资产、通知、字典和审计等模块，作为新业务的实现参考。
- **把跨层约束交给检查**：统一验证入口覆盖格式、静态检查、测试和前端构建；护栏测试检查分层依赖、RBAC 资源登记、字典契约与四语言翻译等约束。
- **缩短从开发到部署的路径**：VS Code F5 启动本地环境与前后端调试器，配套 Compose / Kubernetes 部署和备份恢复脚本。

## 用 AI 开始构建

1. 按下方快速开始启动项目，先体验内置功能。
2. 让 AI 编码代理阅读 [AGENTS.md](AGENTS.md) 及相关子目录指引，再描述你的业务需求、字段、权限与验收条件。
3. 参照已有模块完成数据库、API、权限、菜单、前端和四语言文案，并运行 `scripts/check.py` 验证改动。

例如，可以从这样的需求开始：

```text
先阅读 AGENTS.md 和前后端开发指引，参照现有分页 CRUD 模块，
新增项目管理功能，包含名称、描述和状态，支持列表、筛选、创建、编辑和删除。
同时完成数据库迁移、API、RBAC 资源、菜单、状态字典、前端页面、
中英日韩翻译和测试，运行 scripts/check.py 验证。
```

## 项目结构

```
apps/
├── api/                     # Go API
└── web/                     # Next.js Web

deploy/
├── config/                  # 共用 API 配置模板
├── compose/                 # 单机部署 / 多实例共用基础设施 / 本地开发基础设施
└── k8s/                     # Kubernetes Kustomize base + overlays

scripts/
├── check.py                 # 统一验证入口（本地 / AI 代理）
├── dev.py                   # 本地开发环境准备（F5 调用）
├── build-images.py          # 构建、推送、导出镜像
├── deploy-compose.py        # 本地和远程 Compose 发布
├── compose-instance.py      # 共用基础设施上的实例：生成配置、开通库与 bucket、单实例备份恢复
├── deploy-k8s.py            # Kubernetes 初始化与发布
└── backup.py                # Compose 自托管数据备份 / 恢复
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
| 对象存储 | S3 (RustFS 兼容) |
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

Windows 安装 Python 时需包含 Python 启动器（`py`）。VS Code 任务使用 `py -3`，避免命中旧版 Windows Store `python3` 别名。可用 `py -3 --version` 检查实际版本（要求 3.11+）；下文手动命令中的 `python3` 在 Windows 上替换为 `py -3`。

首次启动会自动创建 PostgreSQL、Redis、RustFS、存储 bucket 和开发配置，安装依赖并初始化数据库，再启动 Go / Next.js 调试器并打开浏览器。后续启动复用配置、密钥和数据，数据库初始化可重复执行。Windows、macOS、Linux 上开发流程一致。Docker Desktop 未运行时会自动启动；Linux 上使用 Docker Engine 时需自行启动服务。

- 前端默认 `http://127.0.0.1:3000`，API 默认 `http://127.0.0.1:1234`；端口占用时自动选择可用端口，以准备任务输出为准。
- 默认管理员为 `system`，初始密码保存在 `.local/dev/settings.json` 的 `admin_password` 字段；已有账号的密码不会被重置。
- `.local/dev/` 已忽略提交，包含开发密钥、API 配置和调试环境变量。脚本会更新连接地址等托管字段，其他 API 配置可在 `api.config.toml` 中调整。
- **Shift+F5** 停止前后端调试。基础设施保留运行以便再次启动；通过“任务：运行任务 → Castor: Stop services”或 `python3 scripts/dev.py stop` 停止容器，数据卷保留。
- 仅准备环境可运行 `python3 scripts/dev.py prepare`。启动脚本支持中、英、日、韩文，使用 `CASTOR_DEV_LANG` 指定语言。

每个仓库路径使用独立的 Compose 项目和数据卷，基础设施端口仅绑定本机并自动分配。Go 调试使用 `CASTOR_CONFIG_FILE` 指向生成的配置文件。

### 不使用 VS Code

需要 Docker、Go 1.26+、Node.js 22+、Bun 和 Python 3.11+。

```bash
python3 scripts/dev.py prepare     # 基础设施、配置、依赖、init-db
python3 scripts/dev.py run api   # 终端 1：Go API
python3 scripts/dev.py run web   # 终端 2：Next.js
```

`dev.py run` 使用 `prepare` 选定的端口与连接配置（来自 `.local/dev/debug.env`）启动进程，不依赖 shell，Windows、macOS、Linux 上命令相同。

## 开发规范

开发指引集中在各级 AGENTS.md，人类贡献者与 AI 编码代理（Claude Code、Codex、Cursor 等）共用同一份规范：

- [AGENTS.md](AGENTS.md)：工作原则、端到端新增模块清单、安全不变量、前后端契约、提交规范
- [apps/api/AGENTS.md](apps/api/AGENTS.md)：后端分层、RBAC/菜单、迁移、测试约定
- [apps/web/AGENTS.md](apps/web/AGENTS.md)：前端数据层、页面、权限、i18n、测试约定
- [deploy/README.zh-CN.md](deploy/README.zh-CN.md)：镜像构建、Compose / Kubernetes 部署、备份恢复
- [SECURITY.zh-CN.md](SECURITY.zh-CN.md)：安全策略

提交前运行统一验证：

```bash
python3 scripts/check.py          # 或 python3 scripts/check.py api | web | scripts
python3 scripts/check.py vuln     # 依赖漏洞扫描，发布前或定期执行
```

首次克隆后在项目根目录执行一次 `bun install` 启用 Git hooks：pre-commit 格式化暂存文件，pre-push 在前端有改动时构建前端，并测试改动的 Go 包。

## 核心功能

- 多方式认证（密码 / 邮箱 / 手机号）
- RBAC3 权限管理（多角色继承、SSD/DSD、会话角色激活）
- 文件/资产管理（S3 兼容存储）
- 审计日志
- 在线会话与强制下线，以及密码策略（长度、复杂度、有效期）
- 部门树与角色数据范围（全部 / 本部门及以下 / 本部门 / 指定部门 / 仅本人）
- 动态系统配置
- 数据字典
- 通知系统
- 国际化（中/英/日/韩）

## 许可证

[MIT](LICENSE)。前端基于 [next-shadcn-dashboard-starter](https://github.com/Kiranism/next-shadcn-dashboard-starter)（MIT）改造，第三方声明见 [NOTICE](NOTICE)。
