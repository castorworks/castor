package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/BurntSushi/toml"
	ginI18n "github.com/gin-contrib/i18n"
	"github.com/gin-gonic/gin"
	"golang.org/x/text/language"
)

func TestCSRFMiddleware(t *testing.T) {

	tests := []struct {
		name       string
		authority  string
		csrfCookie string
		csrfHeader string
		jwtCookie  bool
		method     string
		wantStatus int
	}{
		{name: "safe request does not require csrf", method: http.MethodGet, wantStatus: http.StatusNoContent},
		{name: "cookie request requires csrf", method: http.MethodPost, wantStatus: http.StatusForbidden},
		{name: "matching csrf token is accepted", method: http.MethodPost, csrfCookie: "token", csrfHeader: "token", wantStatus: http.StatusNoContent},
		{name: "valid bearer request is accepted", method: http.MethodPost, authority: "Bearer access-token", wantStatus: http.StatusNoContent},
		{name: "non bearer authorization does not bypass csrf", method: http.MethodPost, authority: "Basic credentials", wantStatus: http.StatusForbidden},
		{name: "bearer header does not bypass csrf when jwt cookie is present", method: http.MethodPost, authority: "Bearer junk", jwtCookie: true, wantStatus: http.StatusForbidden},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			router := gin.New()
			router.Use(ginI18n.Localize(
				ginI18n.WithBundle(&ginI18n.BundleCfg{
					RootPath:         "../../../../configs/i18n",
					AcceptLanguage:   []language.Tag{language.Chinese, language.English},
					DefaultLanguage:  language.Chinese,
					UnmarshalFunc:    toml.Unmarshal,
					FormatBundleFile: "toml",
				}),
			))
			router.Use(CSRFMiddleware())
			router.Any("/", func(c *gin.Context) { c.Status(http.StatusNoContent) })

			request := httptest.NewRequest(test.method, "/", nil)
			if test.authority != "" {
				request.Header.Set("Authorization", test.authority)
			}
			if test.csrfHeader != "" {
				request.Header.Set("X-CSRF-Token", test.csrfHeader)
			}
			if test.csrfCookie != "" || test.jwtCookie {
				request.AddCookie(&http.Cookie{Name: "jwt", Value: "jwt-token"})
			}
			if test.csrfCookie != "" {
				request.AddCookie(&http.Cookie{Name: "csrf_token", Value: test.csrfCookie})
			}

			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d", response.Code, test.wantStatus)
			}
		})
	}
}
