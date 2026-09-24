package apperror

import (
	"errors"

	"github.com/castorworks/castor/internal/domain/shared"
	"github.com/castorworks/castor/internal/pkg/query"
)

// 通用错误
var (
	ErrRecordNotFound = shared.ErrNotFound
	ErrInvalidOrder   = query.ErrInvalidOrder
	// ErrTooManyRequests 业务级限流（按调用者/目标计数），与网关的 IP 限流互补
	ErrTooManyRequests = errors.New("too many requests, please try again later")
)

// 认证相关错误
var (
	ErrRegisterNotEnabled   = errors.New("register is not enabled")
	ErrInvalidConfirmCode   = errors.New("confirm code is invalid")
	ErrUsernameAlreadyTaken = errors.New("username is already taken")
	ErrInvalidCaptchaCode   = errors.New("captcha code is invalid")
	ErrInvalidCodeType      = errors.New("invalid code type")
	ErrRateLimitCheckFailed = errors.New("rate limit check failed")
	ErrInvalidEmailFormat   = errors.New("invalid email format")
	ErrInvalidPhoneFormat   = errors.New("invalid phone number format")
	ErrLoginMethodDisabled  = errors.New("login method is disabled")
	ErrAuthenticationFailed = errors.New("authentication failed")
	// ErrDeliveryChannelNotConfigured 部署未配置对应的验证码投递通道（短信服务密钥或邮件服务器）
	ErrDeliveryChannelNotConfigured = errors.New("verification code delivery channel is not configured")
)

// 用户状态相关错误
var (
	ErrUserDisabled           = errors.New("user account is disabled")
	ErrUserLocked             = errors.New("user account is locked")
	ErrAccountExpired         = errors.New("user account has expired")
	ErrCredentialExpired      = errors.New("user credentials have expired")
	ErrTooManyLoginAttempts   = errors.New("too many login attempts, please try again later")
	ErrInvalidLoginCredential = errors.New("invalid username or password")
)

// 账户相关错误
var (
	ErrCurrentPasswordIncorrect = errors.New("current password is not correct")
	ErrPasswordEncryptFailed    = errors.New("failed to encrypt password")
	ErrPasswordDecryptFailed    = errors.New("failed to decrypt password")
	// ErrInvalidContactType 联系方式类型不是 EMAIL/MOBILE
	ErrInvalidContactType = errors.New("invalid contact type")
	// ErrContactAlreadyUsed 邮箱或手机号已被其它账号占用
	ErrContactAlreadyUsed = errors.New("contact is already used by another account")
	// ErrDefaultAdminPasswordRequired 非开发模式初始化数据库时未提供合规的初始管理员密码
	ErrDefaultAdminPasswordRequired = errors.New("initial admin password is required outside development mode")
)

// 配置相关错误
var (
	ErrSettingKeyExists          = errors.New("setting key already exists")
	ErrInvalidSettingValue       = errors.New("invalid setting value")
	ErrSystemSettingDelete       = errors.New("cannot delete system setting")
	ErrInvalidLoginMethodSetting = errors.New("invalid login method setting")
)

// 字典相关错误
var (
	ErrDictTypeCodeExists   = errors.New("dictionary type code already exists")
	ErrSystemDictTypeDelete = errors.New("cannot delete system dictionary type")
	ErrDictItemValueExists  = errors.New("dictionary item value already exists in this type")
	// 系统字典项的 value 是与代码里枚举常量的连接键，改掉或删掉都会让界面静默退回原始值。
	ErrSystemDictItemDelete      = errors.New("cannot delete system dictionary item")
	ErrSystemDictItemValueLocked = errors.New("cannot change the value of a system dictionary item")
	ErrDictItemColorInvalid      = errors.New("dictionary item color is not a known tag color")
)

// 通知相关错误
var (
	ErrNotificationRecipientsRequired = errors.New("notification recipients are required")
	ErrNotificationNotDelivered       = errors.New("notification is not delivered to user")
)

// 权限相关错误
var (
	ErrResourceNotFound             = errors.New("resource not found")
	ErrResourceCodeExists           = errors.New("resource code already exists")
	ErrInvalidResource              = errors.New("invalid resource definition")
	ErrResourcePermissionConflict   = errors.New("resource path and action conflict")
	ErrSystemResourceDelete         = errors.New("cannot delete system resource")
	ErrRoleNotFound                 = errors.New("role not found")
	ErrRoleCodeExists               = errors.New("role code already exists")
	ErrInvalidRole                  = errors.New("invalid role definition")
	ErrRoleInUse                    = errors.New("role is referenced by a separation of duty constraint")
	ErrSystemRoleDelete             = errors.New("cannot delete system role")
	ErrRolePermissionInvalid        = errors.New("invalid role permission")
	ErrRoleHierarchyCycle           = errors.New("role hierarchy contains a cycle")
	ErrInvalidRoleHierarchy         = errors.New("invalid role hierarchy")
	ErrInvalidConstraint            = errors.New("invalid separation of duty constraint")
	ErrSSDViolation                 = errors.New("static separation of duty constraint violated")
	ErrDSDViolation                 = errors.New("dynamic separation of duty constraint violated")
	ErrRoleNotAuthorized            = errors.New("role is not authorized for this user")
	ErrAuthorizationSessionNotFound = errors.New("authorization session not found")
	ErrAuthorizationSessionExpired  = errors.New("authorization session expired")
	ErrAuthorizationSessionRevoked  = errors.New("authorization session revoked")
)

// 用户管理保护错误
var (
	ErrCannotModifySystemUser = errors.New("cannot modify system user status")
	ErrCannotModifyOwnStatus  = errors.New("cannot modify own account status")
	ErrCannotDeleteSystemUser = errors.New("cannot delete system user")
	ErrCannotDeleteOwnAccount = errors.New("cannot delete own account")
)

// 授权提升保护错误
var (
	ErrCannotModifySystemRole     = errors.New("cannot modify system role")
	ErrCannotModifySystemResource = errors.New("cannot modify system resource")
	ErrGrantExceedsCaller         = errors.New("cannot grant permissions beyond your own")
	// ErrTargetUserExceedsCaller 目标账号拥有调用者没有的权限：重置其密码、改状态、删除或踢下线
	// 都等于接管这些权限，与"不能授予超出自身的权限"是同一条防越权规则。
	ErrTargetUserExceedsCaller = errors.New("cannot manage a user holding permissions beyond your own")
)

// 用户管理验证错误
var (
	ErrNoFieldsToUpdate = errors.New("no fields to update")
	ErrNameTooLong      = errors.New("name exceeds maximum length")
	ErrPasswordTooShort = errors.New("password does not meet minimum length requirement")
	// ErrPasswordTooLong bcrypt 只接受 72 字节以内的口令，超出部分无法参与校验
	ErrPasswordTooLong = errors.New("password exceeds maximum length")
	// ErrPasswordTooWeak 启用复杂度要求时，密码未覆盖足够的字符类别
	ErrPasswordTooWeak = errors.New("password does not meet complexity requirement")
	// ErrPasswordUnchanged 新密码与当前密码相同
	ErrPasswordUnchanged = errors.New("new password must differ from the current password")
	// ErrCredentialNotExpired 过期改密入口只受理凭证确已过期的账号
	ErrCredentialNotExpired = errors.New("credentials have not expired")
)

// 资产相关错误
var (
	ErrAssetNotFound    = errors.New("asset not found")
	ErrAssetTooLarge    = errors.New("file size exceeds upload limit")
	ErrAssetInvalidType = errors.New("file type is not allowed")
	ErrAssetInUse       = errors.New("asset is referenced by business records")
)

// 部门与数据范围相关错误
var (
	ErrDepartmentNotFound    = errors.New("department not found")
	ErrInvalidDepartment     = errors.New("invalid department definition")
	ErrDepartmentConflict    = errors.New("department code already exists")
	ErrDepartmentHierarchy   = errors.New("invalid department hierarchy")
	ErrDepartmentHasChildren = errors.New("department has child departments")
	ErrDepartmentHasMembers  = errors.New("department still has members")
	ErrInvalidDataScope      = errors.New("invalid data scope")
	// ErrDataScopeExceedsCaller 授出、接管或清理的数据范围超出调用者自己的数据范围
	ErrDataScopeExceedsCaller = errors.New("data scope exceeds your own")
)

// 两步验证相关错误
var (
	ErrTOTPRequired        = errors.New("two-factor code required")
	ErrInvalidTOTPCode     = errors.New("invalid two-factor code")
	ErrMFAChallengeExpired = errors.New("two-factor sign-in expired")
	ErrTOTPAlreadyEnabled  = errors.New("two-factor authentication is already enabled")
	ErrTOTPNotEnabled      = errors.New("two-factor authentication is not enabled")
	ErrTOTPSetupNotStarted = errors.New("two-factor setup has not been started")
)

// OIDC 单点登录相关错误
var (
	ErrOIDCProviderNotFound = errors.New("identity provider not found")
	ErrOIDCProviderConflict = errors.New("identity provider code already exists")
	ErrInvalidOIDCProvider  = errors.New("invalid identity provider settings")
	ErrOIDCNotConfigured    = errors.New("single sign-on needs General.PublicURL")
	ErrOIDCDiscoveryFailed  = errors.New("identity provider discovery failed")
	ErrOIDCStateInvalid     = errors.New("single sign-on request expired or was already used")
	ErrOIDCAuthFailed       = errors.New("identity provider sign-in failed")
	ErrOIDCAccountNotLinked = errors.New("no account is linked to this identity")
	ErrOIDCIdentityInUse    = errors.New("this identity is linked to another account")
	ErrOIDCLastSignInMethod = errors.New("cannot remove the last way to sign in")
)

// 导入导出相关错误
var (
	ErrUnsupportedFileFormat = errors.New("unsupported file format")
	ErrFileUnreadable        = errors.New("file cannot be read")
	ErrImportEmpty           = errors.New("import file has no rows")
	ErrImportTooManyRows     = errors.New("import file has too many rows")
	ErrImportMissingColumn   = errors.New("import file is missing a required column")
	ErrImportInvalidRows     = errors.New("import file has invalid rows")
	ErrExportTooLarge        = errors.New("too many rows to export")
)

// 定时任务相关错误
var (
	ErrJobNotFound = errors.New("scheduled job not found")
	ErrInvalidCron = errors.New("invalid cron expression")
	// ErrJobAlreadyRunning 同一任务不重叠执行（跨副本由 Redis 锁保证）
	ErrJobAlreadyRunning = errors.New("scheduled job is already running")
)

var (
	ErrInvalidMenu     = errors.New("invalid menu definition")
	ErrMenuNotFound    = errors.New("menu not found")
	ErrMenuConflict    = errors.New("duplicate menu code or page path")
	ErrMenuHierarchy   = errors.New("invalid menu hierarchy")
	ErrMenuHasChildren = errors.New("menu has children")
	ErrMenuPermission  = errors.New("invalid menu permission")
)
