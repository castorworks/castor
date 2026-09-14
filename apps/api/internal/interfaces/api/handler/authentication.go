package handler

import (
	"fmt"

	"github.com/castorworks/castor/internal/application/dto"
	"github.com/castorworks/castor/internal/application/service"
	"github.com/castorworks/castor/internal/domain/audit_log"
	"github.com/castorworks/castor/internal/interfaces/api/response"
	"github.com/gin-gonic/gin"
)

// AuthenticationHandler 认证处理器
type AuthenticationHandler struct {
	AuthService     service.AuthService
	RsaService      service.RsaService
	AuditLogService service.AuditLogService
}

// NewAuthenticationHandler 创建认证处理器
func NewAuthenticationHandler(
	authService service.AuthService,
	rsaService service.RsaService,
	auditLogService service.AuditLogService,
) *AuthenticationHandler {
	return &AuthenticationHandler{
		AuthService:     authService,
		RsaService:      rsaService,
		AuditLogService: auditLogService,
	}
}

// GetCaptcha 获取图形验证码
func (h *AuthenticationHandler) GetCaptcha(c *gin.Context) {
	item, err := h.AuthService.GetCaptcha(c)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, item)
}

// PostCode 发送一次性验证码
func (h *AuthenticationHandler) PostCode(c *gin.Context) {
	req := &dto.ConfirmCodeReq{}
	if err := c.BindJSON(&req); err != nil {
		response.BadRequestErr(c, err)
		return
	}

	if err := h.AuthService.PostCode(c, req); err != nil {
		response.HandleError(c, err)
		return
	}
	response.SuccessI18n(c, gin.H{"res": "ok"}, response.InfCodeSent)
}

// Register 用户注册接口
func (h *AuthenticationHandler) Register(c *gin.Context) {
	req := &dto.UserRegisterReq{}

	if err := c.BindJSON(&req); err != nil {
		response.BadRequestErr(c, err)
		return
	}

	item, err := h.AuthService.Register(c, req)
	if err != nil {
		h.logAudit(c, audit_log.AuditLogTypeUserCreate, req.Username,
			fmt.Sprintf("User registration failed: %s", err.Error()), false)
		response.HandleError(c, err)
		return
	}
	h.logAudit(c, audit_log.AuditLogTypeUserCreate, req.Username,
		fmt.Sprintf("User registered: %s", req.Username), true)
	response.SuccessI18n(c, item, response.InfUserCreated)
}

// GetRsaPublicKey 获取 RSA 公钥
func (h *AuthenticationHandler) GetRsaPublicKey(c *gin.Context) {
	publicKey, err := h.RsaService.GetPublicKey(c)
	if err != nil {
		response.InternalServerError(c)
		return
	}
	response.Success(c, gin.H{"publicKey": publicKey})
}

// logAudit 记录审计日志（异步）- 公开接口
func (h *AuthenticationHandler) logAudit(c *gin.Context, logType audit_log.AuditLogType, target, details string, success bool) {
	h.AuditLogService.LogAsync(&audit_log.AuditLog{
		LogType:  logType,
		Operator: target, // 公开接口用目标用户名作为操作者
		Target:   target,
		Details:  details,
		IpAddr:   getClientIP(c),
		Success:  success,
	})
}
