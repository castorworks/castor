package handler

import (
	"errors"
	"net/http"

	"github.com/castorworks/castor/internal/application/apperror"
	"github.com/castorworks/castor/internal/application/dto"
	"github.com/castorworks/castor/internal/application/service"
	"github.com/castorworks/castor/internal/domain/audit_log"
	"github.com/castorworks/castor/internal/interfaces/api/response"
	"github.com/castorworks/castor/internal/pkg/log"
	"github.com/castorworks/castor/internal/pkg/ucontext"
	"github.com/gin-gonic/gin"
)

// AccountHandler 账户处理器
type AccountHandler struct {
	UserService       service.UserService
	AccountService    service.AccountService
	UserAvatarService service.UserAvatarService
	AuditLogService   service.AuditLogService
}

// NewAccountHandler 创建账户处理器
func NewAccountHandler(
	userService service.UserService,
	accountService service.AccountService,
	userAvatarService service.UserAvatarService,
	auditLogService service.AuditLogService,
) *AccountHandler {
	return &AccountHandler{
		UserService:       userService,
		AccountService:    accountService,
		UserAvatarService: userAvatarService,
		AuditLogService:   auditLogService,
	}
}

// GetUserInfo 获取当前用户信息
func (h *AccountHandler) GetUserInfo(c *gin.Context) {
	username := ucontext.GetUsername(c)

	item, err := h.UserService.GetByUsername(c, username)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	response.Success(c, item)
}

// PutUserName 更新用户显示名称
func (h *AccountHandler) PutUserName(c *gin.Context) {
	userId := ucontext.GetUserID(c)

	var req dto.UserNamePutReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequestErr(c, err)
		return
	}

	if err := h.AccountService.PutUserName(c, userId, req.Name); err != nil {
		response.HandleError(c, err)
		return
	}

	response.SuccessI18n(c, gin.H{}, response.InfUpdateSuccess)
}

// PutUserPassword 修改密码
func (h *AccountHandler) PutUserPassword(c *gin.Context) {
	userId := ucontext.GetUserID(c)
	username := ucontext.GetUsername(c)

	req := &dto.UserPasswordPutReq{}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequestErr(c, err)
		return
	}

	if err := h.AccountService.PutUserPassword(c, userId, req); err != nil {
		h.logAudit(c, audit_log.AuditLogTypePasswordChange, username,
			"Password change failed: "+passwordAuditReason(err), false)
		response.HandleError(c, err)
		return
	}

	h.logAudit(c, audit_log.AuditLogTypePasswordChange, username, "Password changed successfully", true)
	response.SuccessI18n(c, gin.H{}, response.InfPasswordChanged)
}

// PutUserPasswordByReset 重置密码
func (h *AccountHandler) PutUserPasswordByReset(c *gin.Context) {
	req := &dto.UserPasswordResetPutReq{}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequestErr(c, err)
		return
	}

	if err := h.AccountService.PutUserPasswordByReset(c, req); err != nil {
		h.logAuditPublic(c, audit_log.AuditLogTypePasswordReset, req.Username,
			"Password reset failed: "+passwordAuditReason(err), false)
		response.HandleError(c, err)
		return
	}

	h.logAuditPublic(c, audit_log.AuditLogTypePasswordReset, req.Username, "Password reset successfully", true)
	response.SuccessI18n(c, gin.H{}, response.InfPasswordReset)
}

// PostUserAvatar 上传用户头像
func (h *AccountHandler) PostUserAvatar(c *gin.Context) {
	f, fileHeader, err := c.Request.FormFile("file")
	if err != nil {
		response.BadRequestErr(c, err)
		return
	}
	defer f.Close()

	contentType := fileHeader.Header.Get("Content-Type")
	size := fileHeader.Size
	filename := fileHeader.Filename

	// 验证头像文件
	if err := h.UserAvatarService.ValidateFile(filename, contentType, size); err != nil {
		response.BadRequestErr(c, err)
		return
	}

	// 获取当前用户信息
	username := ucontext.GetUsername(c)
	dbUser, err := h.UserService.GetByUsername(c, username)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	// 上传新头像
	avatarResp, err := h.UserAvatarService.Upload(c.Request.Context(), filename, f, contentType, size)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	// 更新用户头像
	req := &dto.UserPutReq{
		Avatar: avatarResp.ObjectKey,
	}
	_, err = h.UserService.Put(c.Request.Context(), dbUser.ID, req)
	if err != nil {
		// 回滚：删除已上传的新头像
		if delErr := h.UserAvatarService.Delete(c.Request.Context(), avatarResp.ObjectKey); delErr != nil {
			log.Err(c, delErr).Str("objectKey", avatarResp.ObjectKey).Msg("Failed to roll back uploaded avatar")
		}
		response.HandleError(c, err)
		return
	}

	// 删除旧头像（在成功更新用户信息后）
	if dbUser.Avatar != "" {
		if delErr := h.UserAvatarService.Delete(c.Request.Context(), dbUser.Avatar); delErr != nil {
			log.Err(c, delErr).Str("objectKey", dbUser.Avatar).Msg("Failed to delete previous avatar")
		}
	}

	response.SuccessI18n(c, avatarResp, response.InfFileUploaded)
}

// GetUserAvatar 下载用户头像（公开接口）
func (h *AccountHandler) GetUserAvatar(c *gin.Context) {
	objectKey := c.Param("objectKey")
	if objectKey == "" {
		response.BadRequestI18n(c, response.ErrObjectKeyRequired)
		return
	}

	// 拼接完整的 objectKey（带 avatars/ 前缀）
	fullObjectKey := service.AvatarPrefix + objectKey

	if err := h.UserAvatarService.Download(c.Request.Context(), fullObjectKey, c.Writer); err != nil {
		if c.Writer.Written() {
			// 已开始写出内容，无法再返回错误响应
			log.Err(c, err).Str("objectKey", fullObjectKey).Msg("Avatar download interrupted")
			return
		}
		c.Status(http.StatusNotFound)
	}
}

// passwordAuditReason maps password change/reset failures to a fixed, non-sensitive
// reason. Raw error text may carry internal details (SQL, Redis, crypto errors) and must
// not be persisted in audit logs visible to administrators.
func passwordAuditReason(err error) string {
	switch {
	case errors.Is(err, apperror.ErrCurrentPasswordIncorrect):
		return "current password incorrect"
	case errors.Is(err, apperror.ErrInvalidConfirmCode):
		return "invalid confirm code"
	case errors.Is(err, apperror.ErrPasswordDecryptFailed):
		return "invalid password payload"
	default:
		return "internal error"
	}
}

// logAudit 记录审计日志（异步）- 需要登录
func (h *AccountHandler) logAudit(c *gin.Context, logType audit_log.AuditLogType, target, details string, success bool) {
	h.AuditLogService.LogAsync(&audit_log.AuditLog{
		LogType:    logType,
		Operator:   ucontext.GetUsername(c),
		OperatorID: ucontext.GetUserID(c),
		Target:     target,
		Details:    details,
		IpAddr:     getClientIP(c),
		Success:    success,
	})
}

// logAuditPublic 记录审计日志（异步）- 公开接口（无登录）
func (h *AccountHandler) logAuditPublic(c *gin.Context, logType audit_log.AuditLogType, target, details string, success bool) {
	h.AuditLogService.LogAsync(&audit_log.AuditLog{
		LogType:  logType,
		Operator: target, // 公开接口用目标用户名作为操作者
		Target:   target,
		Details:  details,
		IpAddr:   getClientIP(c),
		Success:  success,
	})
}
