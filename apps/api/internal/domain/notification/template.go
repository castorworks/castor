package notification

// TemplateRenderer 渲染通知模板与邮件里的固定文案（服务器默认语言）。
type TemplateRenderer interface {
	Render(key string, data map[string]any) (string, string, error)
	// Text 渲染单个文案 key
	Text(key string, data map[string]any) (string, error)
}
