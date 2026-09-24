package handler

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/castorworks/castor/internal/application/apperror"
	"github.com/castorworks/castor/internal/application/dto"
	"github.com/castorworks/castor/internal/application/service"
	"github.com/castorworks/castor/internal/domain/audit_log"
	"github.com/castorworks/castor/internal/domain/user"
	"github.com/castorworks/castor/internal/interfaces/api/response"
	"github.com/castorworks/castor/internal/pkg/constant"
	"github.com/castorworks/castor/internal/pkg/log"
	"github.com/castorworks/castor/internal/pkg/ucontext"
	"github.com/gin-gonic/gin"
)

// SessionStarter 为认证完成的用户建立会话并写入 cookie（由 JWT 中间件实现）
type SessionStarter interface {
	StartSession(c *gin.Context, u *user.User, remember bool, client service.LoginClient) error
}

const (
	// 站内默认落点
	ssoDefaultRedirect = "/dashboard/overview"
	ssoProfilePath     = "/dashboard/profile"
	ssoSignInPath      = "/auth/sign-in"
)

// SSOHandler OIDC 登录、回调与当前用户的账号关联
type SSOHandler struct {
	sso      service.SSOService
	mfa      service.MFAService
	login    *service.LoginService
	users    service.UserService
	sessions SessionStarter
	audit    service.AuditLogService
}

// NewSSOHandler 创建 OIDC 处理器
func NewSSOHandler(sso service.SSOService, mfa service.MFAService, login *service.LoginService, users service.UserService,
	sessions SessionStarter, audit service.AuditLogService) *SSOHandler {
	return &SSOHandler{sso: sso, mfa: mfa, login: login, users: users, sessions: sessions, audit: audit}
}

// safeRedirect 只接受站内相对路径（与前端 safeRedirectPath 同规则），否则用默认落点。
func safeRedirect(path, fallback string) string {
	if path == "" || !strings.HasPrefix(path, "/") || strings.HasPrefix(path, "//") || strings.ContainsAny(path, "\\\r\n\t") {
		return fallback
	}
	if u, err := url.Parse(path); err != nil || u.Scheme != "" || u.Host != "" {
		return fallback
	}
	return path
}

// withQuery 给站内路径追加查询参数
func withQuery(path string, values url.Values) string {
	sep := "?"
	if strings.Contains(path, "?") {
		sep = "&"
	}
	return path + sep + values.Encode()
}

// ssoErrorCode 回调失败时带回前端的错误标识（前端据此显示本地化提示）
func ssoErrorCode(err error) string {
	switch {
	case errors.Is(err, apperror.ErrOIDCStateInvalid):
		return "expired"
	case errors.Is(err, apperror.ErrOIDCAccountNotLinked):
		return "notLinked"
	case errors.Is(err, apperror.ErrOIDCIdentityInUse):
		return "identityInUse"
	case errors.Is(err, apperror.ErrOIDCProviderNotFound), errors.Is(err, apperror.ErrOIDCNotConfigured), errors.Is(err, apperror.ErrOIDCDiscoveryFailed):
		return "providerUnavailable"
	case errors.Is(err, apperror.ErrUserDisabled), errors.Is(err, apperror.ErrUserLocked), errors.Is(err, apperror.ErrAccountExpired):
		return "accountUnavailable"
	}
	return "failed"
}

// Providers 登录页可用的身份提供方
// GET /api/v1/auth/oidc/providers
func (h *SSOHandler) Providers(c *gin.Context) {
	providers, err := h.sso.EnabledProviders(c.Request.Context())
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, providers)
}

// Authorize 跳转到身份提供方登录
// GET /api/v1/auth/oidc/:provider/authorize?redirect=/dashboard&remember=true
func (h *SSOHandler) Authorize(c *gin.Context) {
	target, err := h.sso.Begin(c.Request.Context(), c.Param("provider"), service.SSOBegin{
		Mode:     service.SSOModeLogin,
		Redirect: safeRedirect(c.Query("redirect"), ssoDefaultRedirect),
		Remember: c.Query("remember") == "true",
	})
	if err != nil {
		log.Warn(c).Err(err).Str("provider", c.Param("provider")).Msg("OIDC authorize failed")
		c.Redirect(http.StatusFound, withQuery(ssoSignInPath, url.Values{"oidcError": {ssoErrorCode(err)}}))
		return
	}
	c.Redirect(http.StatusFound, target)
}

// Callback 身份提供方回调：登录（或关联）后重定向回站内
// GET /api/v1/auth/oidc/:provider/callback
func (h *SSOHandler) Callback(c *gin.Context) {
	client := service.LoginClient{IP: c.ClientIP(), UserAgent: c.GetHeader("User-Agent")}
	if idpError := c.Query("error"); idpError != "" {
		// 用户在身份提供方取消，或提供方拒绝了请求。
		log.Warn(c).Str("provider", c.Param("provider")).Str("error", idpError).Msg("OIDC provider returned an error")
		c.Redirect(http.StatusFound, withQuery(ssoSignInPath, url.Values{"oidcError": {"cancelled"}}))
		return
	}
	result, err := h.sso.Complete(c.Request.Context(), c.Param("provider"), c.Query("state"), c.Query("code"))
	if result != nil && result.Mode == service.SSOModeLink {
		h.finishLink(c, result, err)
		return
	}
	if err != nil {
		log.Warn(c).Err(err).Str("provider", c.Param("provider")).Msg("OIDC sign-in failed")
		if result != nil && result.User != nil {
			h.login.RecordLogin(result.User.Username, client, constant.LOGIN_METHOD_OIDC, false, result.User.ID)
		}
		c.Redirect(http.StatusFound, withQuery(ssoSignInPath, url.Values{"oidcError": {ssoErrorCode(err)}}))
		return
	}
	u := result.User
	if result.Created {
		// 注册已经在 Complete 里成功完成；失败的回调会走上面的登录失败记录。
		logAuditAs(c, h.audit, u.Username, u.ID, audit_log.AuditLogTypeUserCreate, u.Username,
			fmt.Sprintf("Register through identity provider %s", result.Provider.Code), err)
	}
	// 账号开启了两步验证：外部登录不绕过它，转到登录页完成第二步。
	enabled, err := h.mfa.IsEnabled(c.Request.Context(), u.ID)
	if err != nil {
		c.Redirect(http.StatusFound, withQuery(ssoSignInPath, url.Values{"oidcError": {"failed"}}))
		return
	}
	if enabled {
		challenge, err := h.mfa.StartLoginChallenge(c.Request.Context(), service.MFALoginChallenge{
			UserID: u.ID, Username: u.Username, Method: constant.LOGIN_METHOD_OIDC, Remember: result.Remember, IP: client.IP, UserAgent: client.UserAgent,
		})
		if err != nil {
			c.Redirect(http.StatusFound, withQuery(ssoSignInPath, url.Values{"oidcError": {"failed"}}))
			return
		}
		c.Redirect(http.StatusFound, withQuery(ssoSignInPath, url.Values{"mfaChallenge": {challenge}, "redirect": {result.Redirect}}))
		return
	}
	if err := h.sessions.StartSession(c, u, result.Remember, client); err != nil {
		log.Err(c, err).Msg("Failed to start a session after OIDC sign-in")
		c.Redirect(http.StatusFound, withQuery(ssoSignInPath, url.Values{"oidcError": {"failed"}}))
		return
	}
	h.login.RecordLogin(u.Username, client, constant.LOGIN_METHOD_OIDC, true, u.ID)
	c.Redirect(http.StatusFound, safeRedirect(result.Redirect, ssoDefaultRedirect))
}

func (h *SSOHandler) finishLink(c *gin.Context, result *service.SSOResult, err error) {
	var operator string
	var operatorID uint
	if result.User != nil {
		operator, operatorID = result.User.Username, result.User.ID
	}
	logAuditAs(c, h.audit, operator, operatorID, audit_log.AuditLogTypeIdentityLink, result.Provider.Code, "Link an identity provider account", err)
	redirect := safeRedirect(result.Redirect, ssoProfilePath)
	if err != nil {
		log.Warn(c).Err(err).Str("provider", result.Provider.Code).Msg("OIDC link failed")
		c.Redirect(http.StatusFound, withQuery(redirect, url.Values{"oidcError": {ssoErrorCode(err)}}))
		return
	}
	c.Redirect(http.StatusFound, withQuery(redirect, url.Values{"linked": {result.Provider.Code}}))
}

// Identities 当前用户在各身份提供方上的关联
// GET /api/v1/account/identities
func (h *SSOHandler) Identities(c *gin.Context) {
	items, err := h.sso.ListIdentities(c.Request.Context(), ucontext.GetUserID(c))
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, items)
}

// Link 发起关联：返回身份提供方的授权地址，前端把浏览器导航过去
// POST /api/v1/account/identities/:provider/link
func (h *SSOHandler) Link(c *gin.Context) {
	target, err := h.sso.Begin(c.Request.Context(), c.Param("provider"), service.SSOBegin{
		Mode: service.SSOModeLink, Redirect: ssoProfilePath, UserID: ucontext.GetUserID(c),
	})
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, dto.SSOLinkResp{AuthorizeURL: target})
}

// Unlink 解除关联
// DELETE /api/v1/account/identities/:provider
func (h *SSOHandler) Unlink(c *gin.Context) {
	u, err := h.users.GetRawByUsername(c.Request.Context(), ucontext.GetUsername(c))
	if err == nil {
		err = h.sso.Unlink(c.Request.Context(), u, c.Param("provider"))
	}
	logAudit(c, h.audit, audit_log.AuditLogTypeIdentityUnlink, c.Param("provider"), "Unlink an identity provider account", err)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.SuccessI18n(c, nil, response.InfIdentityUnlinked)
}

// AdminSSOHandler 身份提供方管理
type AdminSSOHandler struct {
	sso   service.SSOService
	audit service.AuditLogService
}

// NewAdminSSOHandler 创建身份提供方管理处理器
func NewAdminSSOHandler(sso service.SSOService, audit service.AuditLogService) *AdminSSOHandler {
	return &AdminSSOHandler{sso: sso, audit: audit}
}

// List 全部身份提供方
// GET /api/v1/admin/oidc-providers
func (h *AdminSSOHandler) List(c *gin.Context) {
	items, err := h.sso.ListProviders(c.Request.Context())
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, items)
}

// Create 新建身份提供方
// POST /api/v1/admin/oidc-providers
func (h *AdminSSOHandler) Create(c *gin.Context) { h.save(c, false) }

// Update 修改身份提供方（客户端密钥留空表示不变）
// PUT /api/v1/admin/oidc-providers/:id
func (h *AdminSSOHandler) Update(c *gin.Context) { h.save(c, true) }

func (h *AdminSSOHandler) save(c *gin.Context, update bool) {
	var req dto.OIDCProviderReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequestErr(c, err)
		return
	}
	var id uint
	logType := audit_log.AuditLogTypeOIDCProviderCreate
	if update {
		parsed, err := parseUintParam(c, "id")
		if err != nil {
			response.BadRequestErr(c, err)
			return
		}
		id, logType = parsed, audit_log.AuditLogTypeOIDCProviderUpdate
	}
	resp, err := h.sso.SaveProvider(c.Request.Context(), id, &req)
	// 审计只记录编码与启用状态，不写客户端密钥。
	logAudit(c, h.audit, logType, req.Code, fmt.Sprintf("Save identity provider %s (enabled=%t)", req.Code, req.IsEnabled), err)
	if err != nil {
		if errors.Is(err, apperror.ErrOIDCDiscoveryFailed) {
			log.Warn(c).Err(err).Str("issuer", req.Issuer).Msg("OIDC discovery failed")
		}
		response.HandleError(c, err)
		return
	}
	response.SuccessI18n(c, resp, response.InfOIDCProviderSaved)
}

// Delete 删除身份提供方（连同所有账号关联）
// DELETE /api/v1/admin/oidc-providers/:id
func (h *AdminSSOHandler) Delete(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.BadRequestErr(c, err)
		return
	}
	p, err := h.sso.DeleteProvider(c.Request.Context(), id)
	target := fmt.Sprintf("%d", id)
	if p != nil {
		target = p.Code
	}
	logAudit(c, h.audit, audit_log.AuditLogTypeOIDCProviderDelete, target, "Delete identity provider "+target, err)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.SuccessI18n(c, nil, response.InfOIDCProviderDeleted)
}
