package handler

import (
	"fmt"
	"strings"

	"github.com/castorworks/castor/internal/application/dto"
	"github.com/castorworks/castor/internal/application/service"
	"github.com/castorworks/castor/internal/domain/audit_log"
	"github.com/castorworks/castor/internal/interfaces/api/response"
	"github.com/castorworks/castor/internal/pkg/ucontext"
	"github.com/castorworks/castor/internal/pkg/validator"
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

// PutNotificationPreferences 通知偏好（是否接收通知邮件）
// PUT /api/v1/account/notification-preferences
func (h *AccountHandler) PutNotificationPreferences(c *gin.Context) {
	var req dto.NotificationPreferencesPutReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequestErr(c, err)
		return
	}
	err := h.AccountService.SetNotificationEmailMute(c, ucontext.GetUserID(c), req.MuteNotificationEmails)
	logAudit(c, h.AuditLogService, audit_log.AuditLogTypeProfileUpdate, ucontext.GetUsername(c),
		fmt.Sprintf("Set notification emails muted=%t", req.MuteNotificationEmails), err)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.SuccessI18n(c, gin.H{}, response.InfUpdateSuccess)
}

// PutUserName 更新用户显示名称
func (h *AccountHandler) PutUserName(c *gin.Context) {
	var req dto.UserNamePutReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequestErr(c, err)
		return
	}

	err := h.AccountService.PutUserName(c, ucontext.GetUserID(c), req.Name)
	// 显示名由客户端自由填写，审计只记录“改过名”，不写入具体内容。
	logAudit(c, h.AuditLogService, audit_log.AuditLogTypeProfileUpdate, ucontext.GetUsername(c), "Update display name", err)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.SuccessI18n(c, gin.H{}, response.InfUpdateSuccess)
}

// PutUserPassword 修改密码
func (h *AccountHandler) PutUserPassword(c *gin.Context) {
	req := &dto.UserPasswordPutReq{}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequestErr(c, err)
		return
	}

	err := h.AccountService.PutUserPassword(c, ucontext.GetUserID(c), req)
	logAudit(c, h.AuditLogService, audit_log.AuditLogTypePasswordChange, ucontext.GetUsername(c), "Change password", err)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.SuccessI18n(c, gin.H{}, response.InfPasswordChanged)
}

// PutUserPasswordByReset 重置密码
func (h *AccountHandler) PutUserPasswordByReset(c *gin.Context) {
	req := &dto.UserPasswordResetPutReq{}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequestErr(c, err)
		return
	}

	err := h.AccountService.PutUserPasswordByReset(c, req)
	// 公开接口没有登录身份：以请求里的用户名作为操作者。
	logAuditAs(c, h.AuditLogService, req.Username, 0, audit_log.AuditLogTypePasswordReset, req.Username, "Reset password", err)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.SuccessI18n(c, gin.H{}, response.InfPasswordReset)
}

// PostContactCode 向待绑定的邮箱/手机号发送验证码
func (h *AccountHandler) PostContactCode(c *gin.Context) {
	userId := ucontext.GetUserID(c)

	req := &dto.ContactCodePostReq{}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequestErr(c, err)
		return
	}

	if err := h.AccountService.PostContactCode(c.Request.Context(), userId, req); err != nil {
		response.HandleError(c, err)
		return
	}

	response.SuccessI18n(c, gin.H{}, response.InfCodeSent)
}

// PutContact 绑定邮箱/手机号
func (h *AccountHandler) PutContact(c *gin.Context) {
	req := &dto.ContactPutReq{}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequestErr(c, err)
		return
	}

	err := h.AccountService.PutContact(c.Request.Context(), ucontext.GetUserID(c), req)
	// 审计详情只记录联系方式类型，不记录邮箱/手机号本身。
	logAudit(c, h.AuditLogService, audit_log.AuditLogTypeProfileUpdate, ucontext.GetUsername(c), "Bind contact: "+contactAuditType(req.ContactType), err)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.SuccessI18n(c, gin.H{}, response.InfContactBound)
}

// PostUserAvatar 上传用户头像
func (h *AccountHandler) PostUserAvatar(c *gin.Context) {
	f, fileHeader, err := c.Request.FormFile("file")
	if err != nil {
		response.BadRequestErr(c, err)
		return
	}
	defer f.Close()

	filename := fileHeader.Filename
	if !validator.IsValidFilename(filename) {
		response.BadRequestI18n(c, response.ErrInvalidFilename)
		return
	}

	avatarResp, err := h.UserAvatarService.Upload(c.Request.Context(), ucontext.GetUserID(c),
		filename, f, fileHeader.Header.Get("Content-Type"), fileHeader.Size)
	// 原始文件名是任意的客户端输入，审计记录服务端生成的对象键。
	details := "Upload avatar"
	if err == nil {
		details += ": " + avatarResp.ObjectKey
	}
	logAudit(c, h.AuditLogService, audit_log.AuditLogTypeAvatarUpdate, ucontext.GetUsername(c), details, err)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.SuccessI18n(c, avatarResp, response.InfFileUploaded)
}

// contactAuditType 把客户端传入的联系方式类型收敛为固定取值，
// 避免把任意请求文本写进审计日志。
func contactAuditType(contactType string) string {
	switch strings.ToLower(strings.TrimSpace(contactType)) {
	case service.ContactTypeEmail:
		return service.ContactTypeEmail
	case service.ContactTypeMobile:
		return service.ContactTypeMobile
	default:
		return "unknown"
	}
}
