package dto

import "time"

// TOTPStatusResp 当前用户的两步验证状态
type TOTPStatusResp struct {
	Enabled                bool       `json:"enabled"`
	EnabledAt              *time.Time `json:"enabledAt"`
	RecoveryCodesRemaining int        `json:"recoveryCodesRemaining"`
}

// TOTPSetupResp 开始绑定：二维码（PNG data URL）与无法扫码时手工输入的密钥。
// 密钥只在这一步返回，之后只以密文保存。
type TOTPSetupResp struct {
	Secret     string `json:"secret"`
	OtpauthURL string `json:"otpauthUrl"`
	QRCode     string `json:"qrCode"`
}

// TOTPCodeReq 提交一次性码（6 位）或恢复码
type TOTPCodeReq struct {
	Code string `json:"code" binding:"required,max=32"`
}

// TOTPDisableReq 关闭两步验证：当前密码（RSA 密文；无密码的 OIDC 账号可空）与一次性码或恢复码
type TOTPDisableReq struct {
	Password string `json:"password" binding:"max=1024"`
	Code     string `json:"code" binding:"required,max=32"`
}

// RecoveryCodesResp 新生成的恢复码，只显示这一次
type RecoveryCodesResp struct {
	RecoveryCodes []string `json:"recoveryCodes"`
}

// TOTPLoginReq 登录第二步：第一步响应里的挑战令牌与一次性码或恢复码
type TOTPLoginReq struct {
	Challenge string `json:"challenge" binding:"required,max=128"`
	Code      string `json:"code" binding:"required,max=32"`
}

// MFAChallengeResp 登录第一步成功但需要两步验证时，错误响应的 data
type MFAChallengeResp struct {
	Challenge string `json:"challenge"`
	ExpiresIn int    `json:"expiresIn"`
}
