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

// 用户管理验证错误
var (
	ErrNoFieldsToUpdate = errors.New("no fields to update")
	ErrNameTooLong      = errors.New("name exceeds maximum length")
	ErrPasswordTooShort = errors.New("password does not meet minimum length requirement")
)

// 资产相关错误
var (
	ErrAssetNotFound    = errors.New("asset not found")
	ErrAssetTooLarge    = errors.New("file size exceeds upload limit")
	ErrAssetInvalidType = errors.New("file type is not allowed")
)

var (
	ErrInvalidMenu     = errors.New("invalid menu definition")
	ErrMenuNotFound    = errors.New("menu not found")
	ErrMenuConflict    = errors.New("duplicate menu code or page path")
	ErrMenuHierarchy   = errors.New("invalid menu hierarchy")
	ErrMenuHasChildren = errors.New("menu has children")
	ErrMenuPermission  = errors.New("invalid menu permission")
)
