package middleware

import (
	"github.com/castorworks/castor/internal/application/service"
	"github.com/castorworks/castor/internal/domain/user"
	"github.com/castorworks/castor/internal/interfaces/api/response"
	"github.com/castorworks/castor/internal/pkg/constant"
	"github.com/castorworks/castor/internal/pkg/log"
	"github.com/gin-gonic/gin"
)

type AdminRoleMiddleware struct {
	rbac service.RBACService
}

func NewAdminRoleMiddleware(rbac service.RBACService) *AdminRoleMiddleware {
	return &AdminRoleMiddleware{rbac: rbac}
}

func (m *AdminRoleMiddleware) MiddlewareFunc(c *gin.Context) {
	identity, exists := c.Get(constant.JWT_IDENTITY_KEY)
	u, ok := identity.(*user.User)
	if !exists || !ok || u.AuthorizationSessionID == "" {
		response.UnauthorizedAbort(c)
		return
	}
	resourcePath := c.FullPath()
	if resourcePath == "" {
		log.Warn(c).Str("path", c.Request.URL.Path).Msg("Gin route template is unavailable for RBAC authorization")
		response.ForbiddenErr(c, nil)
		c.Abort()
		return
	}
	allowed, err := m.rbac.AuthorizeSession(c.Request.Context(), u.AuthorizationSessionID, u.ID, resourcePath, c.Request.Method)
	if err != nil {
		log.Err(c, err).Uint("userId", u.ID).Str("sessionId", u.AuthorizationSessionID).Msg("RBAC session validation failed")
		response.UnauthorizedAbort(c)
		return
	}
	if !allowed {
		response.ForbiddenErr(c, nil)
		c.Abort()
		return
	}
	c.Next()
}
