package i18n

import (
	"testing"

	"github.com/castorworks/castor/internal/infrastructure/config"
)

func TestNotificationTemplateRenderer_RenderFromConfiguredToml(t *testing.T) {
	oldRootPath := config.C.I18n.RootPath
	oldDefaultLanguage := config.C.I18n.DefaultLanguage
	config.C.I18n.RootPath = "../../../configs/i18n"
	config.C.I18n.DefaultLanguage = "en"
	defer func() {
		config.C.I18n.RootPath = oldRootPath
		config.C.I18n.DefaultLanguage = oldDefaultLanguage
	}()

	renderer, err := NewNotificationTemplateRenderer()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	title, content, err := renderer.Render("NotificationAssetStatusUpdated", map[string]any{"status": "ACTIVE"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if title != "Asset status updated" {
		t.Fatalf("expected localized title, got %q", title)
	}
	if content != "An asset status changed to ACTIVE." {
		t.Fatalf("expected localized content, got %q", content)
	}
}
