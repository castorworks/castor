package notification

type TemplateRenderer interface {
	Render(key string, data map[string]any) (string, string, error)
}
