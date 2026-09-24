# AGENTS.md — Castor API

Go REST API (Gin + GORM + PostgreSQL + Redis + S3), DDD layering + Wire compile-time injection + RBAC3.
Repository-wide principles, verification commands, the cross-stack checklist, and security invariants are in [../../AGENTS.md](../../AGENTS.md); this file covers backend details only.

## 1. Commands

```bash
go run ./cmd/castor init-db        # Migrations + default dictionaries/resources/roles/menus/admin (advisory lock, safe to rerun)
go run ./cmd/castor                # Start the API (no migrations)
TZ=UTC go test -race -count=1 ./... # Tests (UTC like check.py and the production image; Windows ignores TZ, so keep tests zone independent)
go vet ./...
gofmt -w .                         # Format
cd cmd/castor && wire              # Regenerate wire_gen.go after changing Providers
python3 ../../scripts/check.py api  # Full backend verification
python3 ../../scripts/check.py vuln # Dependency vulnerability scan (govulncheck)
```

Config file path: the `CASTOR_CONFIG_FILE` environment variable, default `/etc/castor/config.toml` (where Compose/Kubernetes mount it; kept outside `/app` so the mount cannot shadow the image's `configs/i18n`). The only template is `deploy/config/api.config.example.toml`; for local development `scripts/dev.py prepare` generates `.local/dev/api.config.toml` from it.

## 2. Layout and layers

Dependency direction: `interfaces → application → domain ← infrastructure`, enforced by `internal/architecture_test.go`: domain must not import gorm/gin/redis or upper-layer packages; application must not import infrastructure, interfaces, gorm, gin, gin-jwt; persistence must not import application.

```
cmd/castor/                  main.go (serve / init-db), wire.go, wire_gen.go (generated), providers.go (config → application-layer policies)
configs/i18n/                {zh,en,ja,ko}.toml — backend message and seed-data translations
internal/
├── domain/{module}/         Entities + Repository interfaces + default seeds (var Default…); depends only on the standard library, domain/shared, pkg/query
│   └── shared/              BaseModel, ErrNotFound, I18nText (four-language text for operator-editable data)
├── application/
│   ├── apperror/            Business sentinel errors (single place of definition)
│   ├── dto/                 Request/response DTOs (binding tags, FromEntity)
│   └── service/             Service interfaces + implementations; interface exported in the same file
├── infrastructure/
│   ├── persistence/         Repository implementations, PaginatedQuery, translateError
│   │   └── models/          GORM Models (tags, TableName, hooks, ToEntity/FromEntity)
│   ├── database/            Connection, migration.go (versioned migrations), dictionary_seed.go (dictionary reconciliation)
│   ├── bootstrap/           init-db orchestration, menu_tree.go (desired menu tree), menu.go (menu reconciliation)
│   ├── authorization/       Default resource/role reconciliation
│   ├── config/              Config structs, defaults, env-var overrides, validation
│   ├── cache/ storage/      Redis, S3
│   ├── external/            SMS, mail, image captcha, verification-code storage
│   └── i18n/                Notification template rendering
├── interfaces/api/
│   ├── handler/             Gin handlers; common.go provides generic pagination/filter/CRUD helpers
│   ├── middleware/          auth(JWT), csrf, admin(RBAC), ratelimit, bodylimit, i18n, log, requestid
│   ├── response/            Response helpers, error codes, i18n keys, error mapping
│   ├── server/              Gin engine, security headers, lifecycle
│   └── router.go            All routes (AdminHandlers / UserHandlers)
└── pkg/                     Shared packages: query (filter/sort), ratelimit, rediskey (per-instance Redis key namespace), etc.
```

Existing modules: `user`, `department`, `asset`, `audit_log`, `login_history`, `permission` (resources/roles/constraints/authorization sessions/data scopes; the online-sessions page is `SessionService` over the same `rbac_sessions`), `menu`, `notification`, `dictionary`, `setting`; `DashboardService` only aggregates the others (monthly counts via `countByMonth` in `persistence/stats.go`, month boundaries in the database session time zone).

**Reference templates**: for paginated + filtered lists copy `audit_log` (one file each for handler/service/repository, simplest structure); for create/update/delete copy `user` or `notification`.
**Do not copy**: `dictionary` and `setting` lists are not paginated.

## 3. Writing each layer

### Domain

```go
// internal/domain/audit_log/repository.go
type Repository interface {
    Gets(ctx context.Context, page, size int, order string, opts ...query.Option) ([]AuditLog, int64, error)
    Get(ctx context.Context, id uint) (*AuditLog, error)
    Create(ctx context.Context, item *AuditLog) error
}
```

- Entities carry only `json` tags; sensitive fields use `json:"-"` (which also automatically excludes them from the filter/sort whitelist).
- Audit fields `ID/CreatedAt/CreatedBy/UpdatedAt/UpdatedBy` may be inlined or come from embedding `shared.BaseModel`.
- "Not found" is always `shared.ErrNotFound`; domain must not import gorm, gin, redis.

### Persistence

- Models go in `persistence/models/models.go` (large modules may use a separate file) and provide `TableName()`, `ToEntity()`, `XxxModelFromEntity()`; when audit users are needed, implement `BeforeCreate/BeforeUpdate` calling `ExtractUserIDFromContext(tx)`.
- Lists use `PaginatedQuery[models.XxxModel](ctx, r.db, page, size, order, opts...)`, which re-validates `order` against the model's column whitelist.
- Wrap single-record queries in `translateError(...)`, which converts `gorm.ErrRecordNotFound` to `shared.ErrNotFound`.
- Always `r.db.WithContext(ctx)`; never concatenate client strings into `Where`/`Order`/`Raw`.

### Migration

The `migrations` slice in `internal/infrastructure/database/migration.go`; applied versions are recorded in the `schema_migrations` table:

```go
{Version: N, Name: "create_widgets", Up: func(db *gorm.DB) error {   // N = current max version + 1
    return db.Migrator().CreateTable(&models.WidgetModel{})
}},
```

- Only append new versions and set `CurrentSchemaVersion` to N; **never modify existing versions**, and never AutoMigrate at startup.
- Migrations run only in `init-db` (serialized by advisory lock); normal startup does not migrate. During rolling releases a new version must stay compatible with the old image still running; destructive changes require a maintenance window, a verified backup, and a separately reviewed data migration. No automatic downgrade: rollback = switch back to an old image compatible with the current database.
- Use a disposable PostgreSQL (`CASTOR_TEST_POSTGRES_DSN`) to verify both an empty database and a database at the previous version.

### Application

- DTOs: `XxxPostReq` (`binding:"required,..."`), `XxxPutReq` (pointer fields mean optional), `XxxResp` + `FromEntity(e) error`.
- Services: exported interface `XxxService` + private implementation + `NewXxxService(deps...) XxxService`.
- Errors: define `ErrXxx = errors.New("...")` in `apperror/errors.go`; check not-found with `errors.Is(err, shared.ErrNotFound)`; do not import gorm.
- When account security state (password, enabled, locked, expired) changes, call `RBACService.RevokeUserSessions`.
- **Acting on another user's account** (password, status, expiry, deletion, revoking their sessions) must pass `RBACService.EnsureCanManageUser(callerID, targetID)`: the target's authorized permissions must be a subset of the caller's, otherwise a delegated user manager could take over a more privileged account. It ignores account state (a locked admin can still be unlocked by an equal admin). With no caller in the context the operation is rejected, never skipped.
- **Data scope** (departments + role data scopes): list and single-record methods of admin services that expose users' data take `scope permission.AccessScope` as the first argument after `ctx`; handlers get it with `requestScope(c, rbac)` (`handler/scope.go`, session-based: the active roles of the RBAC3 session). Lists and aggregates (including the dashboard's monthly counts, whose repository methods accept extra `query.Option`s) append `query.UserScope(column, …)` via `withUserScope` (`"id"` for `users`, the owning user column such as `user_id` / `operator_id` elsewhere); single records outside the scope return `shared.ErrNotFound` so nothing reveals they exist. A new module whose rows belong to users adopts the same two steps; data that is shared across the organization (asset library, dictionaries, settings) stays unscoped. Departments are a tree without materialized paths (`department.Subtree` expands it in memory); roles' `DataScope` constants map to the `role_data_scope` dictionary. Guards compare scopes with `AccessScope.Within` on the *authorized* roles, like the permission guards.
- **Passwords**: every entry point that stores a password a person chose (register, admin create/update, self-service change, code reset, expired-password change) validates it with `SettingHelper.PasswordPolicy(ctx).Validate` (settings `security.password.minLength` / `requireComplexity`, both public so pages can pre-check) and sets `CredentialExpireDate = policy.CredentialExpiry(now)` (`security.password.maxAgeDays`, 0 = never). An expired credential blocks sign-in with `ErrCredentialExpired` (errorCode 2009); the user replaces it with `PUT /api/v1/auth/password/expired`, which shares the password login's attempt counter and captcha switch.
- **One-time verification codes are isolated by purpose**: `service/verification_code.go` defines `VerificationPurpose{Auth,Reset,Bind}`, and the storage key is (purpose, target). A new purpose must use a new constant; never reuse an existing purpose, or a code from one flow can be replayed into another (a login code resetting a password, etc.). Purposes the client may choose are whitelisted by `NormalizeVerificationPurpose`; `bind` can only be specified by the server-side binding flow.
- **Accounts are recoverable**: a user's `Email`/`Mobile` may be empty and, when non-empty, is globally unique (partial unique index created by the initial schema). The `username` of `PUT /api/v1/account/password/reset` is an account identifier, resolved as "verified email → verified mobile → username"; any resolution failure returns `ErrInvalidConfirmCode` and must not reveal whether the account exists. Self-service contact binding goes through `POST /api/v1/account/contact/code` + `PUT /api/v1/account/contact` (`privateAccount` group, no `DefaultResources` registration needed); contact details an admin fills in via user CRUD are marked verified directly.
- **Do not read global config**: when configuration is needed, define a policy struct in `service/policy.go` (e.g. `RuntimePolicy`, `AssetPolicy`, `RsaKeyConfig`) as a constructor parameter, build it from `config.C` in `cmd/castor/providers.go`, and add it to `wire.Build`; tests pass policy values directly and do not modify global state.
- **Do not depend on HTTP**: service methods take `context.Context` and plain parameters; request parsing, client IP/UA extraction, and response writing stay in handlers/middleware (see `LoginService.Login(ctx, method, req, LoginClient)`, `AssetService.WriteContent(ctx, key, io.Writer)`). Log with `log.ErrCtx/WarnCtx(ctx)` (a passed-in gin.Context automatically carries the request ID; the engine has `ContextWithFallback` enabled).
- Async goroutines must not hold the request context; use `context.Background()` + a timeout.
- **Two-factor (TOTP)** (`service/mfa.go`, `domain/mfa`): setup stores an encrypted, inactive secret (replacing any unconfirmed one) and returns a QR code; enabling needs a valid code and returns 10 recovery codes (shown once, SHA-256 stored). `verify` accepts a 6-digit code (±1 step) whose step is newer than `LastUsedStep` (conditional `UPDATE` = replay protection across replicas) or consumes a recovery code (row lock). Login: the JWT Authenticator calls `LoginService.Login` (which records only failures), then if TOTP is on stores an `MFALoginChallenge` in Redis and fails with `ErrTOTPRequired`; `Unauthorized` puts the challenge in `data`. `JwtMiddleware.LoginWithTOTP` completes it and calls `IssueSession`; success is recorded with `LoginService.RecordLogin` only after a session exists. Admin reset (`DELETE /admin/users/:id/totp`) follows `EnsureCanManageUser`.
- **Single sign-on (OIDC)** (`service/sso.go`, `domain/sso`, `handler/sso.go`): providers are rows (`oidc_providers`, four-language name, encrypted client secret) managed at `/admin/oidc-providers`; enabling needs `General.PublicURL` and a successful discovery. `Begin` stores state/nonce/PKCE verifier in Redis (10 min) and returns the authorization URL; `Complete` `GETDEL`s the state, exchanges the code, verifies the ID token with go-oidc and matches `user_identities (provider_id, subject)`. Link mode attaches the subject to the user that started it; login mode signs in the linked user, auto-registers (`UserService.CreateFederated`, account source `OIDC`, no password) when the provider allows it, and otherwise fails with `ErrOIDCAccountNotLinked` — never match by email. The callback handler uses `SessionStarter.StartSession` (cookies, no body) and redirects; accounts with TOTP get a challenge instead. `internal/pkg/oidctest` is an in-process OIDC provider for tests. Discovery documents are cached per provider for an hour.
- **Notification delivery** (`service/notification_stream.go`, `service/notification_email.go`): `NotificationService` publishes a `NotificationEvent` after every write (new → recipients or everyone for global; read/dismiss → that user; admin edit/delete → everyone). `NotificationStream` fans events out through one Redis pub/sub channel per instance namespace (`notifications:events`) to local SSE connections; `Server` starts it with the job scheduler and stops it **before** HTTP shutdown so streams do not hold shutdown open. The SSE handler lifts the write deadline with `http.ResponseController`, sends `Cache-Control: no-cache, no-transform` (keeps proxies, including Next.js, from buffering) and ends at token expiry or session revocation. Emails use an outbox: `notification_emails` rows are enqueued (`INSERT … SELECT`, idempotent per notification+user) for eligible recipients when a notification with `SendEmail` is created or edited; the `deliverNotificationEmails` job (registered only when `Mail.Host` is set) sends due rows with `html/template`, retries with backoff (1m, 5m, 30m, 2h; 5 attempts) and skips rows whose recipient muted emails or whose notification expired. Email strings (`NotificationEmail*`) render in the server default language via `TemplateRenderer.Text`; links become absolute with `General.PublicURL`.
- **Export / import** (`pkg/tabular`, `handler/export.go`, `handler/user_io.go`, `service/user_import.go`): `pkg/tabular` reads and writes one header row plus string rows as `.xlsx` (excelize) or UTF-8 `.csv` with a BOM. It applies the formula-injection guard, keeps file line numbers, and enforces the row and unzip limits. To make a list exportable, add a handler method that calls `exportList(c, modelType, name, columns, fetch)` with the **same** `modelType` and scoped service `Gets` as the list. Each `exportColumn` pairs a header i18n key (`Column*` in the toml files) with a value func; use `exportContext` for `time()` (request `tz`), `label()` (dictionary label in the request language), `bool()` and `result()`. Then register `GET …/export` as a resource and audit it. Imports follow the user import: the handler maps the header (field key in parentheses, bare key, or the localized column name) and the service validates every row into `[]UserImportError{Line, Field, Code}`. `Code` is a toml key that the handler localizes; nothing is created if any row fails. `response.Message(c, key)` and `response.Language(c)` give header text and the negotiated language (the `LanguageCode` toml entry). Enum values written to the database must equal their dictionary item values exactly; `dictionary.TestAccountSourcesHaveDictItems` and `TestLoginMethodsHaveDictItems` guard the account source and login method.
- **Scheduled jobs** (`service/job.go`): jobs are registered in code in `NewJobCatalog` as `JobDefinition{Key, DefaultCron, DefaultEnabled, Timeout, Run func(ctx) (affected int64, err error)}`. The database (`scheduled_jobs`) only stores what operators may change: the cron expression and whether it runs on schedule. It never stores a function name to call. On start, `JobService.Start` adds missing settings and never overwrites existing ones. The loop wakes at every whole minute in the process's local time zone; missed minutes are skipped, not caught up. Cron is the standard 5-field form only: no descriptors, no `TZ=`, and never-firing expressions return `ErrInvalidCron`. Two namespaced Redis locks make it safe with several replicas. `job:tick:<key>:<unix>` makes one replica run each scheduled minute. `job:run:<key>`, with TTL = timeout + 1 min, stops overlapping runs; a manual run during a run returns `ErrJobAlreadyRunning` (409). Runs are recorded in `job_runs` (keeps the latest 100 per job). A `RUNNING` row whose lock nobody holds is closed as failed ("interrupted"). `Server` stops the scheduler before closing the database, cancelling runs and waiting for them to record their result. To add a job: append a definition, give `Run` a context-aware implementation that returns the number of records processed, and add `jobs.catalog.{key}.name/description` to the four frontend message files. Keys are lower camel case so they work as translation keys.
- **Redis keys must carry the instance namespace**: multiple deployments can share one Redis, isolated by the `General.InstanceID` prefix. Components that need Redis receive `rediskey.Namespace` as a constructor parameter (wire provides it via `cache.NewNamespace`), and every key is always `ns.Key("…")`, never a bare concatenated string; third-party stores use their prefix option (e.g. captcha's `WithKeyPrefix(ns.Key(captcha.DefaultKeyPrefix))`). `TestRedisKeysNamespaced` in `architecture_test.go` requires every file importing go-redis to also import `rediskey`; files that only probe connectivity or close connections are registered in `redisWithoutKeys`. New components assert in tests that keys fall under the namespace (see `token_test.go`).

### Handlers and routes

```go
func (h *AdminAuditLogHandler) Gets(c *gin.Context) {
    GenericGets(c, reflect.TypeOf(audit_log.AuditLog{}), func(ctx *gin.Context, page, size int, order string, opts ...query.Option) (interface{}, int64, error) {
        return h.AuditLogService.Gets(ctx, page, size, order, opts...)
    })
}
```

- `handler/common.go`: `GenericGets / GenericGet / GenericPost / GenericPut / GenericPatch / GenericDelete / GenericBatchDelete`, `ParsePageParams`, `ValidateOrderParam`, `parseOrderParam` (sort validation for non-generic lists).
- Respond only with `response.Success / SuccessListPaged / SuccessI18n / BadRequestErr / BadRequestI18n / HandleError`; all errors go through `response.HandleError(c, err)`, which adds the business `errorCode` of the mapping to the body (outlets that bypass it, like the JWT `Unauthorized`, use `response.BusinessErrorCode(msgKey)`).
- Routes: add a field to `AdminHandlers` (or `UserHandlers`) in `router.go` and register under `adminGroup.Group("/widgets")` in `With()`. Paths must be literals (`router_rbac_test.go` parses the source).
- Public routes must attach `middleware.RemoteIpPathRateLimit(r.rateLimiter, duration, count)`.
- **OpenAPI** (`interfaces/api/openapi/`, `interfaces/api/openapi_catalog.go`): every route needs one `route(auth, method, path, tag, summary, …)` entry in `apiOperations`. Options:
  - `body(dto.XReq{})`: JSON request body.
  - `form(...)`: multipart request.
  - `data(dto.XResp{})`: the success `data`.
  - `list(dto.XResp{}, model{})`: a paginated list. Pass the same model as `GenericGets`; it lists the allowed filter and sort fields.
  - `download(...)` / `export(model{})...`: file responses.
  - `raw(...)`: responses without the envelope.
  - `query(...)` and `describe("…")`: extra query parameters and a description.

  Responses built from a `gin.H` get a small named mirror type in the catalog file. The generator reads `json` and `binding` tags: `required`, `min`/`max`, `oneof`, `email`; pointer fields become nullable. Types with custom JSON encoding are rejected. `openapi_catalog_test.go` fails on a missing or extra route, a wrong auth level, or a stale `openapi.json`; regenerate it with `go test ./internal/interfaces/api -run TestOpenAPISpecIsUpToDate -update-openapi` and commit the diff. Summaries and descriptions are English (developer-facing).
- **Every mutating endpoint must write an audit log** (including "erase the trail" operations such as deleting audit logs or login history; guarded by `interfaces/api/router_audit_test.go`, whose `auditExemptRoutes` lists the few write routes that change no business data, each with a reason). Write it with `logAudit(c, svc, type, target, details, err)` from `handler/audit.go` (`logAuditAs` when there is no signed-in user: password reset, registration, OIDC callbacks). Once the request is parsed and validated — i.e. the operation is actually attempted — record **exactly one entry whatever the outcome**: run the operation (chain lookups, permission checks and the write into one `err`), call `logAudit` with that `err`, then answer. Denied attempts, conflicts and failures are what an audit trail is for. `logAudit` marks the entry failed and appends a sanitized reason: the text of the matching registered business error (`response.KnownError`), otherwise `internal error`. With `GenericPost/Put/Delete/BatchDelete`, audit inside the closure. `TestHandlerAuditCallsRecordOutcome` rejects a literal `nil` outcome (success-only auditing) and direct `LogAsync` calls in handlers; exports pass an audit callback to `exportList`, so an oversized or failed export is recorded as well.
- Audits written in the service layer (no `gin.Context`, e.g. an asset promoted to public by `Attach`) take the caller from `ucontext.ActorFromContext(ctx)`: `middleware.ActorContext()` runs after the JWT middleware on every authenticated group and puts user ID, username and client IP on the request context, so a plain `c.Request.Context()` passed down still identifies the operator.
- Add new types to `domain/audit_log/entity.go` and add an item to the `audit_log_type` dictionary in `domain/dictionary/entity.go` (labels are four-language `text(en, zh, ja, ko)`), otherwise `domain/dictionary/audit_log_type_parity_test.go` fails (this guard test requires a one-to-one match between constants and dictionary items). Each kind of operation gets its own type — do not file menu, resource or constraint changes under a neighbouring type; `TestAuditLogTypesInUse` fails for a type no code records. Details describe the attempted operation in the imperative (`Create role ops`), since the same text is used for failed attempts. Rerun `init-db` to give existing databases the new dictionary item (see "Dictionaries" below).
- `Details` must not contain passwords, verification codes, or raw errors; **config values and dictionary item values are likewise forbidden** (they may be credentials or arbitrary payloads). Record only server-side identifiers such as config keys / dictionary type codes / object keys, and never append failure reasons yourself — `logAudit` does it safely.

### Dictionaries

Dictionaries are **the presentation layer for enum values**: the source of truth for values is Go constants (`asset.StatusActive`, `audit_log.AuditLogType*`), and validation and business branching use the constants; a dictionary only provides the value's four-language label, color, icon, sort order, and enabled state, which operators can adjust in "Data Dictionary" without a release. **Do not use the dictionary tables for validation.**

- **Registration**: append to `DefaultDictTypes` / `DefaultDictItems` in `domain/dictionary/entity.go`. Labels use `text(en, zh, ja, ko)`, all four languages required; `Color` may only take values from `dictionary.Colors` (matching the frontend `tag-*` tokens); `Icon` must be a key of the frontend `Icons`; seeds are always `IsSystem: true`. Types referenced by code must also be added to `DICT_TYPES` in the frontend `src/lib/dict.ts`. All of this is guarded by `seed_integrity_test.go`, `i18n_completeness_test.go`, and the frontend `lib/dict.test.ts`.
- **Reaching existing databases**: `database.SeedDictionary` runs after migrations on every `init-db`, creating missing rows by `code` / `(type_code, value)`, **add-only, never update**—labels, colors, icons, sort order, enabled state, and public visibility changed by operators are all preserved. New dictionary items need no migration. `SortOrder` takes effect only on first insert, so append new items to the end of their group rather than inserting in the middle (otherwise new and old databases end up in different orders). The only field forcibly written back is `is_system`: it is not operator-editable but the fact that "this item is referenced by code".
- **System item protection**: an `IsSystem` dictionary item's `value` cannot be changed and the item cannot be deleted (`ErrSystemDictItemValueLocked` / `ErrSystemDictItemDelete`), because `value` is its only link to the Go constant; system types cannot be deleted. Disabling is allowed.
- **Visibility**: `GET /api/v1/account/dictionaries` (login required) returns all enabled types; the unauthenticated `GET /api/v1/dictionaries[/:typeCode]` returns only `IsPublic` types, and non-public and nonexistent both return 404. `IsPublic` defaults to false; enable it only for types that pre-login pages truly need and that are unrelated to business modules.
- **Caching**: the three read endpoints share one Redis snapshot (`ns.Key(constant.DICT_SNAPSHOT_CACHE_KEY)`, TTL 5 minutes). Every write method of `DictionaryService` must call `invalidateSnapshot` (`TestDictionaryService_SnapshotCache` verifies each one); when Redis is unavailable it falls back to the database. `init-db` writes to the database directly, bypassing the service, and converges via the TTL.

### Files in business modules (assets)

The asset module is the file capability shared by all business modules, not just the admin "Asset Management" page. File content lives in object storage (the `asset.Storage` port; the S3 adapter is in `infrastructure/storage`) and metadata in the `assets` table; **business records store only the asset's `objectKey` (a string), never a URL**. Two reference implementations:

| | Avatar (`service/user_avatar.go` + `userService.Put/Delete`) | Notification attachments (`service/notification.go`) |
|---|---|---|
| Field | Single-valued, `users.avatar` stores the object key | Multi-valued, the attachment set **lives only in the reference table** (`NotificationAttachments`); the notification table has no attachment column |
| Visibility | Public (`RequirePublic`), displayed via the unauthenticated download endpoint | Private, via the notification module's own download route; only recipients can download |
| Upload entry | `POST /api/v1/account/avatar` (end users) | `POST /api/v1/admin/notifications/attachments` (notification's own RBAC resource) |

- **References**: business services inject `service.AssetReferencer` (implemented by `AssetService`); a reference is identified by `asset.Reference{OwnerType, OwnerID, Field}`, e.g. `{"user", 42, "avatar"}`.
  - Single-valued fields: `Attach(ctx, objectKey, ref, AttachOptions{...})`; after switching files, `Replace(ctx, ref, newKey)` releases the old reference on that field (empty `newKey` clears it).
  - Multi-valued fields: `Sync(ctx, ref, []asset.AttachedFile{{ObjectKey, Name}}, opts)` makes the field reference exactly the given set—it registers additions first, then releases removals; if any object key is invalid the whole call fails and existing references are untouched. Read with `ListAttached(ctx, asset.Field{OwnerType, Name}, ownerIDs)`, which fetches attachments for a page of records in one query; do not store a second list of object keys in the business table.
  - Each reference records its own display name (`AttachedFile.Name` / `AttachOptions.DisplayName`): deduplication lets multiple uploads share one asset record, and the asset's own name belongs to the earliest uploader, so attachment lists and download filenames must take the name from the reference, or later uploaders will see a filename someone else chose. Likewise, responses to end users use only `dto.AssetAttachmentResp`; never return the full `AssetResp`.
  - Deleting a business record: `DetachOwner(ctx, ownerType, ownerID)`.
  - The order is fixed as "`Attach` the new file first → save the business record → then `Replace`/`Detach` the old file"; if saving fails, `Detach` the new file to roll back (see `changeUserAvatar`). Whichever step fails, the old file is still there.
  - `AttachOptions.RequirePublic`: the business needs to display the file via an unauthenticated URL (avatar, cover). A private asset is promoted to public and audited; non-active or high-risk types are rejected outright. `RequireCategory` restricts the category (e.g. accept only `IMAGE`).
- **Referenced means protected**: a referenced asset cannot be deleted, set to non-`ACTIVE`, or made non-public; all of these return `apperror.ErrAssetInUse` (409). Beyond the service-layer check, the foreign key on `asset_references.asset_id` (`ON DELETE RESTRICT`) is the final safeguard under concurrency, so deletion is always "delete the record first, then the object", and the repository does not cascade-delete references when deleting an asset. Hash deduplication (identical content stored as one asset; the `assets.hash` partial unique index arbitrates concurrency) is safe precisely because of this—**business code must never delete assets directly**; it only releases its own references.
- **`Scope`**: `LIBRARY` is files uploaded and managed manually by operators in "Asset Management"; `ATTACHMENT` is attachments uploaded by business modules, reclaimed automatically when the last reference is released (storage object and record deleted together; reclaim failures are only logged, and leftovers can be filtered by scope in "Asset Management" and deleted manually). Business modules may also reference `LIBRARY` assets ("choose from asset library" in forms), which are never reclaimed.
- **Downloading private files**: the business module opens its own download route, first does its own authorization (whether this business record is visible to the current user), then calls `PrepareAttachedDownload(ctx, objectKey, ref)` + `WriteContent`. It confirms the file is actually attached to `ref`—skip either step and a user can read any file through a record they can see. Any failure returns an empty 404 (see `NotificationHandler.DownloadAttachment`). Admins editing a record download its saved files through an admin route of the module (`GET /api/v1/admin/notifications/:id/attachments/:objectKey`, `PrepareAdminAttachmentDownload`): the record must exist and the file must be attached to it, but the asset-library permission is not required. Object keys are unguessable, but they are not an authorization mechanism.
- **Opening an upload channel for a business module**: the business module opens its own route (its own authentication, rate limiting, RBAC—people who can send notifications need not have asset-library permissions); the handler calls `AssetService.UploadAttachment(ctx, AttachmentUpload{..., Policy: field policy})`, then `Attach`. A field policy (`AssetPolicy`: size, extensions, sniffed MIME) can only tighten within the global `[Asset]` config, never loosen it. The upload pipeline (not trusting client-declared type and size, content sniffing, forcing high-risk types private, deduplication) is the same for every entry point; never bypass it to write to object storage directly.
- **Orphan sweeping**: there is no transaction between upload and `Attach`—a form uploaded but never saved, a process crash between the two steps, or a storage error during reclaim all leave unreferenced `ATTACHMENT`s. The scheduled job `sweepOrphanAttachments` (hourly by default, adjustable on the "Scheduled Jobs" page) calls `AssetService.SweepOrphanAttachments`, reclaiming attachments with zero references whose `updated_at` is older than `[Asset].OrphanAttachmentTTLHours` (default 24; 0 disables it and the job is not registered); an attachment reused through deduplication has its `updated_at` refreshed, restarting the clock. Concurrent sweeping by multiple replicas is safe (idempotent deletes + foreign key protection). Therefore business code **does not need** cleanup logic for files that were "uploaded but never used".
- **File fields in admin forms**: `POST /api/v1/admin/assets` with `scope=ATTACHMENT` (built into the frontend `<FormAssetField>`); the file does not enter the asset library and is reclaimed along with its references. Conversely, a file an operator uploads into the asset library that deduplicates onto a business attachment is converted to `LIBRARY` and is not reclaimed with that business reference; `LIBRARY` is never downgraded.
- **Downloads**: the unauthenticated `GET /api/v1/assets/download/:objectKey` only allows `IsPublic` + `ACTIVE` + non-high-risk types; everything else is 404; the admin `GET /api/v1/admin/assets/download/:objectKey` is RBAC-protected. Both serve with `attachment` + `nosniff`; for unauthenticated downloads of business attachments the filename is only the object key (the original filename is end-user input and may contain personal information).
- **Contract**: asset lists use the unified filter contract (`category-eq`, `status-in`, `scope-eq`, `folder_path-like`); "Asset Management" lists only `scope-eq=LIBRARY` by default. The notification response's `attachments: [{ objectKey, name, size, sizeFormatted, mimeType, category }]` are private attachments that recipients download via `/api/v1/account/notifications/:id/attachments/:objectKey`; creating/updating a notification submits `attachments: [{ objectKey, name }]` (on update, omitted = unchanged, empty array = clear), with files first uploaded via `POST /api/v1/admin/notifications/attachments`.
- On a deduplication hit, `Upload` / `UploadAttachment` returns the **existing asset**, and this request's name and visibility do not take effect; if it must be public, declare `RequirePublic` at `Attach` time rather than relying on upload parameters.

### Error-code chain (a missing link fails the tests)

1. `application/apperror/errors.go`: `ErrWidgetNotFound`
2. `interfaces/api/response/messages.go`: i18n key constant + `msgToCode` entry
3. `interfaces/api/response/errcode.go`: `CodeWidgetNotFound = ErrCode{Code, HTTPStatus, MsgKey}` (numeric ranges in `response/errors.go`)
4. `interfaces/api/response/handle_error.go`: add `{apperror.ErrWidgetNotFound, CodeWidgetNotFound}` to `errorMappings`
5. `configs/i18n/{zh,en,ja,ko}.toml`: translations

### Wire

In `wire.Build` in `cmd/castor/wire.go`, add `persistence.NewXxxRepository`, `service.NewXxxService`, `handler.NewAdminXxxHandler` under the commented groups; `AdminHandlers/UserHandlers` are injected automatically via `wire.Struct(..., "*")`. Then `cd cmd/castor && wire`.

## 4. RBAC and menus

- **Authorization**: `middleware/admin.go` calls `RBACService.AuthorizeSession(ctx, sessionID, userID, c.FullPath(), method)`; the resource path must exactly match the Gin route template (e.g. `/api/v1/admin/widgets/:id`).
- **Resource registration**: declare every protected route in `DefaultResources` in `internal/domain/permission/entity.go`:

  ```go
  {Code: "admin:widgets:list", Name: "seedResources.admin.widgets.list", Path: "/api/v1/admin/widgets",
   Actions: StringSlice{ActionGET}, Category: CategoryAdmin, Module: "widgets", SortOrder: 300, IsSystem: true, IsEnabled: true},
  ```

  Add the translation of `Name` to both `configs/i18n/*.toml` and `apps/web/messages/*.json` (`seedResources.*`); the frontend also needs `resources.modules.widgets`.
- **Default roles** (`infrastructure/authorization/defaults.go`, reconciled by init-db): `admin` has all system resources; `auditor` has admin-category GETs; `user` has user-category resources.
- **Menus**: the desired system navigation tree is declared centrally in `systemMenuTree` in `infrastructure/bootstrap/menu_tree.go`, and `initializeMenus` in `menu.go` reconciles it into the database on every init-db.
  - Append nodes to the `systemMenuTree` slice (parents must come before children):
    `node("<parent directory code>", menu.Menu{Code: "page_widgets", Kind: menu.Page, Titles: menu.Titles{En, Zh, Ja, Ko}, Path: "/dashboard/widgets", Icon: <key of Icons>, SortOrder: 10, AccessMode: "permission", Permissions: permissionOf("/api/v1/admin/widgets")})`
  - Add `"widgets": "widgets"` to the `menuModules` map, register the list permission the page itself expresses in `menuPageResources`; button-level action node titles come from `defaultActionTitles`.
  - Reconciliation is **add-only: never update, never delete**: `createMissingMenus` creates missing nodes by `Code` (creating the parent directory too if it is missing); existing nodes (title, sort order, icon, enabled state, parent) are always preserved, so operators' changes in "Menu management" are never overwritten. Therefore, after a new module ships, rerunning `init-db` on an installed database is enough to get the navigation entry, with no manual additions or migrations.

## 5. Configuration

- Structure and defaults: `infrastructure/config/config.go` (`default:"..."` tags, filled by creasty/defaults).
- Load order: `Load → OverrideFromEnv → Validate` (`config/load.go`).
- Environment-variable overrides are a **fixed list**: `CASTOR_INSTANCE_ID`, `CASTOR_JWT_KEY`, `CASTOR_RSA_SECRET`, `CASTOR_DATA_KEY` (at least 32 bytes; encrypts TOTP and OIDC secrets in the database, so back it up with the database), `CASTOR_PUBLIC_URL` (the site origin, needed for OIDC callbacks), `CASTOR_DB_{HOST,PORT,USER,PASSWORD,NAME}`, `CASTOR_REDIS_{ADDR,PASSWORD}`, `CASTOR_S3_{ENDPOINT,ACCESS_KEY,SECRET_KEY,BUCKET}`, `CASTOR_MAIL_{HOST,PORT,USERNAME,PASSWORD,FROM,TLS_POLICY}`, `CASTOR_SMS_*`, `CASTOR_CORS_ORIGINS`; plus `CASTOR_CONFIG_FILE` and `CASTOR_DEFAULT_ADMIN_PASSWORD` (≥12 characters, required for init-db outside development mode).
- Always validated: `InstanceID` is 1-32 lowercase letters/digits/`-` (it goes into Redis keys and JWTs; `:` and `{}` are not allowed), JwtKey ≥32 bytes, `JwtTimeoutHours ≤ JwtMaxRefreshHours`, required connection settings; outside development mode additionally the RSA key must be exactly 32 bytes, placeholder secrets are rejected, and CORS must not allow `*`.
- `Postgres` / `Redis` / `S3` / `Mail` / `AliyunSms` embed gosuite's config structs directly; **gosuite validates them when creating clients, and the service will not start if validation fails** (e.g. sentinel mode missing `MasterName`, a misspelled `Mail.TLSPolicy`). Keys in TOML that do not exist on the struct are silently ignored, so key names in the template must match field names—`TestConfigTemplateSatisfiesGosuiteValidation` in `config_test.go` checks that the template passes validation and contains no known dead keys.
- **SMS and mail are optional channels**: when both the `AliyunSms` AccessKey and SecretKey are empty, or `Mail.Host` is empty, `external.NewAliyunSmsClient` / `NewMailClient` return nil and the service starts normally; sending a verification code to that channel returns `apperror.ErrDeliveryChannelNotConfigured` (503, error code 5003). Setting only one SMS key is still rejected at startup as a config error. A `Mail.Host` in a documentation domain (`example.com/.net/.org`, `.example`, `.invalid`) is rejected in every mode: it can never deliver, yet it would switch the email channel on (the template leaves `Host` empty and `dev.py prepare` clears the old placeholder). Development mode skips SMS sending anyway and needs no configuration.
- Default mail encryption policy (`Mail.TLSPolicy` empty): implicit TLS on port 465, **mandatory** STARTTLS on other ports; relays on internal networks that do not support TLS must explicitly set `"none"`. When `Mail.From` is empty, `Username` is used as the sender.
- **Instance identifier** `General.InstanceID` (default `castor`): the prefix of this instance's Redis keys and also the JWT `aud`. At startup `cache.NewRedis` uses `ns.Key("instance")` to register "which deployment this namespace belongs to"; the value is the random identifier in the database's `deployment` table (generated by `init-db`, unchanged by host/port changes or whole-database migration); if another deployment has already registered it, startup is refused. So `init-db` must have been run against the database before the API starts. Once the old deployment is confirmed offline, delete that key to let the new deployment take over.
- The S3 client does not connect to the network when created; `storage.NewS3` explicitly calls `EnsureBucket` at startup (15-second timeout), preserving "fail to start if storage is unavailable".
- **Upgrading gosuite**: unit tests go neither through gosuite's database client (the other integration tests open gorm directly from the DSN) nor through real object storage. So run `check.py api` once with `CASTOR_TEST_POSTGRES_DSN` (`TestNewPostgresConnectsWithRegionalTimeZone` uses the same connection path as production), then smoke-test against the dev RustFS: `EnsureBucket`, upload (including `size=-1` streaming multipart), read, list, delete, presign, and start the API. gosuite's S3 client does not auto-detect `Region`; for AWS S3 you must set the bucket's actual region.
- New config items: update both `config.go` and `deploy/config/api.config.example.toml`, and the dev config generated by `scripts/dev.py` when needed.

## 6. Testing conventions

- Standard library `testing`, table-driven + `t.Run`; **hand-written mocks** (`service/test_helpers_test.go`, `handler/handler_test.go`), no mock frameworks.
- Redis uses `miniredis`; property tests use `testing/quick`.
- Call `gin.SetMode` only in `TestMain` (see `service/main_test.go`, `middleware/main_test.go`); tests that modify `config.C`, call `t.Setenv`, or call `gin.SetMode` **must not** use `t.Parallel()`. Application-layer tests inject configuration via policy structs and need not touch `config.C`.
- Real PostgreSQL tests read `CASTOR_TEST_POSTGRES_DSN` and `t.Skip` when it is unset.
- When fixing a bug, write a reproducing test first; security fixes must include regression tests (see `handler/order_security_test.go`, `middleware/auth_refresh_test.go`).

## 7. Code style

- Package names are lowercase without underscores (the existing `audit_log` and `login_history` are historical exceptions; do not add more).
- Wrap errors with `fmt.Errorf("context: %w", err)`; ignoring an error requires a reason, and cleanup failures must at least be logged.
- Import groups: standard library / third-party / this project.
- Log with `github.com/hyperits/gosuite/logger`; never log passwords, tokens, or verification codes.
- Write code comments in Chinese, consistent with existing code; exported symbols get godoc.
