package middleware

import (
	"fmt"
	"time"

	"github.com/castorworks/castor/internal/interfaces/api/response"
	"github.com/castorworks/castor/internal/pkg/ratelimit"
	"github.com/gin-gonic/gin"
)

// RemoteIpPathRateLimit 根据远程 IP 和请求路径进行限流
func RemoteIpPathRateLimit(rateLimiter *ratelimit.RateLimiter, duration time.Duration, maxAllowed int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		// ClientIP only trusts forwarding headers when Gin has been configured
		// with an explicit trusted proxy list. Never key limits on a raw header
		// supplied by the caller, otherwise the limit can be bypassed trivially.
		ip := c.ClientIP()

		path := c.Request.URL.Path
		key := fmt.Sprintf("%s:%s", ip, path)

		reached, err := rateLimiter.IsRequestReachLimit(c.Request.Context(), key, duration, maxAllowed)
		if err != nil {
			response.InternalServerErrorAbort(c)
			return
		}

		if reached {
			response.TooManyRequestsAbort(c)
			return
		}

		c.Next()
	}
}
