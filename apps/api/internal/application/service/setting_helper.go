package service

import (
	"context"
	"fmt"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/castorworks/castor/internal/application/apperror"
	"github.com/castorworks/castor/internal/domain/setting"
	"github.com/hyperits/gosuite/logger"
)

// SettingHelper 系统配置读取辅助工具
// 提供从数据库配置中读取常用配置项的方法，避免各 Service 重复实现
type SettingHelper struct {
	settingRepo setting.Repository
}

// NewSettingHelper 创建配置辅助工具
func NewSettingHelper(settingRepo setting.Repository) *SettingHelper {
	return &SettingHelper{settingRepo: settingRepo}
}

// MaxPasswordBytes 是 bcrypt 能参与校验的最大口令字节数，超出部分会被拒绝而不是截断。
const MaxPasswordBytes = 72

// passwordComplexityClasses 启用复杂度要求时，密码至少覆盖的字符类别数（小写/大写/数字/符号）。
const passwordComplexityClasses = 3

// PasswordPolicy 是当前生效的密码策略，由系统配置 security.password.* 决定。
type PasswordPolicy struct {
	MinLength         int  // 最小字符数（按 Unicode 字符计，而不是字节）
	RequireComplexity bool // 是否要求覆盖至少三类字符
	MaxAgeDays        int  // 密码有效天数，0 表示永不过期
}

// Validate 校验明文密码是否满足策略。所有设置密码的入口（注册、创建/编辑用户、
// 自助修改、验证码重置、过期改密）都必须调用它。
func (p PasswordPolicy) Validate(password string) error {
	if utf8.RuneCountInString(password) < p.MinLength || password == "" {
		return apperror.ErrPasswordTooShort
	}
	if len(password) > MaxPasswordBytes {
		return apperror.ErrPasswordTooLong
	}
	if p.RequireComplexity && passwordClasses(password) < passwordComplexityClasses {
		return apperror.ErrPasswordTooWeak
	}
	return nil
}

// CredentialExpiry 返回在 now 设置的密码何时过期；策略不限有效期时返回零值（永不过期）。
func (p PasswordPolicy) CredentialExpiry(now time.Time) time.Time {
	if p.MaxAgeDays <= 0 {
		return time.Time{}
	}
	return now.AddDate(0, 0, p.MaxAgeDays)
}

func passwordClasses(password string) int {
	var lower, upper, digit, symbol bool
	for _, r := range password {
		switch {
		case unicode.IsLower(r):
			lower = true
		case unicode.IsUpper(r):
			upper = true
		case unicode.IsDigit(r):
			digit = true
		case !unicode.IsSpace(r):
			symbol = true
		}
	}
	count := 0
	for _, present := range []bool{lower, upper, digit, symbol} {
		if present {
			count++
		}
	}
	return count
}

// PasswordPolicy 从系统配置读取密码策略；单项读取失败时退回安全的默认值。
func (h *SettingHelper) PasswordPolicy(ctx context.Context) PasswordPolicy {
	return PasswordPolicy{
		MinLength:         h.positiveInt(ctx, setting.KeySecurityPasswordMinLength, DefaultPasswordMinLength),
		RequireComplexity: h.boolValue(ctx, setting.KeySecurityPasswordComplexity, true),
		MaxAgeDays:        h.positiveInt(ctx, setting.KeySecurityPasswordMaxAgeDays, 0),
	}
}

// ValidatePassword 按当前密码策略校验明文密码。
func (h *SettingHelper) ValidatePassword(ctx context.Context, password string) error {
	return h.PasswordPolicy(ctx).Validate(password)
}

func (h *SettingHelper) positiveInt(ctx context.Context, key string, fallback int) int {
	item, err := h.settingRepo.GetByKey(ctx, key)
	if err != nil {
		logger.Warnf("Failed to get setting %s, using default (%d): %v", key, fallback, err)
		return fallback
	}
	var value int
	if _, err := fmt.Sscanf(item.Value, "%d", &value); err != nil || value < 0 {
		logger.Warnf("Invalid setting %s value=%q, using default (%d)", key, item.Value, fallback)
		return fallback
	}
	if value == 0 && fallback > 0 {
		return fallback
	}
	return value
}

func (h *SettingHelper) boolValue(ctx context.Context, key string, fallback bool) bool {
	item, err := h.settingRepo.GetByKey(ctx, key)
	if err != nil {
		logger.Warnf("Failed to get setting %s, using default (%t): %v", key, fallback, err)
		return fallback
	}
	switch item.Value {
	case "true", "1":
		return true
	case "false", "0":
		return false
	default:
		logger.Warnf("Invalid setting %s value=%q, using default (%t)", key, item.Value, fallback)
		return fallback
	}
}

// IsRegisterEnabled 从系统配置获取注册开关
// 配置 key: feature.register.enabled
// fallback: 当数据库配置读取失败时使用的回退值
func (h *SettingHelper) IsRegisterEnabled(ctx context.Context, fallback bool) bool {
	settingItem, err := h.settingRepo.GetByKey(ctx, setting.KeyFeatureRegisterEnabled)
	if err != nil {
		logger.Warnf("Failed to get register setting from database, using fallback: %v", err)
		return fallback
	}
	return settingItem.Value == "true" || settingItem.Value == "1"
}
