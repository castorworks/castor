package response

import "net/http"

// ErrCode 统一错误码定义
// 将业务错误码、HTTP 状态码和 i18n 消息 key 关联在一起，
// 消除 errors.go 和 messages.go 之间的冗余映射。
type ErrCode struct {
	Code       int    // 业务错误码
	HTTPStatus int    // HTTP 状态码
	MsgKey     string // i18n 消息 key
}

// 预定义的统一错误码
// 每个错误码同时关联了业务码、HTTP 状态码和 i18n key
var (
	// === 成功 ===
	CodeOK = ErrCode{Code: ErrCodeSuccess, HTTPStatus: http.StatusOK, MsgKey: MsgSuccess}

	// === 1xxx: 通用业务错误 ===
	CodeBadReq        = ErrCode{Code: ErrCodeBadRequest, HTTPStatus: http.StatusBadRequest, MsgKey: MsgBadRequest}
	CodeInvalidInput  = ErrCode{Code: ErrCodeInvalidInput, HTTPStatus: http.StatusBadRequest, MsgKey: MsgBadRequest}
	CodeNotFoundErr   = ErrCode{Code: ErrCodeNotFound, HTTPStatus: http.StatusNotFound, MsgKey: MsgNotFound}
	CodeConflictErr   = ErrCode{Code: ErrCodeConflict, HTTPStatus: http.StatusConflict, MsgKey: MsgConflict}
	CodeAlreadyExists = ErrCode{Code: ErrCodeAlreadyExists, HTTPStatus: http.StatusConflict, MsgKey: MsgConflict}
	CodeTooManyReqs   = ErrCode{Code: ErrCodeTooManyRequests, HTTPStatus: http.StatusTooManyRequests, MsgKey: MsgTooManyRequests}

	// === 2xxx: 认证/授权错误 ===
	CodeUnauth           = ErrCode{Code: ErrCodeUnauthorized, HTTPStatus: http.StatusUnauthorized, MsgKey: MsgUnauthorized}
	CodeForbid           = ErrCode{Code: ErrCodeForbidden, HTTPStatus: http.StatusForbidden, MsgKey: MsgForbidden}
	CodeInvalidCaptcha   = ErrCode{Code: ErrCodeInvalidCaptcha, HTTPStatus: http.StatusBadRequest, MsgKey: MsgInvalidCaptchaCode}
	CodeInvalidCreds     = ErrCode{Code: ErrCodeInvalidCredentials, HTTPStatus: http.StatusUnauthorized, MsgKey: MsgInvalidUsernameOrPassword}
	CodeInvalidConfirm   = ErrCode{Code: ErrCodeInvalidConfirmCode, HTTPStatus: http.StatusBadRequest, MsgKey: MsgInvalidConfirmCode}
	CodeInvalidLogin     = ErrCode{Code: ErrCodeInvalidLoginMethod, HTTPStatus: http.StatusBadRequest, MsgKey: MsgInvalidLoginMethod}
	CodeLoginDisabled    = ErrCode{Code: ErrCodeForbidden, HTTPStatus: http.StatusForbidden, MsgKey: MsgLoginMethodDisabled}
	CodeAcctDisabled     = ErrCode{Code: ErrCodeAccountDisabled, HTTPStatus: http.StatusForbidden, MsgKey: MsgUserDisabled}
	CodeAcctLocked       = ErrCode{Code: ErrCodeAccountLocked, HTTPStatus: http.StatusForbidden, MsgKey: MsgUserLocked}
	CodeAcctExpired      = ErrCode{Code: ErrCodeAccountExpired, HTTPStatus: http.StatusForbidden, MsgKey: MsgAccountExpired}
	CodeCredExpired      = ErrCode{Code: ErrCodeCredentialExpired, HTTPStatus: http.StatusForbidden, MsgKey: MsgCredentialExpired}
	CodeTooManyAttempts  = ErrCode{Code: ErrCodeTooManyAttempts, HTTPStatus: http.StatusTooManyRequests, MsgKey: MsgTooManyLoginAttempts}
	CodeRegisterDisabled = ErrCode{Code: ErrCodeRegisterDisabled, HTTPStatus: http.StatusForbidden, MsgKey: MsgRegisterNotEnabled}
	CodeInvalidEmail     = ErrCode{Code: ErrCodeInvalidInput, HTTPStatus: http.StatusBadRequest, MsgKey: ErrInvalidEmail}
	CodeInvalidPhone     = ErrCode{Code: ErrCodeInvalidInput, HTTPStatus: http.StatusBadRequest, MsgKey: ErrInvalidPhoneFormat}

	// === 3xxx: 业务模块错误 ===
	CodeUsernameTaken                = ErrCode{Code: ErrCodeAlreadyExists, HTTPStatus: http.StatusConflict, MsgKey: ErrUsernameAlreadyTaken}
	CodeCurrentPasswordIncorrect     = ErrCode{Code: ErrCodeInvalidInput, HTTPStatus: http.StatusBadRequest, MsgKey: ErrCurrentPasswordIncorrect}
	CodeSettingKeyExists             = ErrCode{Code: ErrCodeAlreadyExists, HTTPStatus: http.StatusConflict, MsgKey: ErrSettingKeyExists}
	CodeInvalidSettingValue          = ErrCode{Code: ErrCodeInvalidInput, HTTPStatus: http.StatusBadRequest, MsgKey: ErrInvalidSettingValue}
	CodeSystemSettingDelete          = ErrCode{Code: ErrCodeForbidden, HTTPStatus: http.StatusForbidden, MsgKey: ErrSystemSettingDelete}
	CodeInvalidLoginMethodSetting    = ErrCode{Code: ErrCodeInvalidInput, HTTPStatus: http.StatusBadRequest, MsgKey: ErrInvalidLoginMethodSetting}
	CodeDictKeyExists                = ErrCode{Code: ErrCodeAlreadyExists, HTTPStatus: http.StatusConflict, MsgKey: ErrDictKeyExists}
	CodeDictItemValueExists          = ErrCode{Code: ErrCodeAlreadyExists, HTTPStatus: http.StatusConflict, MsgKey: ErrDictItemValueExists}
	CodeSystemDictDelete             = ErrCode{Code: ErrCodeForbidden, HTTPStatus: http.StatusForbidden, MsgKey: ErrSystemDictDelete}
	CodeResourceNotFound             = ErrCode{Code: ErrCodeNotFound, HTTPStatus: http.StatusNotFound, MsgKey: ErrResourceNotFound}
	CodeResourceCodeExists           = ErrCode{Code: ErrCodeAlreadyExists, HTTPStatus: http.StatusConflict, MsgKey: ErrResourceCodeExists}
	CodeInvalidResource              = ErrCode{Code: ErrCodeInvalidInput, HTTPStatus: http.StatusBadRequest, MsgKey: ErrInvalidResource}
	CodeResourcePermissionConflict   = ErrCode{Code: ErrCodeConflict, HTTPStatus: http.StatusConflict, MsgKey: ErrResourcePermissionConflict}
	CodeSystemResourceDelete         = ErrCode{Code: ErrCodeForbidden, HTTPStatus: http.StatusForbidden, MsgKey: ErrSystemResourceDelete}
	CodeRoleNotFound                 = ErrCode{Code: ErrCodeNotFound, HTTPStatus: http.StatusNotFound, MsgKey: ErrRoleNotFound}
	CodeRoleCodeExists               = ErrCode{Code: ErrCodeAlreadyExists, HTTPStatus: http.StatusConflict, MsgKey: ErrRoleCodeExists}
	CodeInvalidRole                  = ErrCode{Code: ErrCodeInvalidInput, HTTPStatus: http.StatusBadRequest, MsgKey: ErrInvalidRole}
	CodeRoleInUse                    = ErrCode{Code: ErrCodeConflict, HTTPStatus: http.StatusConflict, MsgKey: ErrRoleInUse}
	CodeSystemRoleDelete             = ErrCode{Code: ErrCodeForbidden, HTTPStatus: http.StatusForbidden, MsgKey: ErrSystemRoleDelete}
	CodeRolePermissionInvalid        = ErrCode{Code: ErrCodeInvalidInput, HTTPStatus: http.StatusBadRequest, MsgKey: ErrInvalidRolePermission}
	CodeRoleHierarchyCycle           = ErrCode{Code: ErrCodeConflict, HTTPStatus: http.StatusConflict, MsgKey: ErrRoleHierarchyCycle}
	CodeInvalidRoleHierarchy         = ErrCode{Code: ErrCodeInvalidInput, HTTPStatus: http.StatusBadRequest, MsgKey: ErrInvalidRoleHierarchy}
	CodeInvalidConstraint            = ErrCode{Code: ErrCodeInvalidInput, HTTPStatus: http.StatusBadRequest, MsgKey: ErrInvalidConstraint}
	CodeSSDViolation                 = ErrCode{Code: ErrCodeConflict, HTTPStatus: http.StatusConflict, MsgKey: ErrSSDViolation}
	CodeDSDViolation                 = ErrCode{Code: ErrCodeConflict, HTTPStatus: http.StatusConflict, MsgKey: ErrDSDViolation}
	CodeRoleNotAuthorized            = ErrCode{Code: ErrCodeForbidden, HTTPStatus: http.StatusForbidden, MsgKey: ErrRoleNotAuthorized}
	CodeAuthorizationSessionNotFound = ErrCode{Code: ErrCodeUnauthorized, HTTPStatus: http.StatusUnauthorized, MsgKey: ErrAuthorizationSessionNotFound}
	CodeAuthorizationSessionExpired  = ErrCode{Code: ErrCodeUnauthorized, HTTPStatus: http.StatusUnauthorized, MsgKey: ErrAuthorizationSessionExpired}
	CodeAuthorizationSessionRevoked  = ErrCode{Code: ErrCodeUnauthorized, HTTPStatus: http.StatusUnauthorized, MsgKey: ErrAuthorizationSessionRevoked}
	CodeCannotModifySystemUser       = ErrCode{Code: ErrCodeForbidden, HTTPStatus: http.StatusForbidden, MsgKey: ErrCannotModifySystemUser}
	CodeCannotModifyOwnStatus        = ErrCode{Code: ErrCodeForbidden, HTTPStatus: http.StatusForbidden, MsgKey: ErrCannotModifyOwnStatus}
	CodeCannotDeleteSystemUser       = ErrCode{Code: ErrCodeForbidden, HTTPStatus: http.StatusForbidden, MsgKey: ErrCannotDeleteSystemUser}
	CodeCannotDeleteOwnAccount       = ErrCode{Code: ErrCodeForbidden, HTTPStatus: http.StatusForbidden, MsgKey: ErrCannotDeleteOwnAccount}
	CodeNoFieldsToUpdate             = ErrCode{Code: ErrCodeBadRequest, HTTPStatus: http.StatusBadRequest, MsgKey: ErrNoFieldsToUpdate}
	CodeNameTooLong                  = ErrCode{Code: ErrCodeInvalidInput, HTTPStatus: http.StatusBadRequest, MsgKey: ErrNameTooLong}
	CodePasswordTooShort             = ErrCode{Code: ErrCodeInvalidInput, HTTPStatus: http.StatusBadRequest, MsgKey: ErrPasswordTooShort}

	// === 4xxx: 资产/文件相关错误 ===
	CodeAssetNotFound    = ErrCode{Code: ErrCodeAssetNotFound, HTTPStatus: http.StatusNotFound, MsgKey: ErrAssetNotFound}
	CodeAssetTooLarge    = ErrCode{Code: ErrCodeAssetTooLarge, HTTPStatus: http.StatusBadRequest, MsgKey: ErrFileTooLarge}
	CodeAssetInvalidType = ErrCode{Code: ErrCodeAssetInvalidType, HTTPStatus: http.StatusBadRequest, MsgKey: ErrInvalidFileType}

	// === 9xxx: 系统内部错误 ===
	CodeInternal = ErrCode{Code: ErrCodeInternalError, HTTPStatus: http.StatusInternalServerError, MsgKey: MsgInternalServerError}
)

// errCodeRegistry 统一错误码注册表（业务码 → ErrCode）
var errCodeRegistry = map[int]ErrCode{
	ErrCodeSuccess:            CodeOK,
	ErrCodeBadRequest:         CodeBadReq,
	ErrCodeInvalidInput:       CodeInvalidInput,
	ErrCodeNotFound:           CodeNotFoundErr,
	ErrCodeConflict:           CodeConflictErr,
	ErrCodeAlreadyExists:      CodeAlreadyExists,
	ErrCodeTooManyRequests:    CodeTooManyReqs,
	ErrCodeUnauthorized:       CodeUnauth,
	ErrCodeForbidden:          CodeForbid,
	ErrCodeInvalidCaptcha:     CodeInvalidCaptcha,
	ErrCodeInvalidCredentials: CodeInvalidCreds,
	ErrCodeInvalidConfirmCode: CodeInvalidConfirm,
	ErrCodeInvalidLoginMethod: CodeInvalidLogin,
	ErrCodeAccountDisabled:    CodeAcctDisabled,
	ErrCodeAccountLocked:      CodeAcctLocked,
	ErrCodeAccountExpired:     CodeAcctExpired,
	ErrCodeCredentialExpired:  CodeCredExpired,
	ErrCodeTooManyAttempts:    CodeTooManyAttempts,
	ErrCodeRegisterDisabled:   CodeRegisterDisabled,
	ErrCodeInternalError:      CodeInternal,
}

// LookupErrCode 根据业务错误码查找统一错误码定义
func LookupErrCode(code int) (ErrCode, bool) {
	ec, ok := errCodeRegistry[code]
	return ec, ok
}

// GetMsgKeyForErrCode 根据业务错误码获取 i18n 消息 key
func GetMsgKeyForErrCode(code int) string {
	if ec, ok := errCodeRegistry[code]; ok {
		return ec.MsgKey
	}
	return MsgInternalServerError
}

// GetHTTPStatusForErrCode 根据业务错误码获取 HTTP 状态码
func GetHTTPStatusForErrCode(code int) int {
	if ec, ok := errCodeRegistry[code]; ok {
		return ec.HTTPStatus
	}
	return http.StatusInternalServerError
}

var (
	CodeInvalidMenu     = ErrCode{Code: ErrCodeInvalidInput, HTTPStatus: http.StatusBadRequest, MsgKey: ErrInvalidMenu}
	CodeMenuNotFound    = ErrCode{Code: ErrCodeNotFound, HTTPStatus: http.StatusNotFound, MsgKey: ErrMenuNotFound}
	CodeMenuConflict    = ErrCode{Code: ErrCodeConflict, HTTPStatus: http.StatusConflict, MsgKey: ErrMenuConflict}
	CodeMenuHierarchy   = ErrCode{Code: ErrCodeConflict, HTTPStatus: http.StatusConflict, MsgKey: ErrMenuHierarchy}
	CodeMenuHasChildren = ErrCode{Code: ErrCodeConflict, HTTPStatus: http.StatusConflict, MsgKey: ErrMenuHasChildren}
	CodeMenuPermission  = ErrCode{Code: ErrCodeInvalidInput, HTTPStatus: http.StatusBadRequest, MsgKey: ErrMenuPermission}
)
