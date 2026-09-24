# Deployment Guide

**English** | [简体中文](README.zh-CN.md) | [日本語](README.ja.md) | [한국어](README.ko.md)

Castor supports two deployment methods: Docker Compose and Kubernetes. Both use the same API / Web images and serve everything from a single domain, routing `/api/v1/*` to the API and all other requests to Web. The Dockerfiles live in `apps/api` and `apps/web`.

The scripts under `scripts/` need only Python 3.9+ (standard library) plus the tools they drive (Docker, kubectl, ssh), and run the same on Windows, macOS, and Linux. On Windows, run them with `py -3` instead of `python3`; the `cp` / `chmod` examples apply to Linux and macOS hosts, while on Windows copy the files as usual and rely on NTFS permissions.

## Release model

1. Build both images with an explicit version tag; never overwrite a published tag.
2. Prepare PostgreSQL, Redis, S3, and the API configuration.
3. Run `init-db` with the current API image.
4. Once initialization succeeds, update the API and Web and wait for them to become ready.

`init-db` holds an advisory lock inside a PostgreSQL transaction, runs schema migrations, and reconciles system settings, dictionaries, permission resources, roles, menus, and the administrator on an add-only basis (on the first run it also generates the deployment ID). On failure it rolls back and exits non-zero. Re-running it does not reset the administrator password. A normal API start only reads the existing database and permission configuration; it does not run migrations.

Updating images and changing the database are separate operations. For incompatible database changes, schedule a maintenance window and back up your data; do not rely on a rolling update alone. Old images are not guaranteed to work with a newer database.

## Choosing a tenancy model

Castor serves one organization per deployment. To host several customers ("tenants"), run one instance per tenant: its own database and role, bucket, Redis key prefix, secrets, and domain, on shared or separate infrastructure (Compose: "Sharing infrastructure between instances" below; Kubernetes: one instance overlay per tenant). No two tenants share a database, so no query can leak data between them, and each tenant can be backed up, restored, moved, or deleted on its own.

| Need | Recommended |
|------|------|
| Departments or subsidiaries of one organization see only their own data | One instance with departments and role data scopes |
| A few to a few dozen customers, strong isolation (government, healthcare, finance) | One instance per tenant |
| Hundreds of self-service tenants created at sign-up, cross-tenant billing | Row-level tenancy built into the downstream product (not provided) |

Each instance has a steady footprint (API and Web processes, database connections; see the `max_connections` note below) and is upgraded on its own; the batch commands below keep that manageable. Row-level tenancy (a `tenant_id` on every row of a shared database) is deliberately not part of Castor, because every downstream project would carry its cost and risk. A product that needs it should plan at least for:

- `tenant_id` on every table, composite unique constraints (usernames, contacts, role/department/dictionary codes, setting keys), and a filter that no repository query or raw SQL can bypass (a GORM scope, with PostgreSQL row-level security as defense in depth);
- deciding for each system table whether its data is global or per tenant (menus, resources, dictionaries, settings), and reworking the seed reconciliation to match;
- asset deduplication per tenant (a global content hash reveals that another tenant holds the same file);
- resolving the tenant at sign-in (domain or code) and carrying it in the JWT, sessions, Redis keys, and rate limits;
- bounding "everyone" notifications, dashboards, and data scopes by the tenant, with a separately authorized cross-tenant operator.

## Building images

Requires Docker + Buildx. Builds target `linux/amd64` by default; set `PLATFORM=linux/arm64` for local verification on ARM hosts. The API build uses the official Go/Ubuntu sources by default; build machines that cannot reach them (for example in mainland China) set the environment variables `GOPROXY`, `UBUNTU_MIRROR`, and `UBUNTU_SECURITY_MIRROR` when running the script, which passes them to the build as arguments, e.g. `GOPROXY=https://goproxy.cn,direct`.

```bash
# Load images locally
python3 scripts/build-images.py load castor local

# Push to a registry (run docker login on both the build and deploy machines)
python3 scripts/build-images.py push registry.example.com/castor v1.0.0

# Ship application images offline
python3 scripts/build-images.py tar castor v1.0.0 deploy/artifacts/castor-v1.0.0.tar.gz
```

Web uses the Node Dockerfile by default so the runtime matches the health check command. Production routing is handled by the entry point, so environment domains do not need to be baked into the frontend image. Any customized `NEXT_PUBLIC_*` values are still build-time configuration.

## Compose

### Configuration

```bash
cp deploy/compose/.env.example deploy/compose/.env
cp deploy/config/api.config.example.toml deploy/compose/config/api.config.toml
chmod 600 deploy/compose/.env
```

Fill in every empty secret in `.env`: the JWT key should be at least 32 bytes; the RSA secret must be exactly 32 ASCII bytes; the administrator password must be at least 12 characters. Generate the JWT key and similar secrets with `python3 -c "import secrets; print(secrets.token_hex(32))"`, and the RSA secret with the same command using 16 instead of 32. Use a distinct value for each. Never commit a real `.env`.

SMS and email verification codes are optional: if you don't use them, leave `CASTOR_SMS_*` in `.env` and `[Mail].Host` in the TOML empty. The service starts normally, and users who pick that method are told it is not configured. To enable SMS, set both `CASTOR_SMS_ACCESS_KEY` and `CASTOR_SMS_SECRET_KEY` plus `SignName` and `TemplateCode` in the TOML; to enable email, fill in the `[Mail]` Host and account, with the password injected via `CASTOR_MAIL_PASSWORD`. On K8s, add the same variables to `secrets.env` as needed.

`CASTOR_INSTANCE_ID` is the instance identifier (Redis key prefix and JWT `aud`); a single instance can keep the default `castor`. When sharing Redis with other deployments, see "Sharing infrastructure between instances" below. `API_IMAGE` / `WEB_IMAGE` must match what you built. `S3_BUCKET` is used both in the API configuration and for initializing the self-hosted bucket; initialization only creates a private bucket.

Regular API settings go in `config/api.config.toml`; secrets are injected from `.env`. The API runs as UID 10001, so the mounted TOML must be readable by that user. Keep only non-sensitive settings in the file and use 0644 permissions. Container logs go to stdout and a temporary directory; long-term log retention is handled by the platform.

### Single-host full deployment

```bash
python3 scripts/deploy-compose.py config full
python3 scripts/deploy-compose.py pull full  # Skip when using locally built images
python3 scripts/deploy-compose.py up full
```

This starts PostgreSQL, Redis, and RustFS and waits for them to become healthy; creates the bucket; runs the one-off database initialization; and finally starts the API, Web, and Nginx. Re-running `up full` reruns initialization, so it never reuses the success status of an exited container from an older version.

The default entry point is `http://localhost:8080`. Only Nginx publishes a port; the database and application services are not exposed on the host. HTTPS is terminated by the server's existing TLS proxy, which forwards to this port. By default it binds only to `127.0.0.1`; set `HTTP_BIND=0.0.0.0` only when the TLS proxy runs on another host. Outside development mode the login cookies are `Secure`, so browsers keep a session only over HTTPS or on `http://localhost`: signing in through a plain-HTTP address such as `http://<server-ip>:8080` appears to succeed but lands back on the sign-in page. Login, upload, and download all use the entry domain. The proxy must preserve the original Host, configure forwarded headers within its trust boundary, allow uploads of at least 100 MB, and use suitable timeouts and buffering for streaming requests.

### Using external infrastructure

Edit the database, Redis, and S3 addresses and credentials in `.env`, and configure database SSL, Redis TLS/Sentinel/Cluster, S3 Secure/Region, and similar options in the TOML (for AWS S3, `Region` must match the bucket's region; it is not auto-detected). External S3 buckets must be created in advance.

```bash
python3 scripts/deploy-compose.py config external
python3 scripts/deploy-compose.py up external
```

This mode never creates, stops, or backs up external infrastructure. Compose's `CASTOR_DB_PORT` overrides the default 5432. For complex Redis topologies, make sure the Address used by config validation is also filled in.

### Sharing infrastructure between instances

Run multiple Castor instances (each with its own API + Web + gateway) on one host against a single PostgreSQL, Redis, and RustFS. Isolation works as follows:

| Resource | Dedicated per instance | Created by |
|------|--------------|----------|
| PostgreSQL | Database + role of the same name (`REVOKE CONNECT FROM PUBLIC`, so other instances' roles cannot connect) | `compose-instance.py provision` |
| RustFS | Bucket + a user that can only read/write objects in that bucket (no access to other buckets, cannot change bucket policies) | `compose-instance.py provision` |
| Redis | Key prefix `CASTOR_INSTANCE_ID` (shared password, logical isolation) | Claimed at API startup |
| Secrets | `CASTOR_JWT_KEY`, `CASTOR_RSA_SECRET`, `CASTOR_DATA_KEY`, initial administrator password | Randomly generated by `compose-instance.py new` |

`CASTOR_INSTANCE_ID` is also the JWT `aud`: tokens issued by one instance are invalid on another. At startup the API registers its Redis namespace under the deployment ID stored in its own database (generated by `init-db`). If another deployment mistakenly uses the same `CASTOR_INSTANCE_ID`, it fails to start and explains why, instead of silently sharing the RSA key, dictionary cache, and login rate limits.

```bash
# 1. Shared infrastructure (once only)
cp deploy/compose/.env.shared.example deploy/compose/.env.shared
chmod 600 deploy/compose/.env.shared      # Fill in POSTGRES_PASSWORD, REDIS_PASSWORD, S3_SECRET_KEY
python3 scripts/deploy-compose.py up shared

# 2. Per instance: generate the env (random secrets, database name, bucket); adjust image, port, and CORS domains as needed
python3 scripts/compose-instance.py new tenant-a 8081   # → deploy/compose/instances/tenant-a.env
python3 scripts/deploy-compose.py up instance deploy/compose/instances/tenant-a.env
```

`up instance` provisions the database and bucket (idempotent; it syncs the role password and RustFS user secret to the values in the env), runs `init-db`, and starts the application. The initial administrator password is in the generated env file. Updates and rollbacks work the same as for a single instance, just replace `full` with `instance <env-file>`; `down instance <env-file>` stops only that instance.

To operate every instance at once, for example to roll out a release, use the batch commands. They process instances in id order and stop at the first failure, so a half-upgraded fleet never goes unnoticed; `--keep-going` continues but still exits with an error:

```bash
python3 scripts/compose-instance.py status
python3 scripts/compose-instance.py all pull
python3 scripts/compose-instance.py all up
python3 scripts/compose-instance.py all backup
```

- Each instance's gateway publishes its own `HTTP_PORT`, and the front TLS proxy routes by domain; **every instance must use its own domain or subdomain**. The login session is a host-only cookie without a Domain attribute, so instances separated by path under the same domain overwrite each other's sessions.
- Only the API joins the shared network `SHARED_NETWORK` and reaches the infrastructure by the service names `postgres`, `redis`, and `rustfs`; Web and the gateway stay on the instance's own network, so `castor-api` only ever resolves to that instance.
- All instances share `config/api.config.toml` (per-instance differences all live in the env); set `API_CONFIG_FILE` in the env when an instance needs its own configuration.
- PostgreSQL connections: each API replica opens at most `MaxOpenConns` (default 25) connections. Instances × replicas × 25 must not exceed `max_connections` (default 100); if it does, lower `MaxOpenConns` or raise the limit.
- If you need Redis permission isolation (not just key isolation), use an external Redis, create an ACL user per instance (`~<instance-id>:* +@all -@dangerous`), and set `Username` under `[Redis]` in the TOML; the password is still injected via `CASTOR_REDIS_PASSWORD`.
- Remote deployment (`deploy-compose.py remote`) supports a single instance only; for multiple instances, run the commands above directly on the target host.

### Local development

Local development does not use the scripts in this section: `python3 scripts/dev.py prepare` starts separate infrastructure and generates the configuration; see the root README.

### Remote deployment

First prepare `deploy/compose/.env` and `config/api.config.toml` locally. The script ships the Compose configuration and secrets over SSH, runs `docker compose` on the remote host, and starts the application after remote initialization succeeds; it does not build images.

```bash
# Registry mode
python3 scripts/deploy-compose.py remote registry deploy@example.com /opt/castor full

# Tar mode (application images built/exported in advance)
python3 scripts/deploy-compose.py remote tar deploy@example.com /opt/castor full deploy/artifacts/castor-v1.0.0.tar.gz
```

The target host needs only Docker Compose v2 (with `up/start --wait` support) and tar; neither Python nor Bash. The tarball contains only the application images; offline environments must also preload the PostgreSQL, Redis, RustFS, rc, and Nginx images listed in the Compose manifest. The first connection to and authentication with the remote host use SSH's own mechanisms.

### Verification and updates

```bash
curl -f http://localhost:8080/healthz
curl -f http://localhost:8080/api/v1/settings/public
docker compose --project-directory deploy/compose -f deploy/compose/compose.yaml ps
```

The API container's readiness check hits `/ready`, which verifies PostgreSQL and Redis. Real acceptance testing should also cover login, file upload, and download to exercise the object storage path, and send yourself a notification: the bell must update at once, otherwise a proxy in front is buffering the event stream (`/api/v1/account/notifications/stream`).

For subsequent releases, change both image versions, then run `pull` and `up`. `up` recreates the application containers so TOML and proxy configuration changes take effect. Compose updates may cause a brief interruption.

To roll back the application, first confirm database compatibility, then revert both image references and the configuration to the old version:

```bash
python3 scripts/deploy-compose.py pull full
python3 scripts/deploy-compose.py rollback full
```

`rollback` only updates the application; it does not run the old version's database initialization.

## Kubernetes

Requires an existing cluster, kubectl, an ingress controller supported by that cluster, external PostgreSQL/Redis/S3, and access to the image registry. This directory does not manage database clusters or storage systems.

### Preparing the environment

```bash
cp deploy/config/api.config.example.toml deploy/k8s/overlays/staging/api.config.toml
cp deploy/k8s/overlays/staging/secrets.env.example deploy/k8s/overlays/staging/secrets.env
cp deploy/k8s/overlays/staging/bootstrap.env.example deploy/k8s/overlays/staging/bootstrap.env
chmod 600 deploy/k8s/overlays/staging/{secrets,bootstrap}.env
```

For production, replace `staging` above with `production`.

Each overlay is one instance (its own namespace); `staging` and `production` are examples. For one instance per tenant, generate an instance overlay: `new` copies `production` into `deploy/k8s/instances/<id>/` with the namespace `castor-<id>`, the domain, and `CASTOR_INSTANCE_ID=<id>`, and generates the JWT key, RSA secret, and initial administrator password. Then fill in the external credentials in `secrets.env` and the TOML (its own database and role, bucket and access keys); `apply` refuses empty values. `apply-all` / `rollback-all` process every instance in id order and stop at the first failure (`--keep-going` continues but still exits with an error).

```bash
python3 scripts/deploy-k8s.py new tenant-a tenant-a.example.com
python3 scripts/deploy-k8s.py apply tenant-a your-kube-context
python3 scripts/deploy-k8s.py apply-all your-kube-context
```

- TOML: set external service addresses, database name/user, TLS, S3 bucket/region, and CORS domains. Sensitive fields are injected from the Secret.
- `secrets.env`: fill in the runtime secrets; `bootstrap.env` contains only the initial administrator password and is used only by the initialization Job.
- `kustomization.yaml`: replace the registry and version of both images and the domain; configure the Ingress class and private registry `imagePullSecrets` for your cluster (in both the application and initialization templates).
- TLS: pre-create a TLS Secret named `castor-tls` in the target namespace, or have an existing certificate controller generate it.
- Configure at least a 100 MB upload limit, streaming requests, and timeouts for your ingress controller. Clusters without a default IngressClass must specify the class explicitly.

ConfigMaps / Secrets use content-hashed names, so configuration changes update the Pod template and trigger a rolling update. Real configuration files and Secret env files are ignored by Git. `render` output contains Secret data; never commit it or post it to public logs.

### Deployment

```bash
python3 scripts/deploy-k8s.py render staging deploy/artifacts/castor-staging.yaml
# Review the contents, then delete the temporary file
python3 scripts/deploy-k8s.py apply staging your-kube-context
python3 scripts/deploy-k8s.py apply production your-kube-context
```

The script requires an explicit context to avoid accidentally targeting the current cluster. In order, it applies the namespace, acquires the environment release lock, applies the configuration and initialization template, creates the Job for the current version, waits for it to complete, applies the Deployments/Ingress, and waits for the rollout.

`castor-init-db` is a CronJob with scheduling disabled; it serves only as the template for each release's Job and never runs on a schedule. If initialization times out or fails, the release stops and the existing application Deployments are left untouched; the Job logs are kept for troubleshooting, and the Job is cleaned up automatically after one day.

Script releases in the same namespace are serialized through the `castor-release-lock` ConfigMap. If the process is killed and the lock is left behind, first confirm that no release process or initialization task is running, then delete that ConfigMap. The script cleans up the lock on normal exit.

Staging runs one replica; in production the HPA manages the replica count (minimum two). The manifests do not set `replicas`, so re-running `apply` does not shrink replicas scaled out by the HPA. The API uses `/health`, `/ready`, and a startup probe; the termination grace period is 30 seconds, longer than the default application shutdown time of 10 seconds. Adjust resource requests/limits to your load.

```bash
kubectl --context your-kube-context -n castor-staging get pods,jobs,ingress
kubectl --context your-kube-context -n castor-staging logs deployment/castor-api
curl -f https://castor-staging.example.com/api/v1/settings/public
```

### Updates and rollbacks

Update the image versions and configuration in the overlay, then run `apply`. If initialization involves destructive changes, schedule a maintenance window first.

Before rolling back, confirm the old application can read the database, restore the old image versions and configuration in the overlay, then run:

```bash
python3 scripts/deploy-k8s.py rollback production your-kube-context
```

This command acquires the release lock, applies the specified configuration, and updates the application without creating an initialization Job.
`kubectl rollout undo` only handles Deployments; it does not roll back the database, Jobs, Ingress, or external services. Old ConfigMaps/Secrets are not deleted automatically, to aid troubleshooting and rollback; clean them up once you have confirmed no Deployment revision still references them.

## Backup and restore

`scripts/backup.py` covers only Compose self-hosted data. It takes physical snapshots with brief downtime of all PostgreSQL, Redis (including AOF), and RustFS data volumes, and also archives the deployment configuration. It never modifies external services. It works on the actual containers in the Compose project and does not depend on fixed container names.

```bash
python3 scripts/backup.py create full
python3 scripts/backup.py list
python3 scripts/backup.py restore full snapshot-20260909_120000 --confirm snapshot-20260909_120000
```

Before a backup/restore it pulls the Alpine tools image, then stops the project's services; on success it restarts the services that were running. A failed backup still restarts the services, but a failed restore leaves them stopped so partially restored data is never served.

A physical restore requires the image IDs of the three infrastructure containers to match those at backup time; use dedicated migration tools for cross-version database upgrades. A restore overwrites the existing data volumes and runs only with an explicit `--confirm <snapshot>`. `config.tar` is for reference only and never overwrites the current deployment configuration; after credential changes, check that they match manually.

For a full cold snapshot of the shared infrastructure, replace `full` with `shared`: it stops PostgreSQL, Redis, and RustFS, making every instance unavailable meanwhile, so stop the instances first. To back up or restore a single instance, use a logical snapshot, which stops only that instance's API:

```bash
python3 scripts/compose-instance.py backup deploy/compose/instances/tenant-a.env
python3 scripts/compose-instance.py list deploy/compose/instances/tenant-a.env
python3 scripts/compose-instance.py restore deploy/compose/instances/tenant-a.env snapshot-20260921_120000 --confirm tenant-a/snapshot-20260921_120000
```

A snapshot contains the database in `pg_dump` custom format, a mirror of the bucket, and the env file at that time, stored in `deploy/backups/instances/<instance-id>/`. Restore uses `pg_restore --clean` to overwrite the instance's database and `rc mirror --remove` to make the bucket match the snapshot; on failure the API stays stopped. Restore only to the same application version; for a different version, restore first and then run `init-db`. Redis holds only expiring runtime data and is not backed up.

Snapshots live in `deploy/backups/`, contain data and secrets, and are created in directories only the current user can read; copy them to a secure backup location yourself. For K8s and external databases, use the platform's backup/PITR and object storage versioning, and test restores regularly.

## Repository verification

`scripts/check.py` (see the root AGENTS.md). Its `bootstrap` integration tests use `CASTOR_TEST_POSTGRES_DSN`, which must point to a separate, empty test database; they verify that two initialization tasks can run concurrently, that re-running does not reset the administrator password, and that role assignments are not duplicated.
