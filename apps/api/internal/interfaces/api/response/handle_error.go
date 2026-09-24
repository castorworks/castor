package response

import (
	"errors"

	"github.com/castorworks/castor/internal/application/apperror"
	"github.com/gin-gonic/gin"
)

// errorMapping 定义 apperror 到 ErrCode 的映射关系
// 当 handler 层收到 service 返回的错误时，通过此映射自动选择正确的 HTTP 响应
type errorMapping struct {
	err     error
	errCode ErrCode
}

// errorMappings 注册所有已知的业务错误映射
// 顺序无关，匹配时使用 errors.Is 逐一比较
var errorMappings = []errorMapping{
	{apperror.ErrInvalidMenu, CodeInvalidMenu},
	{apperror.ErrMenuNotFound, CodeMenuNotFound},
	{apperror.ErrMenuConflict, CodeMenuConflict},
	{apperror.ErrMenuHierarchy, CodeMenuHierarchy},
	{apperror.ErrMenuHasChildren, CodeMenuHasChildren},
	{apperror.ErrMenuPermission, CodeMenuPermission},
	{apperror.ErrJobNotFound, CodeJobNotFound},
	{apperror.ErrOIDCProviderNotFound, CodeOIDCProviderNotFound},
	{apperror.ErrOIDCProviderConflict, CodeOIDCProviderConflict},
	{apperror.ErrInvalidOIDCProvider, CodeInvalidOIDCProvider},
	{apperror.ErrOIDCNotConfigured, CodeOIDCNotConfigured},
	{apperror.ErrOIDCDiscoveryFailed, CodeOIDCDiscoveryFailed},
	{apperror.ErrOIDCStateInvalid, CodeOIDCStateInvalid},
	{apperror.ErrOIDCAuthFailed, CodeOIDCAuthFailed},
	{apperror.ErrOIDCAccountNotLinked, CodeOIDCAccountNotLinked},
	{apperror.ErrOIDCIdentityInUse, CodeOIDCIdentityInUse},
	{apperror.ErrOIDCLastSignInMethod, CodeOIDCLastSignInMethod},
	{apperror.ErrTOTPRequired, CodeTOTPRequired},
	{apperror.ErrInvalidTOTPCode, CodeInvalidTOTPCode},
	{apperror.ErrMFAChallengeExpired, CodeMFAChallengeExpired},
	{apperror.ErrTOTPAlreadyEnabled, CodeTOTPAlreadyEnabled},
	{apperror.ErrTOTPNotEnabled, CodeTOTPNotEnabled},
	{apperror.ErrTOTPSetupNotStarted, CodeTOTPSetupNotStarted},
	{apperror.ErrUnsupportedFileFormat, CodeUnsupportedFileFormat},
	{apperror.ErrFileUnreadable, CodeFileUnreadable},
	{apperror.ErrImportEmpty, CodeImportEmpty},
	{apperror.ErrImportTooManyRows, CodeImportTooManyRows},
	{apperror.ErrImportMissingColumn, CodeImportMissingColumn},
	{apperror.ErrImportInvalidRows, CodeImportInvalidRows},
	{apperror.ErrExportTooLarge, CodeExportTooLarge},
	{apperror.ErrInvalidCron, CodeInvalidCron},
	{apperror.ErrJobAlreadyRunning, CodeJobAlreadyRunning},
	{apperror.ErrDepartmentNotFound, CodeDepartmentNotFound},
	{apperror.ErrInvalidDepartment, CodeInvalidDepartment},
	{apperror.ErrDepartmentConflict, CodeDepartmentConflict},
	{apperror.ErrDepartmentHierarchy, CodeDepartmentHierarchy},
	{apperror.ErrDepartmentHasChildren, CodeDepartmentHasChildren},
	{apperror.ErrDepartmentHasMembers, CodeDepartmentHasMembers},
	{apperror.ErrInvalidDataScope, CodeInvalidDataScope},
	{apperror.ErrDataScopeExceedsCaller, CodeDataScopeExceedsCaller},
	// 通用错误
	{apperror.ErrRecordNotFound, CodeNotFoundErr},
	{apperror.ErrInvalidOrder, ErrCode{Code: ErrCodeInvalidInput, HTTPStatus: CodeBadRequest, MsgKey: ErrInvalidOrder}},
	{apperror.ErrTooManyRequests, CodeTooManyReqs},

	// 认证相关
	{apperror.ErrRegisterNotEnabled, CodeRegisterDisabled},
	{apperror.ErrInvalidConfirmCode, CodeInvalidConfirm},
	{apperror.ErrUsernameAlreadyTaken, CodeUsernameTaken},
	{apperror.ErrInvalidCaptchaCode, CodeInvalidCaptcha},
	{apperror.ErrInvalidCodeType, ErrCode{Code: ErrCodeInvalidInput, HTTPStatus: CodeBadRequest, MsgKey: ErrInvalidCodeType}},
	{apperror.ErrInvalidEmailFormat, CodeInvalidEmail},
	{apperror.ErrInvalidPhoneFormat, CodeInvalidPhone},
	{apperror.ErrLoginMethodDisabled, CodeLoginDisabled},
	{apperror.ErrDeliveryChannelNotConfigured, CodeDeliveryNotConfigured},

	// 用户状态
	{apperror.ErrUserDisabled, CodeAcctDisabled},
	{apperror.ErrUserLocked, CodeAcctLocked},
	{apperror.ErrAccountExpired, CodeAcctExpired},
	{apperror.ErrCredentialExpired, CodeCredExpired},
	{apperror.ErrTooManyLoginAttempts, CodeTooManyAttempts},
	{apperror.ErrInvalidLoginCredential, CodeInvalidCreds},

	// 账户相关
	{apperror.ErrCurrentPasswordIncorrect, CodeCurrentPasswordIncorrect},
	{apperror.ErrInvalidContactType, CodeInvalidContactType},
	{apperror.ErrContactAlreadyUsed, CodeContactAlreadyUsed},
	{apperror.ErrPasswordEncryptFailed, ErrCode{Code: ErrCodeInternalError, HTTPStatus: CodeInternalServerError, MsgKey: ErrPasswordEncryptFailed}},
	{apperror.ErrPasswordDecryptFailed, ErrCode{Code: ErrCodeInternalError, HTTPStatus: CodeInternalServerError, MsgKey: ErrPasswordDecryptFailed}},

	// 配置相关
	{apperror.ErrSettingKeyExists, CodeSettingKeyExists},
	{apperror.ErrInvalidSettingValue, CodeInvalidSettingValue},
	{apperror.ErrSystemSettingDelete, CodeSystemSettingDelete},
	{apperror.ErrInvalidLoginMethodSetting, CodeInvalidLoginMethodSetting},

	// 字典相关
	{apperror.ErrDictTypeCodeExists, CodeDictKeyExists},
	{apperror.ErrDictItemValueExists, CodeDictItemValueExists},
	{apperror.ErrSystemDictTypeDelete, CodeSystemDictDelete},
	{apperror.ErrSystemDictItemDelete, CodeSystemDictDelete},
	{apperror.ErrSystemDictItemValueLocked, CodeSystemDictItemValueLocked},
	{apperror.ErrDictItemColorInvalid, CodeDictItemColorInvalid},

	// 通知相关
	{apperror.ErrNotificationRecipientsRequired, ErrCode{Code: ErrCodeInvalidInput, HTTPStatus: CodeBadRequest, MsgKey: ErrNotificationRecipientsRequired}},
	{apperror.ErrNotificationNotDelivered, ErrCode{Code: ErrCodeForbidden, HTTPStatus: CodeForbidden, MsgKey: ErrNotificationNotDelivered}},

	// 权限相关
	{apperror.ErrResourceNotFound, CodeResourceNotFound},
	{apperror.ErrResourceCodeExists, CodeResourceCodeExists},
	{apperror.ErrInvalidResource, CodeInvalidResource},
	{apperror.ErrResourcePermissionConflict, CodeResourcePermissionConflict},
	{apperror.ErrSystemResourceDelete, CodeSystemResourceDelete},
	{apperror.ErrRoleNotFound, CodeRoleNotFound},
	{apperror.ErrRoleCodeExists, CodeRoleCodeExists},
	{apperror.ErrInvalidRole, CodeInvalidRole},
	{apperror.ErrRoleInUse, CodeRoleInUse},
	{apperror.ErrSystemRoleDelete, CodeSystemRoleDelete},
	{apperror.ErrRolePermissionInvalid, CodeRolePermissionInvalid},
	{apperror.ErrRoleHierarchyCycle, CodeRoleHierarchyCycle},
	{apperror.ErrInvalidRoleHierarchy, CodeInvalidRoleHierarchy},
	{apperror.ErrInvalidConstraint, CodeInvalidConstraint},
	{apperror.ErrSSDViolation, CodeSSDViolation},
	{apperror.ErrDSDViolation, CodeDSDViolation},
	{apperror.ErrRoleNotAuthorized, CodeRoleNotAuthorized},
	{apperror.ErrAuthorizationSessionNotFound, CodeAuthorizationSessionNotFound},
	{apperror.ErrAuthorizationSessionExpired, CodeAuthorizationSessionExpired},
	{apperror.ErrAuthorizationSessionRevoked, CodeAuthorizationSessionRevoked},

	// 用户管理保护
	{apperror.ErrCannotModifySystemUser, CodeCannotModifySystemUser},
	{apperror.ErrCannotModifySystemRole, CodeCannotModifySystemRole},
	{apperror.ErrCannotModifySystemResource, CodeCannotModifySystemResource},
	{apperror.ErrGrantExceedsCaller, CodeGrantExceedsCaller},
	{apperror.ErrTargetUserExceedsCaller, CodeTargetUserExceedsCaller},
	{apperror.ErrCannotModifyOwnStatus, CodeCannotModifyOwnStatus},
	{apperror.ErrCannotDeleteSystemUser, CodeCannotDeleteSystemUser},
	{apperror.ErrCannotDeleteOwnAccount, CodeCannotDeleteOwnAccount},

	// 用户管理验证
	{apperror.ErrNoFieldsToUpdate, CodeNoFieldsToUpdate},
	{apperror.ErrNameTooLong, CodeNameTooLong},
	{apperror.ErrPasswordTooShort, CodePasswordTooShort},
	{apperror.ErrPasswordTooLong, CodePasswordTooLong},
	{apperror.ErrPasswordTooWeak, CodePasswordTooWeak},
	{apperror.ErrPasswordUnchanged, CodePasswordUnchanged},
	{apperror.ErrCredentialNotExpired, CodeCredentialNotExpired},

	// 资产相关
	{apperror.ErrAssetNotFound, CodeAssetNotFound},
	{apperror.ErrAssetTooLarge, CodeAssetTooLarge},
	{apperror.ErrAssetInvalidType, CodeAssetInvalidType},
	{apperror.ErrAssetInUse, CodeAssetInUse},
}

// HandleError 根据错误类型自动选择正确的 HTTP 响应
// 如果错误匹配已注册的业务错误，返回对应的 HTTP 状态码和 i18n 消息
// 如果无法匹配，返回 500 Internal Server Error
//
// 用法示例：
//
//	if err := svc.DoSomething(ctx); err != nil {
//	    response.HandleError(c, err)
//	    return
//	}
func HandleError(c *gin.Context, err error) {
	if err == nil {
		return
	}

	for _, m := range errorMappings {
		if errors.Is(err, m.err) {
			c.JSON(m.errCode.HTTPStatus, wrapErrorI18n(c, nil, m.errCode))
			return
		}
	}

	// 未匹配到已知错误，返回 500
	InternalServerErrorErr(c, err)
}

// wrapErrorI18n 在 { code, data, message } 之外附带业务错误码 errorCode：HTTP 状态码
// 只能区分大类，前端要据 errorCode 判断"需要验证码""凭证已过期"等具体情形。
func wrapErrorI18n(c *gin.Context, data any, errCode ErrCode) map[string]any {
	result := wrapResultI18n(c, data, errCode.MsgKey)
	result["errorCode"] = errCode.Code
	return result
}

// KnownError 返回 err 对应的已登记业务错误（哨兵），未登记的返回 nil。
// 哨兵的文本是固定的、不含内部细节，可以安全地写进审计日志；包装它的错误链则不一定。
func KnownError(err error) error {
	for _, m := range errorMappings {
		if errors.Is(err, m.err) {
			return m.err
		}
	}
	return nil
}

// BusinessErrorCode 返回 i18n 消息 key 对应的业务错误码，供不经过 HandleError 的出口
// （如 JWT 中间件的 Unauthorized）同样附带 errorCode；未登记的 key 返回 0。
func BusinessErrorCode(msgKey string) int {
	for _, m := range errorMappings {
		if m.errCode.MsgKey == msgKey {
			return m.errCode.Code
		}
	}
	for _, errCode := range errCodeRegistry {
		if errCode.MsgKey == msgKey {
			return errCode.Code
		}
	}
	return 0
}

// HandleErrorWithData 与 HandleError 类似，但允许在错误响应中附带额外数据
func HandleErrorWithData(c *gin.Context, err error, data any) {
	if err == nil {
		return
	}

	for _, m := range errorMappings {
		if errors.Is(err, m.err) {
			c.JSON(m.errCode.HTTPStatus, wrapErrorI18n(c, data, m.errCode))
			return
		}
	}

	InternalServerErrorErr(c, err)
}
