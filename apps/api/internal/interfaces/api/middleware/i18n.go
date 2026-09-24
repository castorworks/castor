package middleware

import (
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/castorworks/castor/internal/infrastructure/config"
	ginI18n "github.com/gin-contrib/i18n"
	"github.com/gin-gonic/gin"
	"golang.org/x/text/language"
)

// 语言标签映射
var languageTagMap = map[string]language.Tag{
	"zh":    language.Chinese,
	"zh-CN": language.SimplifiedChinese,
	"zh-TW": language.TraditionalChinese,
	"en":    language.English,
	"en-US": language.AmericanEnglish,
	"en-GB": language.BritishEnglish,
	"ja":    language.Japanese,
	"ko":    language.Korean,
}

// getDefaultLanguageTag 获取默认语言标签
func getDefaultLanguageTag() language.Tag {
	if tag, ok := languageTagMap[config.C.I18n.DefaultLanguage]; ok {
		return tag
	}
	return language.Chinese
}

// getSupportedLanguageTags 获取支持的语言标签列表
func getSupportedLanguageTags() []language.Tag {
	tags := make([]language.Tag, 0, len(config.C.I18n.SupportedLanguages))
	for _, lang := range config.C.I18n.SupportedLanguages {
		if tag, ok := languageTagMap[lang]; ok {
			tags = append(tags, tag)
		}
	}
	if len(tags) == 0 {
		return []language.Tag{language.Chinese, language.English, language.Japanese, language.Korean}
	}
	return tags
}

// GinI18nLocalize 创建国际化中间件
func GinI18nLocalize() gin.HandlerFunc {
	return ginI18n.Localize(
		ginI18n.WithBundle(&ginI18n.BundleCfg{
			RootPath:         config.C.I18n.RootPath,
			AcceptLanguage:   getSupportedLanguageTags(),
			DefaultLanguage:  getDefaultLanguageTag(),
			UnmarshalFunc:    toml.Unmarshal,
			FormatBundleFile: "toml",
		}),
		ginI18n.WithGetLngHandle(getLanguageFromRequest),
	)
}

// getLanguageFromRequest 从请求中获取语言设置
// 优先级: query参数 > X-Language header > Accept-Language header > 默认语言
func getLanguageFromRequest(c *gin.Context, defaultLng string) string {
	// 1. 优先检查 query 参数 (e.g., ?lng=en)
	if lng := c.Query("lng"); lng != "" {
		return normalizeLanguage(lng)
	}

	// 2. 检查自定义 X-Language header
	if lng := c.GetHeader("X-Language"); lng != "" {
		return normalizeLanguage(lng)
	}

	// 3. 检查标准 Accept-Language header
	if acceptLang := c.GetHeader("Accept-Language"); acceptLang != "" {
		return parseAcceptLanguage(acceptLang)
	}

	return defaultLng
}

// parseAcceptLanguage 解析 Accept-Language header
// 支持格式: "zh-CN,zh;q=0.9,en;q=0.8"
func parseAcceptLanguage(header string) string {
	// 取第一个语言（优先级最高）
	if header == "" {
		return ""
	}

	// 处理逗号分隔的多语言
	parts := strings.Split(header, ",")
	if len(parts) == 0 {
		return ""
	}

	// 取第一个语言部分
	firstLang := strings.TrimSpace(parts[0])

	// 移除 quality 参数 (e.g., "en;q=0.9" -> "en")
	if idx := strings.Index(firstLang, ";"); idx > 0 {
		firstLang = firstLang[:idx]
	}

	return normalizeLanguage(firstLang)
}

// normalizeLanguage 标准化语言代码
func normalizeLanguage(lang string) string {
	lang = strings.TrimSpace(lang)
	lang = strings.ToLower(lang)

	// 映射常见变体到基础语言代码
	switch {
	case strings.HasPrefix(lang, "zh"):
		return "zh"
	case strings.HasPrefix(lang, "en"):
		return "en"
	case strings.HasPrefix(lang, "ja"):
		return "ja"
	case strings.HasPrefix(lang, "ko"):
		return "ko"
	default:
		return lang
	}
}
