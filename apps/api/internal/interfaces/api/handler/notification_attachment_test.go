package handler

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	jwt "github.com/appleboy/gin-jwt/v2"
	"github.com/castorworks/castor/internal/application/apperror"
	"github.com/castorworks/castor/internal/application/service"
	"github.com/castorworks/castor/internal/pkg/constant"
	"github.com/gin-gonic/gin"
)

// attachmentNotificationService 只回答"谁能下载哪条通知的哪个文件"
type attachmentNotificationService struct {
	service.NotificationService
	calls []uint
}

func (s *attachmentNotificationService) PrepareAttachmentDownload(_ context.Context, userID, notificationID uint, objectKey string) (*service.AssetDownloadResult, error) {
	s.calls = append(s.calls, userID)
	if userID != 7 || notificationID != 3 || objectKey != "report.pdf" {
		return nil, apperror.ErrNotificationNotDelivered
	}
	return &service.AssetDownloadResult{ObjectKey: objectKey, Filename: "季度报告.pdf", ContentType: "application/pdf", Size: 4}, nil
}

type contentAssetService struct {
	service.AssetService
}

func (contentAssetService) WriteContent(_ context.Context, _ string, w io.Writer) error {
	_, err := w.Write([]byte("%PDF"))
	return err
}

// 附件下载路由：通知模块自己鉴权，任何失败都表现为不存在，不泄露文件或通知是否存在。
func TestNotificationHandler_DownloadAttachment(t *testing.T) {
	notifications := &attachmentNotificationService{}
	h := NewNotificationHandler(notifications, contentAssetService{})

	router := setupTestRouter()
	router.GET("/notifications/:id/attachments/:objectKey", func(c *gin.Context) {
		if c.GetHeader("X-Test-User") == "recipient" {
			c.Set("JWT_PAYLOAD", jwt.MapClaims{constant.JWT_IDENTITY_KEY: float64(7)})
		} else {
			c.Set("JWT_PAYLOAD", jwt.MapClaims{constant.JWT_IDENTITY_KEY: float64(8)})
		}
		h.DownloadAttachment(c)
	})

	get := func(user, path string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("X-Test-User", user)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		return w
	}

	ok := get("recipient", "/notifications/3/attachments/report.pdf")
	if ok.Code != http.StatusOK || ok.Body.String() != "%PDF" {
		t.Fatalf("recipient download = %d %q", ok.Code, ok.Body.String())
	}
	if got := ok.Header().Get("Content-Disposition"); got != `attachment; filename*=UTF-8''%E5%AD%A3%E5%BA%A6%E6%8A%A5%E5%91%8A.pdf` {
		t.Errorf("Content-Disposition = %q", got)
	}
	if ok.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Error("downloads must forbid content sniffing")
	}

	for name, w := range map[string]*httptest.ResponseRecorder{
		"stranger":             get("stranger", "/notifications/3/attachments/report.pdf"),
		"another notification": get("recipient", "/notifications/4/attachments/report.pdf"),
		"another file":         get("recipient", "/notifications/3/attachments/secret.pdf"),
		"malformed id":         get("recipient", "/notifications/abc/attachments/report.pdf"),
	} {
		if w.Code != http.StatusNotFound || w.Body.Len() != 0 {
			t.Errorf("%s: got %d with %d bytes, want an empty 404", name, w.Code, w.Body.Len())
		}
	}
	if len(notifications.calls) != 4 || notifications.calls[1] != 8 {
		t.Errorf("service must be asked with the caller's own user id, calls = %v", notifications.calls)
	}
}
