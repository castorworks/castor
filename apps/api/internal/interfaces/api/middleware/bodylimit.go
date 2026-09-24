package middleware

import (
	"mime"
	"net/http"

	"github.com/castorworks/castor/internal/interfaces/api/response"
	"github.com/gin-gonic/gin"
)

// BodyLimit caps request body size. Multipart uploads get their own (larger) limit;
// every other body, including JSON, is capped at maxBody. Oversized bodies with a
// declared Content-Length are rejected up front with 413; bodies without one are cut
// off by http.MaxBytesReader, which makes binding fail.
func BodyLimit(maxBody, maxMultipart int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Body == nil || c.Request.Body == http.NoBody {
			c.Next()
			return
		}
		limit := maxBody
		if mediaType, _, err := mime.ParseMediaType(c.GetHeader("Content-Type")); err == nil && mediaType == "multipart/form-data" {
			limit = maxMultipart
		}
		if limit <= 0 {
			c.Next()
			return
		}
		if c.Request.ContentLength > limit {
			response.RequestTooLargeAbort(c)
			return
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, limit)
		c.Next()
	}
}
