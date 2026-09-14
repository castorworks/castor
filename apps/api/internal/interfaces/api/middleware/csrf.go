package middleware

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/castorworks/castor/internal/interfaces/api/response"
	"github.com/gin-gonic/gin"
)

// CSRFMiddleware protects unsafe requests authenticated with the JWT cookie.
// Authorization-header requests are not browser ambient authority and do not
// need this check. The token is deliberately readable by browser JavaScript
// and must be echoed in X-CSRF-Token.
func CSRFMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if isUnsafeMethod(c.Request.Method) {
			hasAuthorization := hasValidBearerAuthorization(c)
			if !hasAuthorization {
				jwtCookie, jwtErr := c.Cookie("jwt")
				csrfCookie, csrfErr := c.Cookie("csrf_token")
				csrfHeader := c.GetHeader("X-CSRF-Token")
				if jwtErr != nil || jwtCookie == "" || csrfErr != nil || csrfCookie == "" || csrfHeader == "" ||
					subtle.ConstantTimeCompare([]byte(csrfCookie), []byte(csrfHeader)) != 1 {
					response.Forbidden(c)
					c.Abort()
					return
				}
			}
		}
		c.Next()
	}
}

func hasValidBearerAuthorization(c *gin.Context) bool {
	value := strings.TrimSpace(c.GetHeader("Authorization"))
	if value == "" {
		return false
	}

	parts := strings.Fields(value)
	return len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") && parts[1] != ""
}

func isUnsafeMethod(method string) bool {
	return method == http.MethodPost || method == http.MethodPut || method == http.MethodPatch || method == http.MethodDelete
}
