// Package mfa 第二重身份验证（TOTP）。
package mfa

import "time"

// TOTP 是一个用户的 TOTP 设置。开始绑定时写入未启用的记录，用验证器上的一次性码确认后才启用。
type TOTP struct {
	UserID uint
	// SecretCiphertext 经 secretbox 加密的 base32 密钥；仓储只存取密文
	SecretCiphertext string
	Enabled          bool
	// LastUsedStep 最近一次被接受的时间步（Unix 秒 / 30），不接受不晚于它的码，防止同一码重放
	LastUsedStep int64
	// RecoveryCodeHashes 尚未使用的恢复码（SHA-256 十六进制）；用掉一个删一个
	RecoveryCodeHashes []string
	EnabledAt          *time.Time
	UpdatedAt          time.Time
}
