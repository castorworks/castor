// Package service - error definitions have been moved.
//
// All error sentinel values have been relocated to:
//
//	github.com/castorworks/castor/internal/application/apperror
//
// These re-exports are kept for backward compatibility during transition.
package service

import "github.com/castorworks/castor/internal/application/apperror"

// 通用错误
var (
	ErrRecordNotFound  = apperror.ErrRecordNotFound
	ErrTooManyRequests = apperror.ErrTooManyRequests
)

// 认证相关错误
var (
	ErrRegisterNotEnabled   = apperror.ErrRegisterNotEnabled
	ErrInvalidConfirmCode   = apperror.ErrInvalidConfirmCode
	ErrUsernameAlreadyTaken = apperror.ErrUsernameAlreadyTaken
	ErrInvalidCaptchaCode   = apperror.ErrInvalidCaptchaCode
	ErrInvalidCodeType      = apperror.ErrInvalidCodeType
	ErrRateLimitCheckFailed = apperror.ErrRateLimitCheckFailed
	ErrInvalidEmailFormat   = apperror.ErrInvalidEmailFormat
	ErrInvalidPhoneFormat   = apperror.ErrInvalidPhoneFormat
	ErrLoginMethodDisabled  = apperror.ErrLoginMethodDisabled
	ErrAuthenticationFailed = apperror.ErrAuthenticationFailed

	ErrDeliveryChannelNotConfigured = apperror.ErrDeliveryChannelNotConfigured
)

// 用户状态相关错误
var (
	ErrUserDisabled           = apperror.ErrUserDisabled
	ErrUserLocked             = apperror.ErrUserLocked
	ErrAccountExpired         = apperror.ErrAccountExpired
	ErrCredentialExpired      = apperror.ErrCredentialExpired
	ErrTooManyLoginAttempts   = apperror.ErrTooManyLoginAttempts
	ErrInvalidLoginCredential = apperror.ErrInvalidLoginCredential
)

// 账户相关错误
var (
	ErrCurrentPasswordIncorrect     = apperror.ErrCurrentPasswordIncorrect
	ErrPasswordEncryptFailed        = apperror.ErrPasswordEncryptFailed
	ErrPasswordDecryptFailed        = apperror.ErrPasswordDecryptFailed
	ErrDefaultAdminPasswordRequired = apperror.ErrDefaultAdminPasswordRequired
	ErrInvalidContactType           = apperror.ErrInvalidContactType
	ErrContactAlreadyUsed           = apperror.ErrContactAlreadyUsed
)

// 配置相关错误
var (
	ErrSettingKeyExists          = apperror.ErrSettingKeyExists
	ErrInvalidSettingValue       = apperror.ErrInvalidSettingValue
	ErrSystemSettingDelete       = apperror.ErrSystemSettingDelete
	ErrInvalidLoginMethodSetting = apperror.ErrInvalidLoginMethodSetting
)

// 字典相关错误
var (
	ErrDictTypeCodeExists        = apperror.ErrDictTypeCodeExists
	ErrSystemDictTypeDelete      = apperror.ErrSystemDictTypeDelete
	ErrDictItemValueExists       = apperror.ErrDictItemValueExists
	ErrSystemDictItemDelete      = apperror.ErrSystemDictItemDelete
	ErrSystemDictItemValueLocked = apperror.ErrSystemDictItemValueLocked
	ErrDictItemColorInvalid      = apperror.ErrDictItemColorInvalid
)

// 用户管理保护错误
var (
	ErrCannotModifySystemUser = apperror.ErrCannotModifySystemUser
	ErrCannotModifyOwnStatus  = apperror.ErrCannotModifyOwnStatus
	ErrCannotDeleteSystemUser = apperror.ErrCannotDeleteSystemUser
	ErrCannotDeleteOwnAccount = apperror.ErrCannotDeleteOwnAccount
)

// 用户管理验证错误
var (
	ErrNoFieldsToUpdate = apperror.ErrNoFieldsToUpdate
	ErrNameTooLong      = apperror.ErrNameTooLong
	ErrPasswordTooShort = apperror.ErrPasswordTooShort
)
