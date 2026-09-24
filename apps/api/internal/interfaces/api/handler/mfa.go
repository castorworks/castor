package handler

import (
	"fmt"

	"github.com/castorworks/castor/internal/application/dto"
	"github.com/castorworks/castor/internal/application/service"
	"github.com/castorworks/castor/internal/domain/audit_log"
	"github.com/castorworks/castor/internal/interfaces/api/response"
	"github.com/castorworks/castor/internal/pkg/ucontext"
	"github.com/gin-gonic/gin"
)

// AccountMFAHandler 当前用户的两步验证设置
type AccountMFAHandler struct {
	mfa   service.MFAService
	users service.UserService
	audit service.AuditLogService
}

// NewAccountMFAHandler 创建两步验证处理器
func NewAccountMFAHandler(mfa service.MFAService, users service.UserService, audit service.AuditLogService) *AccountMFAHandler {
	return &AccountMFAHandler{mfa: mfa, users: users, audit: audit}
}

// Status 两步验证状态
// GET /api/v1/account/totp
func (h *AccountMFAHandler) Status(c *gin.Context) {
	status, err := h.mfa.Status(c.Request.Context(), ucontext.GetUserID(c))
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, status)
}

// BeginSetup 生成新的 TOTP 密钥与二维码（尚未启用）
// POST /api/v1/account/totp/setup
func (h *AccountMFAHandler) BeginSetup(c *gin.Context) {
	u, err := h.users.GetRawByUsername(c.Request.Context(), ucontext.GetUsername(c))
	if err != nil {
		response.HandleError(c, err)
		return
	}
	setup, err := h.mfa.BeginTOTPSetup(c.Request.Context(), u)
	logAudit(c, h.audit, audit_log.AuditLogTypeMFASetup, ucontext.GetUsername(c), "Start authenticator setup", err)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, setup)
}

// Enable 用验证器上的码确认绑定，返回恢复码（只显示这一次）
// POST /api/v1/account/totp/enable
func (h *AccountMFAHandler) Enable(c *gin.Context) {
	var req dto.TOTPCodeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequestErr(c, err)
		return
	}
	codes, err := h.mfa.EnableTOTP(c.Request.Context(), ucontext.GetUserID(c), req.Code)
	logAudit(c, h.audit, audit_log.AuditLogTypeMFAEnable, ucontext.GetUsername(c), "Enable two-factor authentication", err)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.SuccessI18n(c, codes, response.InfTOTPEnabled)
}

// Disable 关闭两步验证（当前密码 + 一次性码或恢复码）
// POST /api/v1/account/totp/disable
func (h *AccountMFAHandler) Disable(c *gin.Context) {
	var req dto.TOTPDisableReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequestErr(c, err)
		return
	}
	u, err := h.users.GetRawByUsername(c.Request.Context(), ucontext.GetUsername(c))
	if err != nil {
		response.HandleError(c, err)
		return
	}
	err = h.mfa.DisableTOTP(c.Request.Context(), u, &req)
	logAudit(c, h.audit, audit_log.AuditLogTypeMFADisable, ucontext.GetUsername(c), "Disable two-factor authentication", err)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.SuccessI18n(c, nil, response.InfTOTPDisabled)
}

// RegenerateRecoveryCodes 作废旧恢复码并生成新的一组
// POST /api/v1/account/totp/recovery-codes
func (h *AccountMFAHandler) RegenerateRecoveryCodes(c *gin.Context) {
	var req dto.TOTPCodeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequestErr(c, err)
		return
	}
	codes, err := h.mfa.RegenerateRecoveryCodes(c.Request.Context(), ucontext.GetUserID(c), req.Code)
	logAudit(c, h.audit, audit_log.AuditLogTypeMFARecoveryCodes, ucontext.GetUsername(c), "Regenerate recovery codes", err)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.SuccessI18n(c, codes, response.InfRecoveryCodesRegenerated)
}

// ResetTOTP 管理员清除用户的两步验证（用户丢失了验证器与恢复码）。
// 与其他"管理他人账号"的操作一样，需要目标在数据范围内且权限不超过调用者。
// DELETE /api/v1/admin/users/:id/totp
func (h *AdminUserHandler) ResetTOTP(c *gin.Context) {
	scope, ok := requestScope(c, h.RBAC)
	if !ok {
		return
	}
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.BadRequestErr(c, err)
		return
	}
	err = h.resetTOTP(c, scope, id)
	logAudit(c, h.AuditLogService, audit_log.AuditLogTypeMFAReset, fmt.Sprintf("%d", id), fmt.Sprintf("Reset two-factor authentication of user %d", id), err)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.SuccessI18n(c, nil, response.InfTOTPReset)
}
