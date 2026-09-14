package i18n

import (
	"testing"

	"github.com/castorworks/castor/internal/infrastructure/config"
)

func TestDeploymentMessagesAvailableInAllLanguages(t *testing.T) {
	original := config.C.I18n
	t.Cleanup(func() { config.C.I18n = original })
	config.C.I18n.RootPath = "../../../configs/i18n"
	for _, lang := range []string{"zh", "en", "ja", "ko"} {
		config.C.I18n.DefaultLanguage = lang
		for _, key := range []string{"DeployCommandUsage", "DeployInitFailed", "DeployInitComplete", "DeployGeneratedAdminPassword"} {
			if got := CLIMessage(key); got == key || got == "" {
				t.Fatalf("missing %s in %s", key, lang)
			}
		}
	}
}
