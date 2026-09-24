# 部署指南

[English](README.md) | **简体中文** | [日本語](README.ja.md) | [한국어](README.ko.md)

Castor 提供 Docker Compose 和 Kubernetes 两种部署方式。两者使用相同的 API / Web 镜像，并通过同一域名分发 `/api/v1/*` 到 API、其他请求到 Web。Dockerfile 位于 `apps/api` 和 `apps/web`。

`scripts/` 下的脚本只依赖 Python 3.9+（标准库）以及它们调用的工具（Docker、kubectl、ssh），在 Windows、macOS、Linux 上行为一致。Windows 上用 `py -3` 代替 `python3` 运行；`cp` / `chmod` 示例适用于 Linux 和 macOS 主机，Windows 上照常复制文件，权限由 NTFS 控制。

## 发布模型

1. 构建带明确版本标签的两个镜像，不覆盖已发布标签。
2. 准备 PostgreSQL、Redis、S3 和 API 配置。
3. 使用当前 API 镜像执行 `init-db`。
4. 初始化成功后，更新 API 和 Web，等待就绪。

`init-db` 在 PostgreSQL 事务中持有 advisory lock，执行表结构迁移，并对系统设置、字典、权限资源、角色、菜单和管理员做只增不改的对账（首次还会生成部署标识）。失败回滚并返回非零退出码。重复运行不会重置管理员密码。普通 API 启动只读取已有数据库和权限配置；不运行迁移。

镜像更新与数据库变更是不同操作。涉及不兼容的数据库变更时，安排维护窗口并备份数据，不能仅依赖滚动更新。旧镜像不保证兼容新数据库。

## 多租户怎么选

Castor 的一套部署服务一个组织。要为多个客户（租户）提供服务，采用"每个租户一套实例"：每个租户独立的数据库与角色、bucket、Redis key 前缀、密钥和域名，可以共用或独占基础设施（Compose 见下文"多套实例共用基础设施"；Kubernetes 为每个租户生成一个实例 overlay）。任意两个租户都不共用数据库，任何查询都不可能跨租户泄露数据；每个租户都能单独备份、恢复、迁移或删除。

| 需求 | 推荐做法 |
|------|------|
| 同一组织内的部门或下属公司只看自己的数据 | 一套实例 + 部门与角色数据范围 |
| 几个到几十个客户，要求强隔离（政企、医疗、金融） | 每个租户一套实例 |
| 数百个自助注册即开通的租户、跨租户计费 | 在下游产品中自行实现行级多租户（不提供） |

每套实例都有固定开销（API 与 Web 进程、数据库连接，见下文 `max_connections` 说明），也要逐套升级；下面的批量命令让这件事可控。行级多租户（共享数据库、每行带 `tenant_id`）刻意不纳入 Castor，否则每个下游项目都要承担它的成本和风险。确实需要的产品至少要规划：

- 每张表加 `tenant_id`，唯一约束改为组合约束（用户名、联系方式、角色/部门/字典编码、配置键），并且任何仓储查询和原生 SQL 都无法绕过租户过滤（GORM scope，再以 PostgreSQL 行级安全兜底）；
- 逐个决定系统表的数据是全局还是按租户（菜单、资源、字典、配置），并相应改写种子对账；
- 资产按租户去重（全局内容哈希会暴露其它租户拥有同一文件）；
- 登录时识别租户（域名或租户码），并把租户带进 JWT、会话、Redis key 和限流；
- "全体"通知、仪表盘和数据范围都限定在租户之内，跨租户的运营人员单独授权。

## 构建镜像

要求 Docker + Buildx。默认构建 `linux/amd64`，ARM 主机本地验证时设置 `PLATFORM=linux/arm64`。API 构建默认使用官方 Go/Ubuntu 源；访问不到这些源的构建机（如国内）在运行脚本时设置环境变量 `GOPROXY`、`UBUNTU_MIRROR` 和 `UBUNTU_SECURITY_MIRROR`，脚本会把它们作为构建参数传入，例如 `GOPROXY=https://goproxy.cn,direct`。

```bash
# 本地加载镜像
python3 scripts/build-images.py load castor local

# 推送镜像仓库（构建机和部署机分别执行 docker login）
python3 scripts/build-images.py push registry.example.com/castor v1.0.0

# 离线交付应用镜像
python3 scripts/build-images.py tar castor v1.0.0 deploy/artifacts/castor-v1.0.0.tar.gz
```

Web 默认使用 Node Dockerfile，确保运行时和健康检查命令一致。生产路由由入口处理，环境域名无需写入前端镜像。`NEXT_PUBLIC_*` 如有定制仍属于构建配置。

## Compose

### 配置

```bash
cp deploy/compose/.env.example deploy/compose/.env
cp deploy/config/api.config.example.toml deploy/compose/config/api.config.toml
chmod 600 deploy/compose/.env
```

填写 `.env` 中所有空密钥：JWT 建议 32 字节以上；RSA 必须为恰好 32 个 ASCII 字节；管理员密码至少 12 字符。用 `python3 -c "import secrets; print(secrets.token_hex(32))"` 生成 JWT 等密钥，把其中的 32 换成 16 生成 RSA 密钥。每项使用独立值。不要提交真实 `.env`。

短信和邮件验证码是可选的：不使用时 `.env` 的 `CASTOR_SMS_*`、TOML 的 `[Mail].Host` 留空即可，服务照常启动，用户选择该方式时会提示未配置。启用短信需同时填写 `CASTOR_SMS_ACCESS_KEY`、`CASTOR_SMS_SECRET_KEY` 与 TOML 的 `SignName`、`TemplateCode`；启用邮件需填写 `[Mail]` 的 Host 与账号，密码经 `CASTOR_MAIL_PASSWORD` 注入。K8s 需要时把同名变量加入 `secrets.env`。

`CASTOR_INSTANCE_ID` 是实例标识（Redis key 前缀与 JWT `aud`），单实例保持默认 `castor` 即可；与其他部署共用 Redis 时见下文「多套实例共用基础设施」。`API_IMAGE` / `WEB_IMAGE` 必须与构建结果一致。`S3_BUCKET` 同时用于 API 配置和自托管 bucket 初始化；初始化仅创建私有 bucket。

API 普通配置放在 `config/api.config.toml`，密钥由 `.env` 注入。API 以 UID 10001 运行，挂载的 TOML 要允许该用户读取，建议文件只存非敏感配置并使用 0644 权限。容器日志写入 stdout 和临时目录，长期日志由平台采集。

### 单机完整部署

```bash
python3 scripts/deploy-compose.py config full
python3 scripts/deploy-compose.py pull full  # 使用本地构建镜像时跳过
python3 scripts/deploy-compose.py up full
```

启动 PostgreSQL、Redis、RustFS，等待健康；创建 bucket；执行一次性数据库初始化；最后启动 API、Web、Nginx。重复执行 `up full` 会重新运行初始化，因此不会复用旧版本已退出容器的成功状态。

默认入口为 `http://localhost:8080`。仅 Nginx 发布端口；数据库和应用服务不直接发布到宿主机。HTTPS 由服务器已有 TLS 代理终止并转发到此端口；默认只绑定 `127.0.0.1`，仅当 TLS 代理在另一台主机上时才设 `HTTP_BIND=0.0.0.0`。非开发模式下登录 Cookie 带 `Secure`，浏览器只在 HTTPS 或 `http://localhost` 上保留会话：通过 `http://<服务器 IP>:8080` 这类明文地址登录，看似成功却会回到登录页。登录、上传、下载均使用入口域名。代理需保留原始 Host；在其信任边界内配置转发头，上传限制至少为 100 MB，并为流式请求配置合适的超时和缓冲策略。

### 使用外部基础设施

编辑 `.env` 的数据库、Redis、S3 地址和凭据，并在 TOML 配置数据库 SSL、Redis TLS/Sentinel/Cluster、S3 Secure/Region 等选项（AWS S3 的 `Region` 必须与 bucket 所在地域一致，不会自动探测）。外部 S3 bucket 需事先创建。

```bash
python3 scripts/deploy-compose.py config external
python3 scripts/deploy-compose.py up external
```

此模式不会创建、停止或备份外部基础设施。Compose 的 `CASTOR_DB_PORT` 可覆盖默认 5432；复杂 Redis 拓扑需同时确保配置校验使用的 Address 已填写。

### 多套实例共用基础设施

一台主机上用同一套 PostgreSQL、Redis、RustFS 运行多套 Castor（各自独立的 API + Web + 网关）。隔离方式：

| 资源 | 每套实例独占 | 由谁创建 |
|------|--------------|----------|
| PostgreSQL | 数据库 + 同名角色（`REVOKE CONNECT FROM PUBLIC`，其他实例的角色连不上） | `compose-instance.py provision` |
| RustFS | bucket + 只能读写该 bucket 对象的用户（不能访问其他 bucket、不能改 bucket 策略） | `compose-instance.py provision` |
| Redis | key 前缀 `CASTOR_INSTANCE_ID`（共用密码，逻辑隔离） | API 启动时认领 |
| 密钥 | `CASTOR_JWT_KEY`、`CASTOR_RSA_SECRET`、`CASTOR_DATA_KEY`、管理员初始密码 | `compose-instance.py new` 随机生成 |

`CASTOR_INSTANCE_ID` 同时是 JWT 的 `aud`：一套实例签发的 token 在另一套上无效。API 启动时把 Redis 命名空间登记到本库的部署标识（`init-db` 生成）名下，另一套部署误用相同的 `CASTOR_INSTANCE_ID` 会启动失败并说明原因，而不是悄悄共用 RSA 密钥、字典缓存和登录限流。

```bash
# 1. 共用基础设施（只需一次）
cp deploy/compose/.env.shared.example deploy/compose/.env.shared
chmod 600 deploy/compose/.env.shared      # 填写 POSTGRES_PASSWORD、REDIS_PASSWORD、S3_SECRET_KEY
python3 scripts/deploy-compose.py up shared

# 2. 每套实例：生成 env（随机密钥、库名、bucket），按需修改镜像、端口、CORS 域名
python3 scripts/compose-instance.py new tenant-a 8081   # → deploy/compose/instances/tenant-a.env
python3 scripts/deploy-compose.py up instance deploy/compose/instances/tenant-a.env
```

`up instance` 依次开通数据库与 bucket（可重复执行，会把角色密码和 RustFS 用户密钥同步为 env 中的值）、运行 `init-db`、启动应用。管理员初始密码在生成的 env 文件里。更新、回滚与单实例部署相同，只是把 `full` 换成 `instance <env-file>`；`down instance <env-file>` 只停这一套。

要一次操作所有实例（例如发布新版本），使用批量命令。它们按实例标识顺序执行，遇到第一个失败就停下，半新半旧的状态不会被忽略；`--keep-going` 会继续执行，但仍以错误退出：

```bash
python3 scripts/compose-instance.py status
python3 scripts/compose-instance.py all pull
python3 scripts/compose-instance.py all up
python3 scripts/compose-instance.py all backup
```

- 各实例的网关分别发布 `HTTP_PORT`，由前置 TLS 代理按域名分发；**每套实例必须使用独立域名或子域名**。登录态是不带 Domain 的 host-only cookie，同一域名下按路径区分实例会互相覆盖登录态。
- 只有 API 加入共用网络 `SHARED_NETWORK`，按服务名 `postgres`、`redis`、`rustfs` 访问基础设施；Web 与网关留在实例自己的网络里，`castor-api` 只会解析到本实例。
- 所有实例共用 `config/api.config.toml`（实例差异都在 env 中）；需要单独配置时在 env 里设 `API_CONFIG_FILE`。
- PostgreSQL 连接数：每个 API 副本最多 `MaxOpenConns`（默认 25）个连接，实例数 × 副本数 × 25 不能超过 `max_connections`（默认 100），超出时调小 `MaxOpenConns` 或提高上限。
- 需要 Redis 权限隔离（而不只是 key 隔离）时，使用外部 Redis，为每套实例创建 ACL 用户（`~<instance-id>:* +@all -@dangerous`），并在 TOML 的 `[Redis]` 中填写 `Username`；密码仍经 `CASTOR_REDIS_PASSWORD` 注入。
- 远程发布（`deploy-compose.py remote`）只支持单实例；多实例在目标主机上直接执行上述命令。

### 本地开发

本地开发不走本节脚本：`python3 scripts/dev.py prepare` 启动独立的基础设施并生成配置，见根目录 README。

### 远程发布

先准备本地 `deploy/compose/.env` 和 `config/api.config.toml`。脚本经 SSH 传送 Compose 配置和密钥，在远程主机上执行 `docker compose`，远程初始化成功后启动应用；不自动构建镜像。

```bash
# 仓库方式
python3 scripts/deploy-compose.py remote registry deploy@example.com /opt/castor full

# tar 方式（应用镜像提前构建/导出）
python3 scripts/deploy-compose.py remote tar deploy@example.com /opt/castor full deploy/artifacts/castor-v1.0.0.tar.gz
```

目标机只需要 Docker Compose v2（支持 `up/start --wait`）和 tar，不需要 Python 或 Bash。tar 包只含应用镜像；离线环境还需预加载 Compose 清单中的 PostgreSQL、Redis、RustFS、rc、Nginx 镜像。远程主机首次连接及认证使用 SSH 自身机制。

### 验证与更新

```bash
curl -f http://localhost:8080/healthz
curl -f http://localhost:8080/api/v1/settings/public
docker compose --project-directory deploy/compose -f deploy/compose/compose.yaml ps
```

API 容器的就绪检查访问 `/ready`，验证 PostgreSQL 和 Redis。实际验收还应完成登录、文件上传和下载，以覆盖对象存储链路；并给自己发一条通知，铃铛应立即更新，否则说明前置代理缓冲了事件流（`/api/v1/account/notifications/stream`）。

后续发布修改两个镜像版本，再执行 `pull`、`up`。`up` 会重新创建应用容器，确保 TOML 和代理配置更新生效。Compose 更新期间可能有短暂中断。

应用回滚先确认数据库兼容，再将两个镜像引用及配置改回旧版本：

```bash
python3 scripts/deploy-compose.py pull full
python3 scripts/deploy-compose.py rollback full
```

`rollback` 只更新应用，不运行旧版本数据库初始化。

## Kubernetes

要求已有集群、kubectl、支持该集群的入口控制器、外部 PostgreSQL/Redis/S3，以及镜像仓库访问权限。此目录不管理数据库集群和存储系统。

### 准备环境

```bash
cp deploy/config/api.config.example.toml deploy/k8s/overlays/staging/api.config.toml
cp deploy/k8s/overlays/staging/secrets.env.example deploy/k8s/overlays/staging/secrets.env
cp deploy/k8s/overlays/staging/bootstrap.env.example deploy/k8s/overlays/staging/bootstrap.env
chmod 600 deploy/k8s/overlays/staging/{secrets,bootstrap}.env
```

生产环境将上述 `staging` 替换为 `production`。

每个 overlay 是一套实例（独立 namespace），`staging` 与 `production` 是示例。按"每个租户一套实例"部署时，生成实例 overlay：`new` 把 `production` 复制到 `deploy/k8s/instances/<id>/`，设置 namespace `castor-<id>`、域名和 `CASTOR_INSTANCE_ID=<id>`，并生成 JWT 密钥、RSA 密钥和管理员初始密码。之后在 `secrets.env` 和 TOML 中填写外部凭据（独立的数据库与角色、bucket 与访问密钥）；存在空值时 `apply` 会拒绝执行。`apply-all` / `rollback-all` 按实例标识顺序处理所有实例，遇到第一个失败就停下（`--keep-going` 会继续执行，但仍以错误退出）。

```bash
python3 scripts/deploy-k8s.py new tenant-a tenant-a.example.com
python3 scripts/deploy-k8s.py apply tenant-a your-kube-context
python3 scripts/deploy-k8s.py apply-all your-kube-context
```

- TOML：设置外部服务地址、数据库名/用户、TLS、S3 bucket/region、CORS 域名。敏感字段由 Secret 注入。
- `secrets.env`：填写运行时密钥；`bootstrap.env` 只包含管理员初始密码，只有初始化 Job 使用。
- `kustomization.yaml`：替换两个镜像的仓库和版本、域名；按集群配置 Ingress class、私有仓库 `imagePullSecrets`（应用与初始化模板都要配置）。
- TLS：在目标 namespace 预置名为 `castor-tls` 的 TLS Secret，或使用已有证书控制器生成它。
- 根据实际入口控制器配置至少 100 MB 上传限制、流式请求和超时。没有默认 IngressClass 的集群必须显式指定 class。

ConfigMap / Secret 使用内容哈希名称，配置变化会更新 Pod 模板并触发滚动更新。真实配置文件和 Secret env 文件均被 Git 忽略。`render` 输出包含 Secret 数据，不要提交或发到公开日志。

### 部署

```bash
python3 scripts/deploy-k8s.py render staging deploy/artifacts/castor-staging.yaml
# 检查内容后删除该临时文件
python3 scripts/deploy-k8s.py apply staging your-kube-context
python3 scripts/deploy-k8s.py apply production your-kube-context
```

脚本要求显式指定 context，避免误用当前集群。按顺序应用 namespace、获取环境发布锁、应用配置和初始化模板、创建当前版本 Job、等待完成、应用 Deployment/Ingress、等待滚动更新。

`castor-init-db` 是禁用调度的 CronJob，只作为每次发布 Job 的模板，永远不会定时执行。初始化超时或失败时停止发布，保留原应用 Deployment；Job 日志用于排查，Job 在一天后自动清理。

同一 namespace 的脚本发布通过 `castor-release-lock` ConfigMap 串行化。进程被强制终止后，如锁残留，先确认没有发布进程和初始化任务运行，再删除这个 ConfigMap。脚本正常退出时会清理锁。

staging 为一个副本；production 由 HPA 管理副本数（最少两个）。清单不写 `replicas`，重复 `apply` 不会把 HPA 扩出的副本压回去。API 使用 `/health`、`/ready` 和启动探针；终止宽限期 30 秒，大于默认应用退出时间 10 秒。资源请求/限制可按负载调整。

```bash
kubectl --context your-kube-context -n castor-staging get pods,jobs,ingress
kubectl --context your-kube-context -n castor-staging logs deployment/castor-api
curl -f https://castor-staging.example.com/api/v1/settings/public
```

### 更新和回滚

更新 overlay 镜像版本及配置，再执行 `apply`。初始化涉及破坏性变更时，先安排维护窗口。

回滚前确认数据库可被旧应用读取，再恢复 overlay 中旧版本的镜像和配置，然后执行：

```bash
python3 scripts/deploy-k8s.py rollback production your-kube-context
```

该命令获取发布锁、应用指定配置并更新应用，不创建初始化 Job。
`kubectl rollout undo` 只处理 Deployment，不会回滚数据库、Job、Ingress 或外部服务。历史 ConfigMap/Secret 暂不自动删除，以便排查与回滚；确认不再被 Deployment 历史版本引用后再清理。

## 备份和恢复

`scripts/backup.py` 只负责 Compose 自托管数据，以短暂停机的物理快照保存 PostgreSQL、Redis（含 AOF）、RustFS 全部数据卷，另外归档部署配置。不会修改外部服务。它读取 Compose 项目内的实际容器，不依赖固定容器名。

```bash
python3 scripts/backup.py create full
python3 scripts/backup.py list
python3 scripts/backup.py restore full snapshot-20260909_120000 --confirm snapshot-20260909_120000
```

备份/恢复前拉取 Alpine 工具镜像，再停止项目服务；成功后恢复原来运行的服务。备份失败也会恢复服务，恢复失败则保持停止，防止提供部分恢复的数据。

物理恢复要求三个基础设施容器镜像 ID 与备份时一致；数据库跨版本升级使用专门迁移工具。恢复操作会覆盖现有数据卷，必须显式传入 `--confirm <snapshot>` 才执行。`config.tar` 仅供核对，不会覆盖现有部署配置；凭据变更后需手动核对匹配。

共用基础设施的整体冷快照用 `shared` 替换 `full`：它会停掉 PostgreSQL、Redis、RustFS，期间所有实例不可用，应先停止各实例。备份或恢复单套实例用逻辑快照，只停这一套的 API：

```bash
python3 scripts/compose-instance.py backup deploy/compose/instances/tenant-a.env
python3 scripts/compose-instance.py list deploy/compose/instances/tenant-a.env
python3 scripts/compose-instance.py restore deploy/compose/instances/tenant-a.env snapshot-20260921_120000 --confirm tenant-a/snapshot-20260921_120000
```

快照包含 `pg_dump` 自定义格式的数据库、bucket 镜像和当时的 env 文件，位于 `deploy/backups/instances/<instance-id>/`。恢复用 `pg_restore --clean` 覆盖该实例的库、`rc mirror --remove` 让 bucket 与快照一致，失败时 API 保持停止。只恢复到同一应用版本，跨版本先恢复再运行 `init-db`。Redis 中只有会过期的运行时数据，不备份。

快照位于 `deploy/backups/`，包含数据及密钥，创建为仅当前用户可读的目录，需自行转存到安全的备份位置。K8s 和外部数据库使用对应平台的备份/PITR、对象存储版本控制，并定期验证恢复。

## 仓库验证

`scripts/check.py`（见根目录 AGENTS.md）。其中 `bootstrap` 集成测试使用 `CASTOR_TEST_POSTGRES_DSN`，必须指向单独的空测试数据库；它验证两个初始化任务并发执行、重复执行不重置管理员密码、角色分配不重复。
