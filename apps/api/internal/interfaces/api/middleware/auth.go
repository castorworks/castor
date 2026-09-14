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
}

// NewJwtMiddleware 创建 JWT 中间件
func NewJwtMiddleware(
	loginService *service.LoginService,
	rbacService service.RBACService,
	tokenBlacklistService service.TokenBlacklistService,
	auditLogService service.AuditLogService,
) (*JwtMiddleware, error) {

	authMiddleware, err := jwt.New(&jwt.GinJWTMiddleware{
		Realm:       "castor",
		Key:         []byte(config.C.General.JwtKey),
		Timeout:     time.Duration(config.C.General.JwtTimeoutHours) * time.Hour,
		MaxRefresh:  time.Duration(config.C.General.JwtMaxRefreshHours) * time.Hour,
		IdentityKey: constant.JWT_IDENTITY_KEY,
		PayloadFunc: func(data interface{}) jwt.MapClaims {
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
					"jti": newTokenID(),
				}
			}
			return jwt.MapClaims{}
		},
		IdentityHandler: func(c *gin.Context) interface{} {
			if u := identityFromClaims(jwt.ExtractClaims(c)); u != nil {
				return u
			}
			return nil
		},
		LoginResponse: func(c *gin.Context, _ int, token string, expire time.Time) {
			maxAge := int(time.Duration(config.C.General.JwtTimeoutHours) * time.Hour / time.Second)
			c.SetSameSite(http.SameSiteLaxMode)
			c.SetCookie("jwt", token, maxAge, "/", "", !config.C.General.Development, true)
			setCSRFCookie(c, maxAge)

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
			maxAge := int(time.Duration(config.C.General.JwtTimeoutHours) * time.Hour / time.Second)
			c.SetSameSite(http.SameSiteLaxMode)
			c.SetCookie("jwt", token, maxAge, "/", "", !config.C.General.Development, true)
			setCSRFCookie(c, maxAge)
			response.Success(c, gin.H{"expire": expire})
		},
		Authenticator: func(c *gin.Context) (interface{}, error) {
			var req service.LoginReq
			if err := c.ShouldBindJSON(&req); err != nil {
				log.Err(c, err).Msg("Failed to bind login request")
				return nil, jwt.ErrMissingLoginValues
			}
			u, err := loginService.Login(c, c.Query("method"), &req, service.LoginClient{
				IP:        c.ClientIP(),
				UserAgent: c.GetHeader("User-Agent"),
			})
			if err == nil {
				access, sessionErr := rbacService.CreateSession(c.Request.Context(), u.ID, time.Duration(config.C.General.JwtMaxRefreshHours)*time.Hour)
				if sessionErr != nil {
					return nil, sessionErr
				}
				u.AuthorizationSessionID = access.SessionID
				log.Info(c).
					Str("accountSource", u.AccountSource).
					Uint("userId", u.ID).
					Str("username", u.Username).
					Msg("User logged in successfully")
				// 存储用户信息到 context，供 LoginResponse 使用
				c.Set("login_user", u)
			}
			return u, err
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

			c.JSON(statusCode, gin.H{
				"code":    statusCode,
				"data":    nil,
				"message": gi18n.MustGetMessage(c, msgKey),
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
	}, nil
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
	default:
		return response.MsgUnauthorized
	}
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
	identity := identityFromClaims(claims)
	if identity == nil {
		j.refreshUnauthorized(c)
		return
	}
	if _, err := j.RBACService.GetSessionAccess(c.Request.Context(), identity.AuthorizationSessionID, identity.ID); err != nil {
		j.refreshUnauthorized(c)
		return
	}

	newToken, expire, err := j.AuthMiddleware.TokenGenerator(identity)
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

// identityFromClaims rebuilds the token subject from validated claims.
func identityFromClaims(claims map[string]any) *user.User {
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
