package service

import (
	"context"
	"fmt"

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

// ValidatePasswordLength 从系统配置获取密码最小长度并校验
// 配置 key: security.password.minLength
func (h *SettingHelper) ValidatePasswordLength(ctx context.Context, password string) error {
	minLength := h.GetPasswordMinLength(ctx)
	if len(password) < minLength {
		return fmt.Errorf("password length must be at least %d characters", minLength)
	}
	return nil
}

// GetPasswordMinLength 从系统配置获取密码最小长度
func (h *SettingHelper) GetPasswordMinLength(ctx context.Context) int {
	settingItem, err := h.settingRepo.GetByKey(ctx, setting.KeySecurityPasswordMinLength)
	if err != nil {
		logger.Warnf("Failed to get password min length setting, using default (%d): %v", DefaultPasswordMinLength, err)
		return DefaultPasswordMinLength
	}

	var minLength int
	if _, err := fmt.Sscanf(settingItem.Value, "%d", &minLength); err != nil || minLength <= 0 {
		logger.Warnf("Invalid password min length setting value=%q, using default (%d)", settingItem.Value, DefaultPasswordMinLength)
		return DefaultPasswordMinLength
	}

	return minLength
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
