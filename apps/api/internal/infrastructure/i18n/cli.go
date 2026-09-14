package i18n

import (
	"path/filepath"

	"github.com/castorworks/castor/internal/infrastructure/config"
	"github.com/castorworks/castor/internal/pkg/toml"
)

// CLIMessage uses the same configured language and catalog as the API.
func CLIMessage(key string) string {
	messages := map[string]string{}
	path := filepath.Join(config.C.I18n.RootPath, config.C.I18n.DefaultLanguage+".toml")
	if _, err := toml.DecodeFile(path, &messages); err == nil && messages[key] != "" {
		return messages[key]
	}
	return key
}
