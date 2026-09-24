package setting

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

// SettingType 配置类型
type SettingType string

const (
	SettingTypeString SettingType = "STRING"
	SettingTypeNumber SettingType = "NUMBER"
	SettingTypeBool   SettingType = "BOOL"
	SettingTypeJSON   SettingType = "JSON"
	SettingTypeArray  SettingType = "ARRAY"
	SettingTypeSecret SettingType = "SECRET"
)

// SettingCategory 配置分类
type SettingCategory string

const (
	CategoryGeneral  SettingCategory = "GENERAL"
	CategorySecurity SettingCategory = "SECURITY"
	CategoryEmail    SettingCategory = "EMAIL"
	CategoryStorage  SettingCategory = "STORAGE"
	CategoryFeature  SettingCategory = "FEATURE"
	CategoryCustom   SettingCategory = "CUSTOM"
)

// 配置 Key 常量
const (
	KeySiteName                    = "site.name"
	KeySiteLogo                    = "site.logo"
	KeySiteDescription             = "site.description"
	KeyFeatureRegisterEnabled      = "feature.register.enabled"
	KeyFeatureCaptchaEnabled       = "feature.captcha.enabled"
	KeySecurityPasswordMinLength   = "security.password.minLength"
	KeySecurityPasswordComplexity  = "security.password.requireComplexity"
	KeySecurityPasswordMaxAgeDays  = "security.password.maxAgeDays"
	KeySecurityLoginMaxAttempts    = "security.login.maxAttempts"
	KeySecurityLoginLockMinutes    = "security.login.lockMinutes"
	KeySecurityLoginAllowedMethods = "security.login.allowedMethods"
)

// Setting 系统配置实体（纯 Domain 模型，无 ORM 依赖）
type Setting struct {
	ID          uint            `json:"id"`
	CreatedAt   time.Time       `json:"createdAt"`
	CreatedBy   uint            `json:"createdBy"`
	UpdatedAt   time.Time       `json:"updatedAt"`
	UpdatedBy   uint            `json:"updatedBy"`
	Key         string          `json:"key"`
	Value       string          `json:"value"`
	Type        SettingType     `json:"type"`
	Category    SettingCategory `json:"category"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	DefaultVal  string          `json:"defaultVal"`
	Options     SettingOptions  `json:"options"`
	IsSystem    bool            `json:"isSystem"`
	IsPublic    bool            `json:"isPublic"`
	SortOrder   int             `json:"sortOrder"`
}

// SettingOptions 配置选项
type SettingOptions []SettingOption

type SettingOption struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

// Scan 实现 sql.Scanner 接口
func (o *SettingOptions) Scan(value interface{}) error {
	if value == nil {
		*o = nil
		return nil
	}
	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return nil
	}
	if len(bytes) == 0 {
		*o = nil
		return nil
	}
	return json.Unmarshal(bytes, o)
}

// Value 实现 driver.Valuer 接口
func (o SettingOptions) Value() (driver.Value, error) {
	if o == nil {
		return nil, nil
	}
	return json.Marshal(o)
}

// DefaultSettings 默认系统配置
var DefaultSettings = []Setting{
	{Key: "site.name", Value: "Castor Admin", Type: SettingTypeString, Category: CategoryGeneral, Name: "seedSettings.siteName", Description: "seedSettings.siteNameDesc", IsSystem: true, IsPublic: true, SortOrder: 1},
	{Key: "site.logo", Value: "", Type: SettingTypeString, Category: CategoryGeneral, Name: "seedSettings.siteLogo", Description: "seedSettings.siteLogoDesc", IsSystem: true, IsPublic: true, SortOrder: 2},
	{Key: "site.description", Value: "Castor Admin", Type: SettingTypeString, Category: CategoryGeneral, Name: "seedSettings.siteDescription", Description: "seedSettings.siteDescriptionDesc", IsSystem: true, IsPublic: true, SortOrder: 3},
	{Key: "feature.register.enabled", Value: "false", Type: SettingTypeBool, Category: CategoryFeature, Name: "seedSettings.openRegistration", Description: "seedSettings.openRegistrationDesc", DefaultVal: "false", IsSystem: true, IsPublic: true, SortOrder: 10},
	{Key: "feature.captcha.enabled", Value: "true", Type: SettingTypeBool, Category: CategoryFeature, Name: "seedSettings.loginCaptcha", Description: "seedSettings.loginCaptchaDesc", DefaultVal: "true", IsSystem: true, IsPublic: true, SortOrder: 11},
	// 密码策略公开：注册、找回密码、过期改密等未登录页面要据此提示并预校验，策略本身不是秘密。
	{Key: "security.password.minLength", Value: "8", Type: SettingTypeNumber, Category: CategorySecurity, Name: "seedSettings.minPasswordLength", Description: "seedSettings.minPasswordLengthDesc", DefaultVal: "8", IsSystem: true, IsPublic: true, SortOrder: 20},
	{Key: "security.login.maxAttempts", Value: "5", Type: SettingTypeNumber, Category: CategorySecurity, Name: "seedSettings.maxLoginAttempts", Description: "seedSettings.maxLoginAttemptsDesc", DefaultVal: "5", IsSystem: true, IsPublic: false, SortOrder: 21},
	{Key: "security.login.lockMinutes", Value: "30", Type: SettingTypeNumber, Category: CategorySecurity, Name: "seedSettings.loginLockDuration", Description: "seedSettings.loginLockDurationDesc", DefaultVal: "30", IsSystem: true, IsPublic: false, SortOrder: 22},
	{
		Key:         KeySecurityLoginAllowedMethods,
		Value:       `["password","email","mobile"]`,
		Type:        SettingTypeArray,
		Category:    CategorySecurity,
		Name:        "seedSettings.allowedLoginMethods",
		Description: "seedSettings.allowedLoginMethodsDesc",
		DefaultVal:  `["password","email","mobile"]`,
		Options: SettingOptions{
			{Label: "loginMethods.password", Value: "password"},
			{Label: "loginMethods.email", Value: "email"},
			{Label: "loginMethods.mobile", Value: "mobile"},
		},
		IsSystem:  true,
		IsPublic:  true,
		SortOrder: 23,
	},
	{Key: KeySecurityPasswordComplexity, Value: "true", Type: SettingTypeBool, Category: CategorySecurity, Name: "seedSettings.passwordComplexity", Description: "seedSettings.passwordComplexityDesc", DefaultVal: "true", IsSystem: true, IsPublic: true, SortOrder: 24},
	{Key: KeySecurityPasswordMaxAgeDays, Value: "0", Type: SettingTypeNumber, Category: CategorySecurity, Name: "seedSettings.passwordMaxAge", Description: "seedSettings.passwordMaxAgeDesc", DefaultVal: "0", IsSystem: true, IsPublic: false, SortOrder: 25},
}

// PublicPasswordPolicyKeys 是公开的密码策略配置：未登录页面（注册、重置密码）据此做前端预校验，
// 与后端的实际要求保持一致。
var PublicPasswordPolicyKeys = []string{KeySecurityPasswordMinLength, KeySecurityPasswordComplexity}
