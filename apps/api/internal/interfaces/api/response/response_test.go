package response

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/BurntSushi/toml"
	ginI18n "github.com/gin-contrib/i18n"
	"github.com/gin-gonic/gin"
	"golang.org/x/text/language"
)

func setupResponseTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(ginI18n.Localize(
		ginI18n.WithBundle(&ginI18n.BundleCfg{
			RootPath:         "../../../../configs/i18n",
			AcceptLanguage:   []language.Tag{language.Chinese, language.English},
			DefaultLanguage:  language.Chinese,
			UnmarshalFunc:    toml.Unmarshal,
			FormatBundleFile: "toml",
		}),
	))
	return r
}

func TestForbiddenErrAllowsNilError(t *testing.T) {
	router := setupResponseTestRouter()
	router.GET("/forbidden", func(c *gin.Context) {
		ForbiddenErr(c, nil)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/forbidden", nil)

	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d, body: %s", http.StatusForbidden, w.Code, w.Body.String())
	}
}
