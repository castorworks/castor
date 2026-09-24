package shared

// I18nText 是一段面向用户展示的四语言文案。
//
// 用于运营可编辑、因而无法放进 messages 文件的数据（菜单标题、字典标签）。
// 固定文案仍然走 i18n key：后端在 configs/i18n/*.toml，前端在 messages/*.json。
type I18nText struct {
	En string `json:"en"`
	Zh string `json:"zh"`
	Ja string `json:"ja"`
	Ko string `json:"ko"`
}

// Text 构造一段四语言文案。
func Text(en, zh, ja, ko string) I18nText {
	return I18nText{En: en, Zh: zh, Ja: ja, Ko: ko}
}

// Localized 返回指定语言的文案，缺失时回退到英文。
func (t I18nText) Localized(lang string) string {
	switch lang {
	case "zh":
		if t.Zh != "" {
			return t.Zh
		}
	case "ja":
		if t.Ja != "" {
			return t.Ja
		}
	case "ko":
		if t.Ko != "" {
			return t.Ko
		}
	}
	return t.En
}

// IsComplete 报告四种语言是否都已填写。
func (t I18nText) IsComplete() bool {
	return t.En != "" && t.Zh != "" && t.Ja != "" && t.Ko != ""
}
