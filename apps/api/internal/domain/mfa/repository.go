package mfa

import "context"

// Repository TOTP 设置的持久化。并发敏感的操作（推进时间步、消费恢复码）必须是原子的：
// 同一个码或恢复码在多个副本上同时提交时，只能有一个成功。
type Repository interface {
	// Get 读取设置；没有时返回 shared.ErrNotFound
	Get(ctx context.Context, userID uint) (*TOTP, error)
	// SavePending 写入（或覆盖）尚未启用的密钥；已启用时返回 false
	SavePending(ctx context.Context, userID uint, secretCiphertext string) (bool, error)
	// Enable 启用：记录恢复码与确认时用掉的时间步；只对未启用的记录生效
	Enable(ctx context.Context, userID uint, recoveryCodeHashes []string, step int64) (bool, error)
	Delete(ctx context.Context, userID uint) error
	// AdvanceStep 把最近使用的时间步推进到 step；step 不晚于已记录的值时返回 false（重放）
	AdvanceStep(ctx context.Context, userID uint, step int64) (bool, error)
	// ConsumeRecoveryCode 删除一个未用过的恢复码；不存在时返回 false
	ConsumeRecoveryCode(ctx context.Context, userID uint, hash string) (bool, error)
	ReplaceRecoveryCodes(ctx context.Context, userID uint, hashes []string) error
	// EnabledAmong 返回给定用户中已启用 TOTP 的那些
	EnabledAmong(ctx context.Context, userIDs []uint) (map[uint]bool, error)
}
