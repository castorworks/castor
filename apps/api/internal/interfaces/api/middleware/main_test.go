package middleware

import (
	"os"
	"testing"

	"github.com/BurntSushi/toml"
	ginI18n "github.com/gin-contrib/i18n"
	"github.com/gin-gonic/gin"
	"golang.org/x/text/language"
)

// TestMain sets the Gin mode once; gin.SetMode writes package globals and must not be
// called from parallel tests.
func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	os.Exit(m.Run())
}

// newI18nTestRouter returns an engine with the i18n middleware used by response helpers.
func newI18nTestRouter() *gin.Engine {
	router := gin.New()
	router.Use(ginI18n.Localize(ginI18n.WithBundle(&ginI18n.BundleCfg{
		RootPath:         "../../../../configs/i18n",
		AcceptLanguage:   []language.Tag{language.English},
		DefaultLanguage:  language.English,
		UnmarshalFunc:    toml.Unmarshal,
		FormatBundleFile: "toml",
	})))
	return router
}
