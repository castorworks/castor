package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/castorworks/castor/internal/application/apperror"
	"github.com/gin-gonic/gin"
)

// 前端靠业务错误码区分"需要验证码""凭证过期"等情形（HTTP 状态码 400/403 不足以区分），
// 所以映射过的业务错误必须把 ErrCode.Code 写进响应体的 errorCode。
func TestHandleErrorExposesBusinessErrorCode(t *testing.T) {
	router := setupResponseTestRouter()
	cases := map[string]struct {
		err        error
		wantStatus int
		wantCode   int
	}{
		"/captcha":    {apperror.ErrInvalidCaptchaCode, http.StatusBadRequest, ErrCodeInvalidCaptcha},
		"/credential": {apperror.ErrCredentialExpired, http.StatusForbidden, ErrCodeCredentialExpired},
		"/locked":     {apperror.ErrUserLocked, http.StatusForbidden, ErrCodeAccountLocked},
	}
	for path, tc := range cases {
		err := tc.err
		router.GET(path, func(c *gin.Context) { HandleError(c, err) })
	}
	for path, tc := range cases {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		var body struct {
			Code      int    `json:"code"`
			ErrorCode int    `json:"errorCode"`
			Message   string `json:"message"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatalf("%s: invalid body %s", path, w.Body.String())
		}
		if w.Code != tc.wantStatus || body.Code != tc.wantStatus {
			t.Fatalf("%s: status=%d code=%d, want %d", path, w.Code, body.Code, tc.wantStatus)
		}
		if body.ErrorCode != tc.wantCode {
			t.Fatalf("%s: errorCode=%d, want %d (body %s)", path, body.ErrorCode, tc.wantCode, w.Body.String())
		}
		if body.Message == "" {
			t.Fatalf("%s: message must be localized", path)
		}
	}
}

func TestBusinessErrorCodeForMessageKey(t *testing.T) {
	if got := BusinessErrorCode(MsgInvalidCaptchaCode); got != ErrCodeInvalidCaptcha {
		t.Fatalf("BusinessErrorCode(captcha) = %d, want %d", got, ErrCodeInvalidCaptcha)
	}
	if got := BusinessErrorCode("no-such-key"); got != 0 {
		t.Fatalf("unknown key must map to 0, got %d", got)
	}
}
