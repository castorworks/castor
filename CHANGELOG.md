# Changelog

**English** | [简体中文](CHANGELOG.zh-CN.md) | [日本語](CHANGELOG.ja.md) | [한국어](CHANGELOG.ko.md)

Notable changes per release, newest first. Versions are Git tags; build images with the same version (see [deploy/README.md](deploy/README.md)).

## v1.0.0 — 2026-09-24

First public release. Castor is an admin scaffold with a Go API and a Next.js dashboard; everything below ships in four languages (English, Simplified Chinese, Japanese, Korean).

- Accounts: password, email and SMS sign-in with captcha; a password policy (length, complexity, maximum age) with a change-expired-password flow; "remember me"; TOTP two-factor authentication with recovery codes; OpenID Connect single sign-on with account linking.
- Authorization: RBAC3 roles, resources, role hierarchy and separation-of-duty constraints; departments and role data scopes that limit which users' data each role can see; online sessions with forced sign-out.
- Administration: users with Excel/CSV export and bulk import, menus, dictionaries, system settings, audit logs and login histories (both exportable), and a dashboard.
- Files: a shared asset library with content sniffing, deduplication and reference tracking; business modules attach public or private files.
- Notifications: global or targeted notices with attachments, delivered live over Server-Sent Events, with optional email delivery through an outbox.
- Scheduled jobs registered in code, whose schedule operators can change, pause and run on demand.
- An OpenAPI 3.1 document generated from the route catalog, with an API docs page in the dashboard.
- Deployment: Docker Compose (single host, or shared infrastructure with one instance per tenant) and Kubernetes, with scripts for image builds, backups and restores.
