package i18n

import (
	"bytes"
	"fmt"
	"path/filepath"
	"text/template"

	"github.com/castorworks/castor/internal/domain/notification"
	"github.com/castorworks/castor/internal/infrastructure/config"
	"github.com/castorworks/castor/internal/pkg/toml"
)

type notificationTemplateRenderer struct {
	templates map[string]string
}

func NewNotificationTemplateRenderer() (notification.TemplateRenderer, error) {
	path := filepath.Join(config.C.I18n.RootPath, config.C.I18n.DefaultLanguage+".toml")

	templates := map[string]string{}
	if _, err := toml.DecodeFile(path, &templates); err != nil {
		return nil, err
	}

	return &notificationTemplateRenderer{templates: templates}, nil
}

func (r *notificationTemplateRenderer) Render(key string, data map[string]any) (string, string, error) {
	title, err := r.render(key+"Title", data)
	if err != nil {
		return "", "", err
	}
	content, err := r.render(key+"Content", data)
	if err != nil {
		return "", "", err
	}
	return title, content, nil
}

func (r *notificationTemplateRenderer) Text(key string, data map[string]any) (string, error) {
	return r.render(key, data)
}

func (r *notificationTemplateRenderer) render(key string, data map[string]any) (string, error) {
	text, ok := r.templates[key]
	if !ok {
		return "", fmt.Errorf("notification template not found: %s", key)
	}

	tpl, err := template.New(key).Parse(text)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}
