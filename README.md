# Castor

**English** | [简体中文](README.zh-CN.md) | [日本語](README.ja.md) | [한국어](README.ko.md)

**A full-stack scaffold for vibe coding.**

Castor gives you and your AI coding agent an extensible application foundation built with Go + Next.js. Describe your business requirements in natural language and build on existing authentication, permissions, file management, internationalization, and deployment tooling to turn ideas into working full-stack applications.

## Why Castor for vibe coding

- **Give your agent clear instructions**: root and application-level `AGENTS.md` files define architecture, code patterns, API contracts, and a checklist for adding modules, shared by you and your AI coding agent.
- **Extend complete features**: a public website and admin dashboard include users, roles, menus, assets, notifications, dictionaries, and audit logs as references for new business modules.
- **Check constraints across layers**: a single verification entry point covers formatting, static analysis, tests, and the frontend build. Guardrail tests check architectural dependencies, RBAC resource registration, dictionary contracts, and translations across four languages.
- **Move from development to deployment**: VS Code F5 prepares the local environment and starts both debuggers, with Compose / Kubernetes deployment and backup tooling included.

## Build with AI

1. Follow the quick start below and explore the built-in features.
2. Ask your AI coding agent to read [AGENTS.md](AGENTS.md) and the relevant application guides, then describe your business requirements, fields, permissions, and acceptance criteria.
3. Follow existing modules to implement the database, API, permissions, menus, frontend, and translations in all four languages, then run `scripts/check.py` to verify the changes.

For example, start with a request like this:

```text
Read AGENTS.md and the backend and frontend development guides. Following
the existing paginated CRUD modules, add project management with name,
description, and status, supporting listing, filtering, creation, editing,
and deletion. Include database migrations, API endpoints, RBAC resources,
menus, a status dictionary, frontend pages, Chinese/English/Japanese/Korean
translations, and tests. Run scripts/check.py to verify the changes.
```

## Project structure

```text
apps/
├── api/                     # Go API
└── web/                     # Next.js web application

deploy/
├── config/                  # Shared API configuration templates
├── compose/                 # Single-host, shared multi-instance, and local infrastructure
└── k8s/                     # Kubernetes Kustomize base + overlays

scripts/
├── check.py                 # Unified verification for developers and AI agents
├── dev.py                   # Local environment preparation (used by F5)
├── build-images.py          # Build, push, and export images
├── deploy-compose.py        # Local and remote Compose deployment
├── compose-instance.py      # Shared infrastructure: instance config, provisioning, backup/restore
├── deploy-k8s.py            # Kubernetes initialization and deployment
└── backup.py                # Backup and restore for self-hosted Compose deployments
```

## Tech stack

### Backend

| Component | Technology |
|-----------|------------|
| Language | Go 1.26.0+ |
| Web framework | Gin |
| ORM | GORM |
| Database | PostgreSQL |
| Cache | Redis |
| Object storage | S3 (RustFS compatible) |
| Access control | RBAC3 (role hierarchies, SSD, DSD, authorization sessions) |
| Authentication | JWT |
| Dependency injection | Wire |

### Frontend

| Component | Technology |
|-----------|------------|
| Framework | Next.js 16 (App Router) |
| UI | React 19 + shadcn/ui + Tailwind CSS 4 |
| State management | Zustand |
| Data fetching | TanStack Query |
| Forms | TanStack Form |
| Tables | TanStack Table |

## Quick start

### Debug with VS Code (F5)

Open the repository root in VS Code, install the recommended Go extension, and press **F5** with the `Castor` debug configuration selected. Install Docker Desktop, Go, Node.js 22+, Bun, Python 3.11+, and Chrome first; database services do not need to be installed manually.

On Windows, install the Python launcher (`py`) along with Python. VS Code tasks use `py -3` to avoid an older Windows Store `python3` alias. Check the selected version with `py -3 --version` (3.11+ required); for manual commands below, replace `python3` with `py -3`.

The first launch creates PostgreSQL, Redis, RustFS, a storage bucket, and development configuration, installs dependencies, initializes the database, then starts the Go / Next.js debuggers and opens the browser. Subsequent launches reuse configuration, keys, and data; database initialization can be run repeatedly. Development works the same on Windows, macOS, and Linux. If Docker Desktop is not running, it is started automatically; with Docker Engine on Linux, start the service yourself.

- The frontend defaults to `http://127.0.0.1:3000` and the API to `http://127.0.0.1:1234`. If a port is occupied, an available one is selected automatically; check the preparation task output for the actual addresses.
- The default administrator is `system`. Its initial password is stored in the `admin_password` field of `.local/dev/settings.json`; existing account passwords are not reset.
- `.local/dev/` is ignored by Git and contains development keys, API configuration, and debugger environment variables. The script updates managed fields such as connection addresses; other API settings can be adjusted in `api.config.toml`.
- **Shift+F5** stops both debuggers. Infrastructure keeps running for the next session. Use “Tasks: Run Task → Castor: Stop services” or `python3 scripts/dev.py stop` to stop containers while preserving data volumes.
- To prepare the environment only, run `python3 scripts/dev.py prepare`. The script supports Chinese, English, Japanese, and Korean; select a language with `CASTOR_DEV_LANG`.

Each repository path uses a separate Compose project and data volumes. Infrastructure ports bind only to localhost and are assigned automatically. Go debugging uses `CASTOR_CONFIG_FILE` to locate the generated configuration.

### Without VS Code

Requires Docker, Go 1.26+, Node.js 22+, Bun, and Python 3.11+.

```bash
python3 scripts/dev.py prepare     # Infrastructure, configuration, dependencies, init-db
python3 scripts/dev.py run api   # Terminal 1: Go API
python3 scripts/dev.py run web   # Terminal 2: Next.js
```

`dev.py run` starts each process with the ports and connection settings chosen by `prepare` (from `.local/dev/debug.env`); it needs no shell, so the same commands work on Windows, macOS, and Linux.

## Development guidelines

Development instructions live in AGENTS.md files, shared by human contributors and AI coding agents (Claude Code, Codex, Cursor, and others):

- [AGENTS.md](AGENTS.md): working principles, end-to-end module checklist, security invariants, API contracts, and commit conventions
- [apps/api/AGENTS.md](apps/api/AGENTS.md): backend layers, RBAC/menus, migrations, and testing conventions
- [apps/web/AGENTS.md](apps/web/AGENTS.md): frontend data layer, pages, permissions, i18n, and testing conventions
- [deploy/README.md](deploy/README.md): image builds, Compose / Kubernetes deployment, backup and restore
- [SECURITY.md](SECURITY.md): security policy

Run the unified checks before committing:

```bash
python3 scripts/check.py          # Or python3 scripts/check.py api | web | scripts
python3 scripts/check.py vuln     # Dependency vulnerability scans, before releases or periodically
```

After cloning, run `bun install` once at the repository root to enable Git hooks: pre-commit formats staged files, and pre-push builds the frontend (when it changed) and tests changed Go packages.

## Built-in features

- Multiple authentication methods (password / email / phone number)
- RBAC3 access control (multiple role inheritance, SSD/DSD, session role activation)
- File and asset management (S3-compatible storage)
- Audit logs
- Online sessions with forced sign-out, and a password policy (length, complexity, expiry)
- Department tree and role data scopes (all / own department and below / own department / selected departments / self)
- Dynamic system settings
- Data dictionaries
- Notifications
- Internationalization (Chinese / English / Japanese / Korean)

## License

[MIT](LICENSE). The frontend is adapted from [next-shadcn-dashboard-starter](https://github.com/Kiranism/next-shadcn-dashboard-starter) (MIT). See [NOTICE](NOTICE) for third-party notices.
