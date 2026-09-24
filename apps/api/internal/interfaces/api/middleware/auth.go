package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	jwt "github.com/appleboy/gin-jwt/v2"
	"github.com/castorworks/castor/internal/application/apperror"
	"github.com/castorworks/castor/internal/application/dto"
	"github.com/castorworks/castor/internal/application/service"
	"github.com/castorworks/castor/internal/domain/audit_log"
	"github.com/castorworks/castor/internal/domain/user"
	"github.com/castorworks/castor/internal/infrastructure/config"
	"github.com/castorworks/castor/internal/interfaces/api/response"
	"github.com/castorworks/castor/internal/pkg/constant"
	"github.com/castorworks/castor/internal/pkg/log"
	"github.com/castorworks/castor/internal/pkg/ucontext"
	gi18n "github.com/gin-contrib/i18n"
	"github.com/gin-gonic/gin"
	"github.com/hyperits/gosuite/logger"
)

// JwtMiddleware JWT 中间件
type JwtMiddleware struct {
	AuthMiddleware        *jwt.GinJWTMiddleware
	LoginService          *service.LoginService
	RBACService           service.RBACService
	TokenBlacklistService service.TokenBlacklistService
	AuditLogService       service.AuditLogService
	MFAService            service.MFAService
	// audience 是本实例签发、也只接受的 token 受众（General.InstanceID）
	audience string
}

// jwtRememberClaim 标记令牌所属登录是否勾选了"记住我"，刷新时沿用，决定 cookie 是否跨浏览器重启保留。
const jwtRememberClaim = "rem"

// LoginRequest 是登录接口的请求体（导出供 OpenAPI 文档引用）：认证字段交给 LoginService，"记住我"只影响 cookie 的保存方式。
type LoginRequest struct {
	service.LoginReq
	RememberMe bool `json:"rememberMe"`
}

// tokenSubject 是签发令牌的主体：用户身份加上该登录的"记住我"选择。
type tokenSubject struct {
	*user.User
	Remember bool
}

// authCookieMaxAge 决定 jwt 与 csrf_token cookie 的保存方式。cookie 必须活得和刷新窗口一样久，
// 否则访问令牌一过期 cookie 就被浏览器丢弃，刷新接口拿不到旧令牌，MaxRefresh 形同虚设。
//   - 记住我：持久 cookie，保存到刷新窗口结束，浏览器重启后仍可静默刷新；
//   - 否则：会话 cookie（不设 Max-Age），关闭浏览器即失效。
func authCookieMaxAge(remember bool) int {
	if !remember {
		return 0
	}
	return int(time.Duration(config.C.General.JwtMaxRefreshHours) * time.Hour / time.Second)
}

func setAuthCookies(c *gin.Context, token string, remember bool) {
	maxAge := authCookieMaxAge(remember)
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("jwt", token, maxAge, "/", "", !config.C.General.Development, true)
	setCSRFCookie(c, maxAge)
}

// NewJwtMiddleware 创建 JWT 中间件
func NewJwtMiddleware(
	loginService *service.LoginService,
	rbacService service.RBACService,
	tokenBlacklistService service.TokenBlacklistService,
	auditLogService service.AuditLogService,
	mfaService service.MFAService,
) (*JwtMiddleware, error) {
	// 多套部署即使误用了同一个 JwtKey，一套签发的 token 也不能在另一套上冒充同 ID 的用户。
	audience := config.C.General.InstanceID

	authMiddleware, err := jwt.New(&jwt.GinJWTMiddleware{
		Realm:       "castor",
		Key:         []byte(config.C.General.JwtKey),
		Timeout:     time.Duration(config.C.General.JwtTimeoutHours) * time.Hour,
		MaxRefresh:  time.Duration(config.C.General.JwtMaxRefreshHours) * time.Hour,
		IdentityKey: constant.JWT_IDENTITY_KEY,
		PayloadFunc: func(data interface{}) jwt.MapClaims {
			remember := false
			if subject, ok := data.(*tokenSubject); ok {
				data, remember = subject.User, subject.Remember
			}
			if v, ok := data.(*user.User); ok {
				return jwt.MapClaims{
					constant.JWT_IDENTITY_KEY:           v.ID,
					constant.JWT_NAME:                   v.Name,
					constant.JWT_USERNAME:               v.Username,
					constant.JWT_ACCOUNT_SOURCE:         v.AccountSource,
					constant.JWT_REFRESH_TOKEN_DURATION: config.C.General.JwtMaxRefreshHours,
					constant.JWT_AUTHORIZATION_SESSION:  v.AuthorizationSessionID,
					// A unique token ID keeps every issued token distinct, so retiring a
					// refreshed token can never collide with its replacement.
					"jti":            newTokenID(),
					"aud":            audience,
					jwtRememberClaim: remember,
				}
			}
			return jwt.MapClaims{}
		},
		IdentityHandler: func(c *gin.Context) interface{} {
			if u := identityFromClaims(jwt.ExtractClaims(c), audience); u != nil {
				return u
			}
			return nil
		},
		LoginResponse: func(c *gin.Context, _ int, token string, expire time.Time) {
			setAuthCookies(c, token, c.GetBool("login_remember"))

			// 构建包含用户信息的登录响应
			loginData := gin.H{"expire": expire}

			// 从 context 中获取登录用户信息（由 Authenticator 设置）
			if u, exists := c.Get("login_user"); exists {
				if loginUser, ok := u.(*user.User); ok {
					access, _ := rbacService.GetSessionAccess(c.Request.Context(), loginUser.AuthorizationSessionID, loginUser.ID)
					if access == nil {
						response.InternalServerErrorErr(c, errors.New("authorization session unavailable"))
						return
					}
					loginData["user"] = gin.H{
						"id":                     loginUser.ID,
						"username":               loginUser.Username,
						"name":                   loginUser.Name,
						"avatar":                 loginUser.Avatar,
						"assignedRoles":          access.AssignedRoles,
						"authorizedRoles":        access.AuthorizedRoles,
						"activeRoles":            access.ActiveRoles,
						"permissions":            access.Permissions,
						"authorizationSessionId": loginUser.AuthorizationSessionID,
					}
				}
			}

			response.SuccessI18n(c, loginData, response.InfLoginSuccess)
		},
		RefreshResponse: func(c *gin.Context, _ int, token string, expire time.Time) {
			setAuthCookies(c, token, c.GetBool("login_remember"))
			response.Success(c, gin.H{"expire": expire})
		},
		Authenticator: func(c *gin.Context) (interface{}, error) {
			var req LoginRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				log.Err(c, err).Msg("Failed to bind login request")
				return nil, jwt.ErrMissingLoginValues
			}
			client := service.LoginClient{IP: c.ClientIP(), UserAgent: c.GetHeader("User-Agent")}
			method := c.Query("method")
			u, err := loginService.Login(c, method, &req.LoginReq, client)
			if err != nil {
				return nil, err
			}
			// 开启了两步验证：第一重凭证通过后只发挑战令牌，不签发会话。
			enabled, err := mfaService.IsEnabled(c.Request.Context(), u.ID)
			if err != nil {
				return nil, err
			}
			if enabled {
				challenge, err := mfaService.StartLoginChallenge(c.Request.Context(), service.MFALoginChallenge{
					UserID: u.ID, Username: u.Username, Method: method, Remember: req.RememberMe, IP: client.IP, UserAgent: client.UserAgent,
				})
				if err != nil {
					return nil, err
				}
				c.Set(mfaChallengeContextKey, challenge)
				return nil, apperror.ErrTOTPRequired
			}
			if err := prepareLogin(c, rbacService, u, req.RememberMe, client); err != nil {
				return nil, err
			}
			loginService.RecordLogin(u.Username, client, method, true, u.ID)
			return &tokenSubject{User: u, Remember: req.RememberMe}, nil
		},
		Authorizator: func(data interface{}, c *gin.Context) bool {
			// 检查 Token 是否在黑名单中
			token := jwt.GetToken(c)
			if tokenBlacklistService.IsBlacklisted(c.Request.Context(), token) {
				log.Warn(c).Msg("Token is blacklisted")
				return false
			}

			u, ok := data.(*user.User)
			if !ok {
				return false
			}
			_, err := rbacService.GetSessionAccess(c.Request.Context(), u.AuthorizationSessionID, u.ID)
			return err == nil
		},
		Unauthorized: func(c *gin.Context, code int, message string) {
			log.Warn(c).
				Int("code", code).
				Str("message", message).
				Msg("Unauthorized")

			msgKey := jwtUnauthorizedMessageKey(message)
			statusCode := jwtUnauthorizedStatus(code, msgKey)

			// 需要两步验证：把挑战令牌交给客户端，用它提交验证码。
			var data any
			if challenge := c.GetString(mfaChallengeContextKey); challenge != "" {
				data = dto.MFAChallengeResp{Challenge: challenge, ExpiresIn: int(mfaChallengeTTLSeconds)}
			}
			c.JSON(statusCode, gin.H{
				"code":      statusCode,
				"errorCode": unauthorizedErrorCode(msgKey),
				"data":      data,
				"message":   gi18n.MustGetMessage(c, msgKey),
			})
		},
		TokenLookup:   "header: Authorization, cookie: jwt",
		TokenHeadName: "Bearer",
		TimeFunc:      time.Now,
	})

	if err != nil {
		logger.Errorf("Failed to create jwt middleware: %v", err)
		return nil, err
	}

	return &JwtMiddleware{
		AuthMiddleware:        authMiddleware,
		LoginService:          loginService,
		RBACService:           rbacService,
		TokenBlacklistService: tokenBlacklistService,
		AuditLogService:       auditLogService,
		MFAService:            mfaService,
		audience:              audience,
	}, nil
}

const (
	// mfaChallengeContextKey Authenticator 把挑战令牌交给 Unauthorized 的 context 键
	mfaChallengeContextKey = "mfa_challenge"
	// mfaChallengeTTLSeconds 与 service 里挑战的有效期一致，告诉客户端还剩多久
	mfaChallengeTTLSeconds = 300
)

// prepareLogin 为认证通过的用户创建授权会话，并把用户与"记住我"放进 context 供 LoginResponse 使用。
func prepareLogin(c *gin.Context, rbacService service.RBACService, u *user.User, remember bool, client service.LoginClient) error {
	access, err := rbacService.CreateSession(c.Request.Context(), u.ID, time.Duration(config.C.General.JwtMaxRefreshHours)*time.Hour,
		service.SessionOrigin{IP: client.IP, UserAgent: client.UserAgent, Remember: remember})
	if err != nil {
		return err
	}
	u.AuthorizationSessionID = access.SessionID
	log.Info(c).
		Str("accountSource", u.AccountSource).
		Uint("userId", u.ID).
		Str("username", u.Username).
		Msg("User logged in successfully")
	c.Set("login_user", u)
	c.Set("login_remember", remember)
	return nil
}

// IssueSession 为已完成认证的用户签发会话（cookie 与登录响应），与密码登录的成功响应完全相同。
// 两步验证第二步与 OIDC 回调都经由这里登录。
func (j *JwtMiddleware) IssueSession(c *gin.Context, u *user.User, remember bool, client service.LoginClient) error {
	if err := prepareLogin(c, j.RBACService, u, remember, client); err != nil {
		return err
	}
	token, expire, err := j.AuthMiddleware.TokenGenerator(&tokenSubject{User: u, Remember: remember})
	if err != nil {
		return err
	}
	j.AuthMiddleware.LoginResponse(c, http.StatusOK, token, expire)
	return nil
}

// StartSession 为已完成认证的用户创建会话并写入 jwt / csrf_token cookie，不写响应体：
// OIDC 回调用它登录后再重定向回站内。
func (j *JwtMiddleware) StartSession(c *gin.Context, u *user.User, remember bool, client service.LoginClient) error {
	if err := prepareLogin(c, j.RBACService, u, remember, client); err != nil {
		return err
	}
	token, _, err := j.AuthMiddleware.TokenGenerator(&tokenSubject{User: u, Remember: remember})
	if err != nil {
		return err
	}
	setAuthCookies(c, token, remember)
	return nil
}

// LoginWithTOTP 登录第二步：用第一步返回的挑战令牌与一次性码（或恢复码）换取会话。
// POST /api/v1/auth/login/totp
func (j *JwtMiddleware) LoginWithTOTP(c *gin.Context) {
	var req dto.TOTPLoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequestErr(c, err)
		return
	}
	challenge, err := j.MFAService.CompleteLoginChallenge(c.Request.Context(), req.Challenge, req.Code)
	if err != nil {
		if challenge != nil {
			j.LoginService.RecordLogin(challenge.Username, service.LoginClient{IP: c.ClientIP(), UserAgent: c.GetHeader("User-Agent")}, challenge.Method, false, challenge.UserID)
		}
		response.HandleError(c, err)
		return
	}
	client := service.LoginClient{IP: c.ClientIP(), UserAgent: c.GetHeader("User-Agent")}
	u, err := j.LoginService.ResumeLogin(c.Request.Context(), challenge.UserID)
	if err != nil {
		j.LoginService.RecordLogin(challenge.Username, client, challenge.Method, false, challenge.UserID)
		response.HandleError(c, err)
		return
	}
	if err := j.IssueSession(c, u, challenge.Remember, client); err != nil {
		response.HandleError(c, err)
		return
	}
	j.LoginService.RecordLogin(u.Username, client, challenge.Method, true, u.ID)
}

func jwtUnauthorizedMessageKey(message string) string {
	switch {
	case strings.Contains(message, apperror.ErrUserDisabled.Error()):
		return response.MsgUserDisabled
	case strings.Contains(message, apperror.ErrUserLocked.Error()):
		return response.MsgUserLocked
	case strings.Contains(message, apperror.ErrAccountExpired.Error()):
		return response.MsgAccountExpired
	case strings.Contains(message, apperror.ErrCredentialExpired.Error()):
		return response.MsgCredentialExpired
	case strings.Contains(message, apperror.ErrTooManyLoginAttempts.Error()):
		return response.MsgTooManyLoginAttempts
	case strings.Contains(message, apperror.ErrInvalidCaptchaCode.Error()):
		return response.MsgInvalidCaptchaCode
	case strings.Contains(message, apperror.ErrLoginMethodDisabled.Error()):
		return response.MsgLoginMethodDisabled
	case strings.Contains(message, apperror.ErrTOTPRequired.Error()):
		return response.ErrTOTPRequired
	default:
		return response.MsgUnauthorized
	}
}

// unauthorizedErrorCode 给登录失败附带业务错误码，前端据此弹出验证码或过期改密。
func unauthorizedErrorCode(msgKey string) int {
	if code := response.BusinessErrorCode(msgKey); code != 0 {
		return code
	}
	return response.ErrCodeUnauthorized
}

func jwtUnauthorizedStatus(defaultCode int, msgKey string) int {
	if msgKey == response.MsgUnauthorized {
		return defaultCode
	}
	return response.GetCode(msgKey)
}

// RefreshWithBlacklistCheck exchanges a still-refreshable token for a new one. The old
// token is blacklisted until it could no longer be refreshed, so a leaked or replayed
// token cannot be used again once its holder has refreshed it.
func (j *JwtMiddleware) RefreshWithBlacklistCheck(c *gin.Context) {
	token := j.extractToken(c)
	if token != "" && j.TokenBlacklistService.IsBlacklisted(c.Request.Context(), token) {
		log.Warn(c).Msg("Refresh rejected: token is blacklisted")
		j.refreshUnauthorized(c)
		return
	}
	claims, err := j.AuthMiddleware.CheckIfTokenExpire(c)
	if err != nil {
		j.refreshUnauthorized(c)
		return
	}
	// ParseToken records the token it actually validated; retire exactly that one.
	if parsed := jwt.GetToken(c); parsed != "" {
		token = parsed
	}
	identity := identityFromClaims(claims, j.audience)
	if identity == nil {
		j.refreshUnauthorized(c)
		return
	}
	if _, err := j.RBACService.GetSessionAccess(c.Request.Context(), identity.AuthorizationSessionID, identity.ID); err != nil {
		j.refreshUnauthorized(c)
		return
	}

	remember, _ := claims[jwtRememberClaim].(bool)
	newToken, expire, err := j.AuthMiddleware.TokenGenerator(&tokenSubject{User: identity, Remember: remember})
	if err != nil {
		response.InternalServerErrorErr(c, err)
		return
	}
	if remaining := time.Until(refreshableUntil(claims, j.AuthMiddleware.MaxRefresh)); remaining > 0 {
		if err := j.TokenBlacklistService.AddToBlacklist(c.Request.Context(), token, remaining); err != nil {
			log.Err(c, err).Msg("Refresh rejected: failed to retire previous token")
			response.InternalServerErrorErr(c, err)
			return
		}
	}
	// 刷新即说明会话仍在使用；记录失败不影响刷新本身。
	if err := j.RBACService.TouchSession(c.Request.Context(), identity.AuthorizationSessionID); err != nil {
		log.Err(c, err).Msg("Failed to record session activity")
	}
	c.Set("login_remember", remember)
	j.AuthMiddleware.RefreshResponse(c, http.StatusOK, newToken, expire)
}

func (j *JwtMiddleware) refreshUnauthorized(c *gin.Context) {
	msg := response.MsgUnauthorized
	if _, exists := c.Get("i18n"); exists {
		msg = gi18n.MustGetMessage(c, response.MsgUnauthorized)
	}
	c.JSON(http.StatusUnauthorized, gin.H{
		"code":    http.StatusUnauthorized,
		"data":    nil,
		"message": msg,
	})
	c.Abort()
}

// identityFromClaims rebuilds the token subject from validated claims. A token issued
// for another instance (different aud) identifies nobody here.
func identityFromClaims(claims map[string]any, audience string) *user.User {
	if aud, ok := claims["aud"].(string); !ok || aud != audience {
		return nil
	}
	id, idOK := claims[constant.JWT_IDENTITY_KEY].(float64)
	name, nameOK := claims[constant.JWT_NAME].(string)
	username, usernameOK := claims[constant.JWT_USERNAME].(string)
	accountSource, sourceOK := claims[constant.JWT_ACCOUNT_SOURCE].(string)
	sessionID, sessionOK := claims[constant.JWT_AUTHORIZATION_SESSION].(string)
	if !idOK || !nameOK || !usernameOK || !sourceOK || !sessionOK || sessionID == "" {
		return nil
	}
	return &user.User{
		ID:                     uint(id),
		Name:                   name,
		Username:               username,
		AccountSource:          accountSource,
		AuthorizationSessionID: sessionID,
	}
}

// refreshableUntil returns the moment after which a token can neither authenticate
// nor be refreshed: the later of its expiry and orig_iat + MaxRefresh.
func refreshableUntil(claims map[string]any, maxRefresh time.Duration) time.Time {
	var until time.Time
	if exp, ok := claims["exp"].(float64); ok {
		until = time.Unix(int64(exp), 0)
	}
	if origIat, ok := claims["orig_iat"].(float64); ok {
		if refreshEnd := time.Unix(int64(origIat), 0).Add(maxRefresh); refreshEnd.After(until) {
			until = refreshEnd
		}
	}
	return until
}

func newTokenID() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		panic(fmt.Sprintf("crypto/rand unavailable: %v", err))
	}
	return hex.EncodeToString(buf)
}

// extractToken extracts the JWT token from the request using the same sources
// as the middleware's TokenLookup (header: Authorization, cookie: jwt).
func (j *JwtMiddleware) extractToken(c *gin.Context) string {
	// Try Authorization header first
	authHeader := c.Request.Header.Get("Authorization")
	if authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && parts[0] == "Bearer" {
			return parts[1]
		}
	}
	// Try jwt cookie
	if cookie, err := c.Cookie("jwt"); err == nil && cookie != "" {
		return cookie
	}
	return ""
}

// LogoutHandler 自定义登出处理器，将 Token 加入黑名单
func (m *JwtMiddleware) LogoutHandler(c *gin.Context) {
	token := jwt.GetToken(c)
	if token == "" {
		response.BadRequestErr(c, errors.New("token not found"))
		return
	}

	// 获取 Token 剩余有效期
	claims := jwt.ExtractClaims(c)
	if sessionID, ok := claims[constant.JWT_AUTHORIZATION_SESSION].(string); ok {
		if err := m.RBACService.RevokeSession(c.Request.Context(), sessionID); err != nil {
			response.HandleError(c, err)
			return
		}
	}
	exp, ok := claims["exp"].(float64)
	if !ok {
		response.BadRequestErr(c, errors.New("invalid token claims"))
		return
	}

	// 计算剩余有效期
	expTime := time.Unix(int64(exp), 0)
	remaining := time.Until(expTime)
	if remaining <= 0 {
		// Token 已过期，无需加入黑名单
		m.logAudit(c, audit_log.AuditLogTypeLogout, ucontext.GetUsername(c), "Logout (token expired)", true)
		c.SetSameSite(http.SameSiteLaxMode)
		c.SetCookie("jwt", "", -1, "/", "", !config.C.General.Development, true)
		c.SetCookie("csrf_token", "", -1, "/", "", !config.C.General.Development, false)
		response.SuccessI18n(c, nil, response.InfLogoutSuccess)
		return
	}

	// 将 Token 加入黑名单，有效期为 Token 剩余有效期
	if err := m.TokenBlacklistService.AddToBlacklist(c.Request.Context(), token, remaining); err != nil {
		log.Err(c, err).Msg("Failed to add token to blacklist")
		m.logAudit(c, audit_log.AuditLogTypeLogout, ucontext.GetUsername(c), "Logout failed", false)
		response.InternalServerErrorErr(c, err)
		return
	}

	log.Info(c).Dur("expiresIn", remaining).Msg("Token added to blacklist")
	m.logAudit(c, audit_log.AuditLogTypeLogout, ucontext.GetUsername(c), "Logout successful", true)
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("jwt", "", -1, "/", "", !config.C.General.Development, true)
	c.SetCookie("csrf_token", "", -1, "/", "", !config.C.General.Development, false)
	response.SuccessI18n(c, nil, response.InfLogoutSuccess)
}

func setCSRFCookie(c *gin.Context, maxAge int) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return
	}
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("csrf_token", hex.EncodeToString(buf), maxAge, "/", "", !config.C.General.Development, false)
}

// logAudit 记录审计日志（异步）
func (m *JwtMiddleware) logAudit(c *gin.Context, logType audit_log.AuditLogType, target, details string, success bool) {
	m.AuditLogService.LogAsync(&audit_log.AuditLog{
		LogType:    logType,
		Operator:   ucontext.GetUsername(c),
		OperatorID: ucontext.GetUserID(c),
		Target:     target,
		Details:    details,
		IpAddr:     c.ClientIP(),
		Success:    success,
	})
}
