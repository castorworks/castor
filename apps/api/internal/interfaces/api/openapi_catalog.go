package api

import (
	"net/http"
	"sync"
	"time"

	"github.com/castorworks/castor/internal/application/dto"
	"github.com/castorworks/castor/internal/application/service"
	"github.com/castorworks/castor/internal/domain/asset"
	"github.com/castorworks/castor/internal/domain/audit_log"
	"github.com/castorworks/castor/internal/domain/department"
	"github.com/castorworks/castor/internal/domain/job"
	"github.com/castorworks/castor/internal/domain/login_history"
	"github.com/castorworks/castor/internal/domain/notification"
	"github.com/castorworks/castor/internal/domain/permission"
	"github.com/castorworks/castor/internal/domain/user"
	"github.com/castorworks/castor/internal/interfaces/api/handler"
	"github.com/castorworks/castor/internal/interfaces/api/middleware"
	"github.com/castorworks/castor/internal/interfaces/api/openapi"
	"github.com/castorworks/castor/internal/interfaces/api/response"
	"github.com/castorworks/castor/internal/pkg/constant"
	"github.com/gin-gonic/gin"
)

// 路由目录：每个在 router.go 注册的路由都要在 apiOperations 里登记一条（openapi_catalog_test.go 把关），
// 写明请求体与响应 data 的 Go 类型。handler 里用 gin.H 拼出的响应，在下面用同形的类型描述。

// EmptyObject 响应 data 为空对象 {}
type EmptyObject struct{}

// PublicKeyData GET /auth/public-key
type PublicKeyData struct {
	PublicKey string `json:"publicKey"`
}

// LoginUser 登录响应里的用户与授权快照
type LoginUser struct {
	ID                     uint                             `json:"id"`
	Username               string                           `json:"username"`
	Name                   string                           `json:"name"`
	Avatar                 string                           `json:"avatar"`
	AssignedRoles          []permission.Role                `json:"assignedRoles"`
	AuthorizedRoles        []permission.Role                `json:"authorizedRoles"`
	ActiveRoles            []permission.Role                `json:"activeRoles"`
	Permissions            []permission.EffectivePermission `json:"permissions"`
	AuthorizationSessionID string                           `json:"authorizationSessionId"`
}

// LoginData POST /auth/login：令牌只在 httpOnly 的 jwt cookie 里，响应体不含令牌
type LoginData struct {
	Expire time.Time `json:"expire"`
	User   LoginUser `json:"user"`
}

// ExpireData POST /auth/refresh-token
type ExpireData struct {
	Expire time.Time `json:"expire"`
}

// CodeSentData POST /auth/code
type CodeSentData struct {
	Res string `json:"res"`
}

// DeletedData 删除结果
type DeletedData struct {
	Deleted bool `json:"deleted"`
}

// RoleDeletedData DELETE /admin/roles/:role
type RoleDeletedData struct {
	Deleted bool   `json:"deleted"`
	Role    string `json:"role"`
}

// UpdatedData 更新结果
type UpdatedData struct {
	Updated bool `json:"updated"`
}

// AddedData 添加结果
type AddedData struct {
	Added bool `json:"added"`
}

// RolePermissionsData GET /admin/roles/:role/permissions
type RolePermissionsData struct {
	Grants    []service.RolePermissionDetail `json:"grants"`
	Menus     []dto.MenuResp                 `json:"menus"`
	Resources []dto.ResourceResp             `json:"resources"`
}

// ActiveRolesRequest PUT /account/session/roles
type ActiveRolesRequest struct {
	RoleCodes []string `json:"roleCodes"`
}

// RoleHierarchyRequest PUT /admin/roles/:role/hierarchy
type RoleHierarchyRequest struct {
	JuniorRoleCodes []string `json:"juniorRoleCodes"`
}

// ProbeStatus /health 与 /ready（不套响应信封）
type ProbeStatus struct {
	Status string `json:"status"`
}

type opOption func(*openapi.Operation)

func route(auth openapi.Auth, method, path, tag, summary string, options ...opOption) openapi.Operation {
	op := openapi.Operation{Method: method, Path: path, Tag: tag, Summary: summary, Auth: auth}
	for _, option := range options {
		option(&op)
	}
	return op
}

func body(v any) opOption        { return func(op *openapi.Operation) { op.Body = v } }
func data(v any) opOption        { return func(op *openapi.Operation) { op.Data = v } }
func raw(v any) opOption         { return func(op *openapi.Operation) { op.Raw = v } }
func redirect(s string) opOption { return func(op *openapi.Operation) { op.Redirect = s } }
func stream(s string) opOption   { return func(op *openapi.Operation) { op.Stream = s } }
func describe(s string) opOption { return func(op *openapi.Operation) { op.Description = s } }
func list(item, filter any) opOption {
	return func(op *openapi.Operation) { op.List, op.Filter = item, filter }
}
func query(params ...openapi.Param) opOption {
	return func(op *openapi.Operation) { op.Query = append(op.Query, params...) }
}
func form(fields ...openapi.FormField) opOption {
	return func(op *openapi.Operation) { op.Form = fields }
}
func download(mediaTypes ...string) opOption {
	if len(mediaTypes) == 0 {
		mediaTypes = []string{"application/octet-stream"}
	}
	return func(op *openapi.Operation) { op.File = mediaTypes }
}

// export 导出接口：与列表同样的筛选、搜索与排序参数，外加格式与时区
func export(filter any) []opOption {
	return []opOption{
		download("application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", "text/csv"),
		describe("Takes the same filter, search and `order` parameters as the list (no paging) and applies the caller's data scope. More than 10,000 matching rows is rejected (`ErrExportTooLarge`). Headers and dictionary-backed cells follow `Accept-Language`."),
		query(formatParam, openapi.Param{Name: "tz", Description: "IANA time zone for time cells; defaults to the API's zone."}),
		func(op *openapi.Operation) { op.Filter = filter },
	}
}

var formatParam = openapi.Param{Name: "format", Enum: []string{"xlsx", "csv"}, Description: "Defaults to xlsx."}

const (
	opGet    = http.MethodGet
	opPost   = http.MethodPost
	opPut    = http.MethodPut
	opDelete = http.MethodDelete
)

const (
	pub     = openapi.Public
	session = openapi.Session
	admin   = openapi.Admin
)

// apiOperations 全部路由的文档
var apiOperations = func() []openapi.Operation {
	ops := []openapi.Operation{
		// System
		route(pub, opGet, "/health", "System", "Liveness probe", raw(ProbeStatus{})),
		route(pub, opGet, "/ready", "System", "Readiness probe (database and Redis)", raw(ProbeStatus{}),
			describe("Answers 503 with `{\"status\":\"not ready\"}` when a dependency is down.")),
		route(admin, opGet, "/api/v1/admin/oidc-providers", "Single sign-on", "List identity providers", data([]dto.OIDCProviderResp{})),
		route(admin, opPost, "/api/v1/admin/oidc-providers", "Single sign-on", "Create an identity provider", body(dto.OIDCProviderReq{}), data(dto.OIDCProviderResp{}),
			describe("Enabling a provider requires `General.PublicURL` and a reachable issuer (discovery is checked). The client secret is stored encrypted and never returned.")),
		route(admin, opPut, "/api/v1/admin/oidc-providers/:id", "Single sign-on", "Update an identity provider", body(dto.OIDCProviderReq{}), data(dto.OIDCProviderResp{}),
			describe("Leave `clientSecret` empty to keep the stored one.")),
		route(admin, opDelete, "/api/v1/admin/oidc-providers/:id", "Single sign-on", "Delete an identity provider",
			describe("Also removes every account link to it.")),
		route(admin, opGet, "/api/v1/admin/openapi.json", "System", "This OpenAPI document", raw(map[string]any{})),

		// Auth
		route(pub, opGet, "/api/v1/auth/captcha", "Auth", "Get a captcha image", data(dto.CaptchaResp{})),
		route(pub, opGet, "/api/v1/auth/public-key", "Auth", "Get the RSA public key for encrypting passwords", data(PublicKeyData{}),
			describe("Passwords and other credentials are sent RSA-OAEP (SHA-256) encrypted with this key, base64-encoded.")),
		route(pub, opPost, "/api/v1/auth/login", "Auth", "Sign in", body(middleware.LoginRequest{}), data(LoginData{}),
			query(openapi.Param{Name: "method", Enum: []string{constant.LOGIN_METHOD_PASSWORD, constant.LOGIN_METHOD_EMAIL, constant.LOGIN_METHOD_MOBILE}, Description: "`credential` is the RSA-encrypted password, or the verification code."}),
			describe("Sets the httpOnly `jwt` cookie and the `csrf_token` cookie. `errorCode` 2002 asks for a captcha, 2009 means the password expired (replace it with `PUT /api/v1/auth/password/expired`).")),
		route(pub, opPost, "/api/v1/auth/login/totp", "Auth", "Complete a two-factor sign-in", body(dto.TOTPLoginReq{}), data(LoginData{}),
			describe("When the account has two-factor authentication on, `POST /api/v1/auth/login` answers `errorCode` 2012 with `data: {challenge, expiresIn}` instead of a session. Send that challenge with a 6-digit authenticator code or a recovery code; 5 attempts per challenge, valid for 5 minutes.")),
		route(pub, opGet, "/api/v1/auth/oidc/providers", "Auth", "List single sign-on providers", data([]dto.OIDCProviderPublicResp{}),
			describe("Enabled identity providers for the sign-in page (empty until `General.PublicURL` is configured).")),
		route(pub, opGet, "/api/v1/auth/oidc/:provider/authorize", "Auth", "Sign in with an identity provider",
			query(openapi.Param{Name: "redirect", Description: "Site path to return to (relative, e.g. `/dashboard/overview`)."},
				openapi.Param{Name: "remember", Type: "boolean", Description: "Keep the session across browser restarts."}),
			redirect("To the provider's authorization endpoint (authorization code flow with PKCE, state and nonce), or back to `/auth/sign-in?oidcError=…` when the provider is unavailable."),
			describe("A browser navigation, not an XHR.")),
		route(pub, opGet, "/api/v1/auth/oidc/:provider/callback", "Auth", "Identity provider callback",
			redirect("To the requested site path with session cookies set; to `/auth/sign-in?mfaChallenge=…` when the account has two-factor authentication; or to `/auth/sign-in?oidcError=expired|notLinked|accountUnavailable|providerUnavailable|cancelled|failed`. For account linking it returns to the profile with `linked=` or `oidcError=identityInUse|…`."),
			describe("Register `{PublicURL}/api/v1/auth/oidc/{provider}/callback` as the redirect URI at the provider. Accounts are matched by the provider's subject; unknown subjects are registered only when the provider allows it, never linked by email.")),
		route(pub, opPost, "/api/v1/auth/refresh-token", "Auth", "Refresh the access token", data(ExpireData{}),
			describe("Issues a new token (cookies reset) and blacklists the old one.")),
		route(pub, opPost, "/api/v1/auth/code", "Auth", "Send a verification code", body(dto.ConfirmCodeReq{}), data(CodeSentData{})),
		route(pub, opPut, "/api/v1/auth/password/expired", "Auth", "Replace an expired password", body(dto.ExpiredPasswordPutReq{}), data(EmptyObject{})),
		route(pub, opPost, "/api/v1/auth/register", "Auth", "Register an account", body(dto.UserRegisterReq{}), data(dto.UserResp{}),
			describe("Only when the `feature.register.enabled` setting is on.")),
		route(session, opPost, "/api/v1/auth/logout", "Auth", "Sign out"),

		// Account
		route(session, opGet, "/api/v1/account/info", "Account", "Get my profile", data(dto.UserResp{})),
		route(session, opPut, "/api/v1/account/password", "Account", "Change my password", body(dto.UserPasswordPutReq{}), data(EmptyObject{})),
		route(session, opPut, "/api/v1/account/notification-preferences", "Account", "Set my notification preferences", body(dto.NotificationPreferencesPutReq{}), data(EmptyObject{}),
			describe("`muteNotificationEmails` stops notification emails for this account; in-app notifications are unaffected.")),
		route(session, opPut, "/api/v1/account/name", "Account", "Change my display name", body(dto.UserNamePutReq{}), data(EmptyObject{})),
		route(session, opPost, "/api/v1/account/avatar", "Account", "Upload my avatar",
			form(openapi.FormField{Name: "file", File: true, Required: true}), data(service.UserAvatarResp{})),
		route(session, opPost, "/api/v1/account/contact/code", "Account", "Send a code to bind an email or mobile number", body(dto.ContactCodePostReq{}), data(EmptyObject{})),
		route(session, opPut, "/api/v1/account/contact", "Account", "Bind an email or mobile number", body(dto.ContactPutReq{}), data(EmptyObject{})),
		route(session, opGet, "/api/v1/account/login-histories", "Account", "List my sign-ins", list(dto.LoginHistoryResp{}, login_history.LoginHistory{})),
		route(session, opGet, "/api/v1/account/roles", "Account", "Get my roles", data(service.AccessSnapshot{})),
		route(session, opGet, "/api/v1/account/permissions", "Account", "Get my permissions", data(service.AccessSnapshot{}),
			describe("Permission identifiers are `{resourcePath}:{METHOD}`.")),
		route(session, opGet, "/api/v1/account/navigation", "Account", "Get my menu tree and allowed routes", data(dto.NavigationResp{})),
		route(session, opGet, "/api/v1/account/dictionaries", "Account", "Get all enabled dictionaries", data(map[string][]dto.DictItemResp{})),
		route(session, opPut, "/api/v1/account/session/roles", "Account", "Choose the active roles of my session", body(ActiveRolesRequest{}), data(service.AccessSnapshot{})),
		route(session, opGet, "/api/v1/account/totp", "Account", "Get my two-factor status", data(dto.TOTPStatusResp{})),
		route(session, opPost, "/api/v1/account/totp/setup", "Account", "Start authenticator setup", data(dto.TOTPSetupResp{}),
			describe("Returns a new secret (QR code and text) that stays inactive until confirmed; starting again replaces an unconfirmed secret.")),
		route(session, opPost, "/api/v1/account/totp/enable", "Account", "Turn on two-factor authentication", body(dto.TOTPCodeReq{}), data(dto.RecoveryCodesResp{}),
			describe("Confirms the setup with a code from the authenticator and returns 10 recovery codes, shown only once.")),
		route(session, opPost, "/api/v1/account/totp/disable", "Account", "Turn off two-factor authentication", body(dto.TOTPDisableReq{}),
			describe("Needs the current password (RSA-encrypted; accounts without a password leave it empty) and an authenticator or recovery code.")),
		route(session, opPost, "/api/v1/account/totp/recovery-codes", "Account", "Regenerate recovery codes", body(dto.TOTPCodeReq{}), data(dto.RecoveryCodesResp{})),
		route(session, opGet, "/api/v1/account/identities", "Account", "List my linked sign-in providers", data([]dto.UserIdentityResp{})),
		route(session, opPost, "/api/v1/account/identities/:provider/link", "Account", "Start linking an identity provider", data(dto.SSOLinkResp{}),
			describe("Navigate the browser to `authorizeUrl`; the callback links the provider account and returns to the profile.")),
		route(session, opDelete, "/api/v1/account/identities/:provider", "Account", "Unlink an identity provider",
			describe("Refused when it is the account's only way to sign in (no password and no other linked provider).")),
		route(session, opGet, "/api/v1/account/notifications", "Notifications", "List my notifications",
			list(dto.NotificationResp{}, nil), query(openapi.Param{Name: "unreadOnly", Type: "boolean"}),
			describe("Paging only; the filter and sort parameters are ignored.")),
		route(session, opGet, "/api/v1/account/notifications/stream", "Notifications", "Stream my notification events",
			stream("Server-Sent Events. `unread` carries `{count}` on connect and after every change; `notification` carries `{id, title, type, level, link}` when a new one arrives (followed by `unread`). Comment lines are heartbeats. The stream ends when the access token expires or the session is revoked; reconnect after refreshing the token."),
			describe("Up to 5 concurrent streams per user (429 beyond). Events reach every API replica through Redis pub/sub.")),
		route(session, opGet, "/api/v1/account/notifications/unread-count", "Notifications", "Count my unread notifications", data(dto.NotificationUnreadCountResp{})),
		route(session, opPut, "/api/v1/account/notifications/:id/read", "Notifications", "Mark a notification read", data(EmptyObject{})),
		route(session, opPut, "/api/v1/account/notifications/batch-read", "Notifications", "Mark notifications read", body(dto.MarkNotificationsReadReq{}), data(EmptyObject{})),
		route(session, opPut, "/api/v1/account/notifications/read-all", "Notifications", "Mark all notifications read", data(EmptyObject{})),
		route(session, opDelete, "/api/v1/account/notifications/:id", "Notifications", "Delete a notification from my inbox", data(EmptyObject{})),
		route(session, opGet, "/api/v1/account/notifications/:id/attachments/:objectKey", "Notifications", "Download a notification attachment", download(),
			describe("Any failure answers an empty 404.")),
		route(pub, opPut, "/api/v1/account/password/reset", "Account", "Reset a forgotten password with a verification code", body(dto.UserPasswordResetPutReq{}), data(EmptyObject{})),

		// Public assets, settings and dictionaries
		route(pub, opGet, "/api/v1/assets/download/:objectKey", "Assets", "Download a public asset", download(),
			describe("Only public, active, non-high-risk assets; everything else answers an empty 404.")),
		route(pub, opGet, "/api/v1/assets/public", "Assets", "List public assets", list(dto.AssetResp{}, asset.Asset{}),
			describe("Paging and `order` only.")),
		route(pub, opGet, "/api/v1/settings/public", "Settings", "Get public settings", data(map[string]any{})),
		route(pub, opGet, "/api/v1/dictionaries", "Dictionaries", "Get public dictionaries", data(map[string][]dto.DictItemResp{})),
		route(pub, opGet, "/api/v1/dictionaries/:typeCode", "Dictionaries", "Get a public dictionary", data([]dto.DictItemResp{}),
			describe("Non-public and unknown types both answer 404.")),

		// Admin: dashboard, assets
		route(admin, opGet, "/api/v1/admin/dashboard/stats", "Dashboard", "Get dashboard statistics", data(dto.DashboardStatsResp{})),
		route(admin, opGet, "/api/v1/admin/assets/download/:objectKey", "Assets", "Download an asset", download()),
		route(admin, opGet, "/api/v1/admin/assets", "Assets", "List assets", list(dto.AssetResp{}, asset.Asset{})),
		route(admin, opGet, "/api/v1/admin/assets/stats", "Assets", "Get asset statistics", data(dto.AssetStatsResp{})),
		route(admin, opGet, "/api/v1/admin/assets/:id", "Assets", "Get an asset with its references", data(dto.AssetDetailResp{})),
		route(admin, opPost, "/api/v1/admin/assets", "Assets", "Upload an asset", data(dto.AssetUploadResp{}),
			form(
				openapi.FormField{Name: "file", File: true, Required: true},
				openapi.FormField{Name: "name"}, openapi.FormField{Name: "description"},
				openapi.FormField{Name: "tags", Description: "Comma-separated."},
				openapi.FormField{Name: "folderPath"}, openapi.FormField{Name: "isPublic", Description: "`true` or `false`."},
				openapi.FormField{Name: "scope", Description: "`LIBRARY` (default) or `ATTACHMENT` (reclaimed with its references)."},
			)),
		route(admin, opPut, "/api/v1/admin/assets/:id", "Assets", "Update an asset", body(dto.AssetUpdateReq{}), data(dto.AssetResp{})),
		route(admin, opPut, "/api/v1/admin/assets/:id/status", "Assets", "Change an asset's status", body(dto.AssetStatusUpdateReq{})),
		route(admin, opPut, "/api/v1/admin/assets/:id/move", "Assets", "Move an asset to another folder", body(dto.AssetMoveReq{})),
		route(admin, opDelete, "/api/v1/admin/assets/:id", "Assets", "Delete an asset", data(EmptyObject{}),
			describe("Assets referenced by business records cannot be deleted (`errorCode` 4005).")),
		route(admin, opPost, "/api/v1/admin/assets/batch/delete", "Assets", "Delete assets", body(dto.AssetBatchDeleteReq{})),
		route(admin, opPut, "/api/v1/admin/assets/batch/status", "Assets", "Change the status of assets", body(dto.AssetBatchStatusUpdateReq{})),

		// Admin: login histories, sessions, jobs, audit logs
		route(admin, opGet, "/api/v1/admin/login-histories", "Login histories", "List sign-ins", list(dto.LoginHistoryResp{}, login_history.LoginHistory{})),
		route(admin, opGet, "/api/v1/admin/login-histories/export", "Login histories", "Export sign-ins", export(login_history.LoginHistory{})...),
		route(admin, opDelete, "/api/v1/admin/login-histories/:id", "Login histories", "Delete a sign-in record", data(EmptyObject{})),
		route(admin, opPost, "/api/v1/admin/login-histories/batch/delete", "Login histories", "Delete sign-in records", body(dto.LoginHistoryBatchDeleteReq{}), data(EmptyObject{})),
		route(admin, opGet, "/api/v1/admin/sessions", "Sessions", "List online sessions", list(dto.SessionResp{}, permission.AuthorizationSession{})),
		route(admin, opDelete, "/api/v1/admin/sessions/:id", "Sessions", "Sign a session out"),
		route(admin, opGet, "/api/v1/admin/jobs", "Scheduled jobs", "List scheduled jobs", data(dto.JobListResp{})),
		route(admin, opPut, "/api/v1/admin/jobs/:key", "Scheduled jobs", "Change a job's schedule", body(dto.JobUpdateRequest{}), data(dto.JobResp{})),
		route(admin, opPost, "/api/v1/admin/jobs/:key/run", "Scheduled jobs", "Run a job now", data(job.Run{}),
			describe("Starts the run in the background and returns it (status `RUNNING`); 409 when the job is already running.")),
		route(admin, opGet, "/api/v1/admin/job-runs", "Scheduled jobs", "List job runs", list(job.Run{}, job.Run{})),
		route(admin, opGet, "/api/v1/admin/audit-logs", "Audit logs", "List audit logs", list(audit_log.AuditLog{}, audit_log.AuditLog{})),
		route(admin, opGet, "/api/v1/admin/audit-logs/export", "Audit logs", "Export audit logs", export(audit_log.AuditLog{})...),
		route(admin, opPost, "/api/v1/admin/audit-logs/cleanup", "Audit logs", "Delete old audit logs", body(dto.AuditLogDeleteBeforeReq{}), data(dto.AuditLogDeleteBeforeResp{})),

		// Admin: settings, dictionaries
		route(admin, opGet, "/api/v1/admin/settings", "Settings", "List settings", data([]dto.SettingResp{}), query(openapi.Param{Name: "category"})),
		route(admin, opGet, "/api/v1/admin/settings/:id", "Settings", "Get a setting", data(dto.SettingResp{})),
		route(admin, opGet, "/api/v1/admin/settings/key/:key", "Settings", "Get a setting by key", data(dto.SettingResp{})),
		route(admin, opPost, "/api/v1/admin/settings", "Settings", "Create a setting", body(dto.SettingPostReq{}), data(dto.SettingResp{})),
		route(admin, opPut, "/api/v1/admin/settings/:id", "Settings", "Update a setting", body(dto.SettingPutReq{}), data(dto.SettingResp{})),
		route(admin, opPut, "/api/v1/admin/settings/batch", "Settings", "Update settings", body(dto.SettingBatchUpdateReq{}), data(EmptyObject{})),
		route(admin, opDelete, "/api/v1/admin/settings/:id", "Settings", "Delete a setting", data(EmptyObject{})),
		route(admin, opGet, "/api/v1/admin/dict-types", "Dictionaries", "List dictionary types", data([]dto.DictTypeResp{})),
		route(admin, opGet, "/api/v1/admin/dict-types/:id", "Dictionaries", "Get a dictionary type", data(dto.DictTypeResp{})),
		route(admin, opPost, "/api/v1/admin/dict-types", "Dictionaries", "Create a dictionary type", body(dto.DictTypePostReq{}), data(dto.DictTypeResp{})),
		route(admin, opPut, "/api/v1/admin/dict-types/:id", "Dictionaries", "Update a dictionary type", body(dto.DictTypePutReq{}), data(dto.DictTypeResp{})),
		route(admin, opDelete, "/api/v1/admin/dict-types/:id", "Dictionaries", "Delete a dictionary type", data(EmptyObject{})),
		route(admin, opGet, "/api/v1/admin/dict-items", "Dictionaries", "List dictionary items", data([]dto.DictItemResp{}), query(openapi.Param{Name: "typeCode"})),
		route(admin, opGet, "/api/v1/admin/dict-items/type/:typeCode", "Dictionaries", "List the items of a dictionary type", data([]dto.DictItemResp{})),
		route(admin, opGet, "/api/v1/admin/dict-items/:id", "Dictionaries", "Get a dictionary item", data(dto.DictItemResp{})),
		route(admin, opPost, "/api/v1/admin/dict-items", "Dictionaries", "Create a dictionary item", body(dto.DictItemPostReq{}), data(dto.DictItemResp{})),
		route(admin, opPut, "/api/v1/admin/dict-items/:id", "Dictionaries", "Update a dictionary item", body(dto.DictItemPutReq{}), data(dto.DictItemResp{}),
			describe("The `value` of system (seeded) items cannot change.")),
		route(admin, opDelete, "/api/v1/admin/dict-items/:id", "Dictionaries", "Delete a dictionary item", data(EmptyObject{})),

		// Admin: notifications
		route(admin, opGet, "/api/v1/admin/notifications", "Notifications", "List notifications", list(dto.NotificationResp{}, notification.Notification{})),
		route(admin, opGet, "/api/v1/admin/notifications/channels", "Notifications", "List available delivery channels", data(dto.NotificationChannelsResp{}),
			describe("`email` is true when SMTP is configured; only then can a notification set `sendEmail`. Emails go to recipients with a verified email who have not muted notification emails, sent by the `deliverNotificationEmails` scheduled job with retries.")),
		route(admin, opGet, "/api/v1/admin/notifications/:id", "Notifications", "Get a notification", data(dto.NotificationResp{})),
		route(admin, opGet, "/api/v1/admin/notifications/:id/attachments/:objectKey", "Notifications", "Download a notification attachment", download(),
			describe("Only files attached to that notification; any failure answers an empty 404.")),
		route(admin, opGet, "/api/v1/admin/notifications/:id/recipients", "Notifications", "List a notification's recipients", list(dto.NotificationRecipientResp{}, nil),
			describe("Paging only.")),
		route(admin, opPost, "/api/v1/admin/notifications", "Notifications", "Create a notification", body(dto.NotificationPostReq{}), data(dto.NotificationResp{})),
		route(admin, opPut, "/api/v1/admin/notifications/:id", "Notifications", "Update a notification", body(dto.NotificationPutReq{}), data(dto.NotificationResp{})),
		route(admin, opDelete, "/api/v1/admin/notifications/:id", "Notifications", "Delete a notification", data(EmptyObject{})),
		route(admin, opPost, "/api/v1/admin/notifications/batch/delete", "Notifications", "Delete notifications", body(dto.NotificationBatchDeleteReq{}), data(EmptyObject{})),
		route(admin, opPost, "/api/v1/admin/notifications/attachments", "Notifications", "Upload a notification attachment",
			form(openapi.FormField{Name: "file", File: true, Required: true}), data(dto.AssetAttachmentResp{})),

		// Admin: users
		route(admin, opGet, "/api/v1/admin/users", "Users", "List users", list(dto.UserResp{}, user.User{})),
		route(admin, opGet, "/api/v1/admin/users/export", "Users", "Export users", export(user.User{})...),
		route(admin, opGet, "/api/v1/admin/users/import-template", "Users", "Download the user import template",
			download("application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", "text/csv"), query(formatParam),
			describe("Headers look like `Username* (username)`; the key in parentheses is language-neutral.")),
		route(admin, opPost, "/api/v1/admin/users/import", "Users", "Import users", data(dto.UserImportResult{}),
			form(
				openapi.FormField{Name: "file", File: true, Required: true, Description: "`.xlsx` or UTF-8 `.csv`, at most 5 MB and 1,000 rows."},
				openapi.FormField{Name: "password", Required: true, Description: "RSA-encrypted initial password shared by every imported user; it is already expired, so each user must change it at first sign-in."},
			),
			describe("Every row is validated first. If any row is invalid nothing is created and the error response's `data` is `{created, errors: [{line, field, code, message}]}`.")),
		route(admin, opGet, "/api/v1/admin/users/:id", "Users", "Get a user", data(dto.UserResp{})),
		route(admin, opPost, "/api/v1/admin/users", "Users", "Create a user", body(dto.UserPostReq{}), data(dto.UserResp{})),
		route(admin, opPut, "/api/v1/admin/users/:id", "Users", "Update a user", body(dto.UserPutReq{}), data(dto.UserResp{})),
		route(admin, opDelete, "/api/v1/admin/users/:id", "Users", "Delete a user", data(EmptyObject{})),
		route(admin, opDelete, "/api/v1/admin/users/:id/totp", "Users", "Reset a user's two-factor authentication",
			describe("For users who lost their authenticator and recovery codes. The caller must be able to manage the user (data scope and permissions).")),

		// Admin: menus, resources
		route(admin, opGet, "/api/v1/admin/menus", "Menus", "Get the menu tree and its permission catalog", data(dto.MenuCatalog{})),
		route(admin, opPost, "/api/v1/admin/menus", "Menus", "Create a menu", body(dto.MenuRequest{}), data(dto.MenuResp{})),
		route(admin, opPut, "/api/v1/admin/menus/:id", "Menus", "Update a menu", body(dto.MenuRequest{}), data(dto.MenuResp{})),
		route(admin, opDelete, "/api/v1/admin/menus/:id", "Menus", "Delete a menu"),
		route(admin, opGet, "/api/v1/admin/resources", "Resources", "List resources", list(dto.ResourceResp{}, permission.Resource{})),
		route(admin, opGet, "/api/v1/admin/resources/modules", "Resources", "List resource modules", data([]string{})),
		route(admin, opGet, "/api/v1/admin/resources/:id", "Resources", "Get a resource", data(dto.ResourceResp{})),
		route(admin, opPost, "/api/v1/admin/resources", "Resources", "Create a resource", body(dto.ResourceCreateReq{}), data(dto.ResourceResp{})),
		route(admin, opPut, "/api/v1/admin/resources/:id", "Resources", "Update a resource", body(dto.ResourceUpdateReq{}), data(dto.ResourceResp{})),
		route(admin, opDelete, "/api/v1/admin/resources/:id", "Resources", "Delete a resource", data(DeletedData{})),

		// Admin: roles, departments
		route(admin, opGet, "/api/v1/admin/roles", "Roles", "List roles", data([]dto.RoleResp{})),
		route(admin, opGet, "/api/v1/admin/roles/:role", "Roles", "Get a role", data(dto.RoleDetailResp{})),
		route(admin, opPost, "/api/v1/admin/roles", "Roles", "Create a role", body(dto.RoleCreateReq{}), data(dto.RoleResp{}),
			describe("New roles start with the `SELF` data scope.")),
		route(admin, opPut, "/api/v1/admin/roles/:role", "Roles", "Update a role", body(dto.RoleUpdateReq{}), data(dto.RoleResp{})),
		route(admin, opDelete, "/api/v1/admin/roles/:role", "Roles", "Delete a role", data(RoleDeletedData{})),
		route(admin, opGet, "/api/v1/admin/roles/:role/permissions", "Roles", "Get a role's permissions", data(RolePermissionsData{})),
		route(admin, opPut, "/api/v1/admin/roles/:role/permissions", "Roles", "Set a role's permissions", body(dto.RolePermissionSetReq{}), data(UpdatedData{})),
		route(admin, opGet, "/api/v1/admin/roles/:role/users", "Roles", "List a role's members", data([]string{}),
			describe("Only members inside the caller's data scope.")),
		route(admin, opGet, "/api/v1/admin/roles/:role/hierarchy", "Roles", "Get a role's junior roles", data([]permission.Role{})),
		route(admin, opPut, "/api/v1/admin/roles/:role/hierarchy", "Roles", "Set a role's junior roles", body(RoleHierarchyRequest{}), data(UpdatedData{})),
		route(admin, opPut, "/api/v1/admin/roles/:role/data-scope", "Roles", "Set a role's data scope", body(dto.RoleDataScopeReq{}), data(dto.RoleResp{}),
			describe("Cannot exceed the caller's own data scope.")),
		route(admin, opGet, "/api/v1/admin/departments", "Departments", "Get the department tree", data([]dto.DepartmentResp{}),
			describe("A flat list sorted by `sortOrder`; `inScope` marks departments inside the caller's data scope.")),
		route(admin, opPost, "/api/v1/admin/departments", "Departments", "Create a department", body(dto.DepartmentRequest{}), data(department.Department{})),
		route(admin, opPut, "/api/v1/admin/departments/:id", "Departments", "Update a department", body(dto.DepartmentRequest{}), data(department.Department{})),
		route(admin, opDelete, "/api/v1/admin/departments/:id", "Departments", "Delete a department",
			describe("Rejected while the department has sub-departments or members.")),

		// Admin: authorization
		route(admin, opGet, "/api/v1/admin/authorization/constraints", "Authorization", "List separation-of-duty constraints", data([]permission.SeparationConstraint{})),
		route(admin, opPost, "/api/v1/admin/authorization/constraints", "Authorization", "Create a constraint", body(permission.SeparationConstraint{}), data(permission.SeparationConstraint{})),
		route(admin, opPut, "/api/v1/admin/authorization/constraints/:id", "Authorization", "Update a constraint", body(permission.SeparationConstraint{}), data(permission.SeparationConstraint{})),
		route(admin, opDelete, "/api/v1/admin/authorization/constraints/:id", "Authorization", "Delete a constraint", data(DeletedData{})),
		route(admin, opGet, "/api/v1/admin/authorization/users/:username/roles", "Authorization", "List a user's roles", data([]string{})),
		route(admin, opPost, "/api/v1/admin/authorization/users/:username/roles", "Authorization", "Assign a role to a user", body(dto.RoleForUserReq{}), data(AddedData{})),
		route(admin, opDelete, "/api/v1/admin/authorization/users/:username/roles/:role", "Authorization", "Remove a role from a user", data(DeletedData{})),
		route(admin, opGet, "/api/v1/admin/authorization/users/:username/permissions", "Authorization", "Get a user's permissions", data(service.AccessSnapshot{})),
		route(admin, opGet, "/api/v1/admin/authorization/metadata/subjects", "Authorization", "List role codes", data([]string{})),
		route(admin, opGet, "/api/v1/admin/authorization/metadata/objects", "Authorization", "List resource paths", data([]string{})),
		route(admin, opGet, "/api/v1/admin/authorization/metadata/actions", "Authorization", "List actions", data([]string{})),
		route(admin, opGet, "/api/v1/admin/authorization/metadata/policies", "Authorization", "List permission policies", data([]dto.PermissionResp{})),
	}
	return ops
}()

// tagDescriptions 接口分组说明
var tagDescriptions = map[string]string{
	"System":          "Probes and this document.",
	"Auth":            "Sign-in, token refresh, verification codes and registration. Credentials are RSA-encrypted with the key from `/api/v1/auth/public-key`.",
	"Account":         "The signed-in user's own profile, password, contacts, roles and navigation.",
	"Notifications":   "Inbox of the signed-in user, and notification management for admins.",
	"Assets":          "Files. Business records store asset object keys; public assets are downloadable without signing in.",
	"Settings":        "System settings; public ones are readable without signing in.",
	"Dictionaries":    "Enum labels, colors and icons in four languages. Values are owned by backend constants.",
	"Dashboard":       "Statistics, limited to the caller's data scope.",
	"Login histories": "Sign-in records, limited to the caller's data scope.",
	"Sessions":        "Online sessions and forced sign-out.",
	"Scheduled jobs":  "Jobs registered in code: schedules, manual runs and run history.",
	"Audit logs":      "Audit trail of administrative actions.",
	"Users":           "User management, export and import. Single users outside the caller's data scope answer 404.",
	"Menus":           "Navigation menu tree and the permissions each entry needs.",
	"Resources":       "API resources (route template + methods) that RBAC authorizes.",
	"Roles":           "Roles, their permissions, hierarchy, members and data scope.",
	"Departments":     "Department tree used by data scopes.",
	"Authorization":   "User-role assignments, separation-of-duty constraints and RBAC metadata.",
	"Single sign-on":  "OpenID Connect identity providers users can sign in with.",
}

// errorCodeDescriptions errorCode 的说明（openapi_catalog_test.go 核对 response/errors.go 里每个码都有说明）
var errorCodeDescriptions = map[int]string{
	response.ErrCodeBadRequest:            "Bad request",
	response.ErrCodeInvalidInput:          "Invalid input",
	response.ErrCodeNotFound:              "Not found",
	response.ErrCodeConflict:              "Conflict",
	response.ErrCodeAlreadyExists:         "Already exists",
	response.ErrCodeTooManyRequests:       "Too many requests",
	response.ErrCodeUnauthorized:          "Not signed in, or the token is invalid",
	response.ErrCodeForbidden:             "Forbidden",
	response.ErrCodeInvalidCaptcha:        "Captcha required or wrong",
	response.ErrCodeInvalidCredentials:    "Wrong username or password",
	response.ErrCodeInvalidConfirmCode:    "Wrong verification code",
	response.ErrCodeInvalidLoginMethod:    "Unsupported sign-in method",
	response.ErrCodeAccountDisabled:       "Account disabled",
	response.ErrCodeAccountLocked:         "Account locked",
	response.ErrCodeAccountExpired:        "Account expired",
	response.ErrCodeCredentialExpired:     "Password expired; replace it via PUT /api/v1/auth/password/expired",
	response.ErrCodeTooManyAttempts:       "Too many sign-in attempts",
	response.ErrCodeRegisterDisabled:      "Registration is closed",
	response.ErrCodeTOTPRequired:          "Two-factor code required: send `data.challenge` with the code to POST /api/v1/auth/login/totp",
	response.ErrCodeInvalidTOTP:           "Wrong or already used two-factor code or recovery code",
	response.ErrCodeMFAChallengeExpired:   "The two-factor step expired or ran out of attempts; sign in again",
	response.ErrCodeUserNotFound:          "User not found",
	response.ErrCodeUserCreateFailed:      "User creation failed",
	response.ErrCodeUserUpdateFailed:      "User update failed",
	response.ErrCodeAssetUploadFailed:     "Upload failed",
	response.ErrCodeAssetNotFound:         "Asset not found",
	response.ErrCodeAssetTooLarge:         "File too large",
	response.ErrCodeAssetInvalidType:      "File type not allowed",
	response.ErrCodeAssetInUse:            "Asset is referenced by business records",
	response.ErrCodeSettingNotFound:       "Setting not found",
	response.ErrCodeSettingSaveFailed:     "Setting could not be saved",
	response.ErrCodeDeliveryNotConfigured: "SMS or email delivery is not configured",
	response.ErrCodeInternalError:         "Internal error",
	response.ErrCodeDatabaseError:         "Database error",
	response.ErrCodeCacheError:            "Cache error",
	response.ErrCodeStorageError:          "Storage error",
}

var (
	openAPIOnce sync.Once
	openAPIJSON []byte
	openAPIErr  error
)

// BuildOpenAPI 生成完整的 OpenAPI 文档
func BuildOpenAPI() (openapi.Document, error) {
	return openapi.Build(apiOperations, openapi.Options{
		Info: openapi.Info{
			Title:   "Castor API",
			Version: "v1",
			Description: "Responses use the envelope `{ code, data, message }` (`code` 200 on success; `message` localized per `Accept-Language`). " +
				"Errors add `errorCode`, the business code clients branch on. Admin endpoints are authorized by RBAC per route template and method, " +
				"and lists that expose users' data only cover the caller's data scope.",
		},
		FilterFields:    handler.AllowedQueryFields,
		ErrorCodes:      errorCodeDescriptions,
		TagDescriptions: tagDescriptions,
	})
}

func openAPIDocument() ([]byte, error) {
	openAPIOnce.Do(func() {
		doc, err := BuildOpenAPI()
		if err != nil {
			openAPIErr = err
			return
		}
		openAPIJSON, openAPIErr = doc.JSON()
	})
	return openAPIJSON, openAPIErr
}

// serveOpenAPI GET /api/v1/admin/openapi.json
func (r *Router) serveOpenAPI(c *gin.Context) {
	doc, err := openAPIDocument()
	if err != nil {
		response.HandleError(c, err)
		return
	}
	c.Data(http.StatusOK, "application/json; charset=utf-8", doc)
}
