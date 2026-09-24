package handler

import (
	"github.com/castorworks/castor/internal/application/dto"
	"github.com/castorworks/castor/internal/application/service"
	"github.com/castorworks/castor/internal/domain/audit_log"
	"github.com/castorworks/castor/internal/interfaces/api/response"
	"github.com/gin-gonic/gin"
)

// AuthenticationHandler 认证处理器
type AuthenticationHandler struct {
	AuthService     service.AuthService
	LoginService    *service.LoginService
	RsaService      service.RsaService
	AuditLogService service.AuditLogService
}

// NewAuthenticationHandler 创建认证处理器
func NewAuthenticationHandler(
	authService service.AuthService,
	loginService *service.LoginService,
	rsaService service.RsaService,
	auditLogService service.AuditLogService,
) *AuthenticationHandler {
	return &AuthenticationHandler{
		AuthService:     authService,
		LoginService:    loginService,
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

	if err := h.AuthService.PostCode(c.Request.Context(), req, c.ClientIP()); err != nil {
		response.HandleError(c, err)
		return
	}
	response.SuccessI18n(c, gin.H{"res": "ok"}, response.InfCodeSent)
}

// PutExpiredPassword 凭证过期的用户凭当前密码设置新密码（未登录调用）
func (h *AuthenticationHandler) PutExpiredPassword(c *gin.Context) {
	req := &dto.ExpiredPasswordPutReq{}
	if err := c.ShouldBindJSON(req); err != nil {
		response.BadRequestErr(c, err)
		return
	}
	err := h.LoginService.ChangeExpiredPassword(c, req)
	// 公开接口没有登录身份：以请求里的用户名作为操作者。
	logAuditAs(c, h.AuditLogService, req.Username, 0, audit_log.AuditLogTypePasswordChange, req.Username, "Change expired password", err)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.SuccessI18n(c, gin.H{}, response.InfPasswordChanged)
}

// Register 用户注册接口
func (h *AuthenticationHandler) Register(c *gin.Context) {
	req := &dto.UserRegisterReq{}
	if err := c.BindJSON(&req); err != nil {
		response.BadRequestErr(c, err)
		return
	}

	item, err := h.AuthService.Register(c, req)
	logAuditAs(c, h.AuditLogService, req.Username, 0, audit_log.AuditLogTypeUserCreate, req.Username, "Register "+req.Username, err)
	if err != nil {
		response.HandleError(c, err)
		return
	}
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
