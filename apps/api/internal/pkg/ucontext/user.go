package ucontext

import (
	"context"

	jwt "github.com/appleboy/gin-jwt/v2"
	"github.com/castorworks/castor/internal/pkg/constant"
	"github.com/gin-gonic/gin"
)

// GetUserID 获取当前用户ID，如果未登录，则返回0
func GetUserID(c *gin.Context) uint {
	if c == nil {
		return 0
	}

	claims := jwt.ExtractClaims(c)
	if claims != nil {
		if id, ok := claims[constant.JWT_IDENTITY_KEY].(float64); ok {
			return uint(id)
		}
	}

	return 0
}

// GetUsername 获取当前用户名，如果未登录，则返回空字符串
func GetUsername(c *gin.Context) string {
	if c == nil {
		return ""
	}

	claims := jwt.ExtractClaims(c)
	if claims != nil {
		if username, ok := claims[constant.JWT_USERNAME].(string); ok {
			return username
		}
	}

	return ""
}

func GetAuthorizationSessionID(c *gin.Context) string {
	if c == nil {
		return ""
	}
	claims := jwt.ExtractClaims(c)
	if claims != nil {
		if id, ok := claims[constant.JWT_AUTHORIZATION_SESSION].(string); ok {
			return id
		}
	}
	return ""
}

// WithAuditContext 创建带有审计用户 ID 的 context
// 用于在 handler 层将用户 ID 注入到 context 中，供 model 层使用
func WithAuditContext(c *gin.Context) context.Context {
	userID := GetUserID(c)
	return WithAuditUserID(c.Request.Context(), userID)
}
