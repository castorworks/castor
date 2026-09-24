package middleware

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestBodyLimit(t *testing.T) {
	t.Parallel()
	router := newI18nTestRouter()
	router.Use(BodyLimit(16, 64))
	router.POST("/", func(c *gin.Context) {
		if _, err := io.ReadAll(c.Request.Body); err != nil {
			c.Status(http.StatusBadRequest)
			return
		}
		c.Status(http.StatusOK)
	})

	tests := []struct {
		name        string
		contentType string
		body        string
		hideLength  bool
		wantStatus  int
	}{
		{"json within limit", "application/json", `{"a":1}`, false, http.StatusOK},
		{"json over limit", "application/json", strings.Repeat("a", 17), false, http.StatusRequestEntityTooLarge},
		{"json over limit without length", "application/json", strings.Repeat("a", 17), true, http.StatusBadRequest},
		{"multipart uses upload limit", "multipart/form-data; boundary=x", strings.Repeat("a", 60), false, http.StatusOK},
		{"multipart over upload limit", "multipart/form-data; boundary=x", strings.Repeat("a", 65), false, http.StatusRequestEntityTooLarge},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var body io.Reader = bytes.NewBufferString(tt.body)
			if tt.hideLength {
				body = io.NopCloser(body)
			}
			req := httptest.NewRequest(http.MethodPost, "/", body)
			if tt.hideLength {
				req.ContentLength = -1
			}
			req.Header.Set("Content-Type", tt.contentType)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			if w.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", w.Code, tt.wantStatus)
			}
		})
	}
}
