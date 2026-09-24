package persistence

import (
	"context"
	"encoding/json"
	"slices"
	"time"

	"github.com/castorworks/castor/internal/domain/mfa"
	"github.com/castorworks/castor/internal/infrastructure/persistence/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type mfaRepository struct{ db *gorm.DB }

func NewMFARepository(db *gorm.DB) mfa.Repository {
	return &mfaRepository{db: db}
}

func (r *mfaRepository) Get(ctx context.Context, userID uint) (*mfa.TOTP, error) {
	var row models.UserTOTPModel
	if err := translateError(r.db.WithContext(ctx).Where("user_id = ?", userID).First(&row).Error); err != nil {
		return nil, err
	}
	return row.ToEntity()
}

func (r *mfaRepository) SavePending(ctx context.Context, userID uint, secretCiphertext string) (bool, error) {
	row := models.UserTOTPModel{UserID: userID, SecretCiphertext: secretCiphertext, RecoveryCodeHashes: "[]"}
	// 已启用的记录不能被新的绑定覆盖：WHERE 条件让冲突更新只作用于未启用的行。
	result := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.Assignments(map[string]any{
			"secret_ciphertext": secretCiphertext, "last_used_step": 0, "recovery_code_hashes": "[]", "updated_at": time.Now(),
		}),
		Where: clause.Where{Exprs: []clause.Expression{clause.Expr{SQL: "user_totp.enabled = false"}}},
	}).Create(&row)
	return result.RowsAffected == 1, result.Error
}

func (r *mfaRepository) Enable(ctx context.Context, userID uint, hashes []string, step int64) (bool, error) {
	encoded, err := json.Marshal(hashes)
	if err != nil {
		return false, err
	}
	now := time.Now()
	result := r.db.WithContext(ctx).Model(&models.UserTOTPModel{}).
		Where("user_id = ? AND enabled = false", userID).
		Updates(map[string]any{"enabled": true, "enabled_at": now, "last_used_step": step, "recovery_code_hashes": string(encoded)})
	return result.RowsAffected == 1, result.Error
}

func (r *mfaRepository) Delete(ctx context.Context, userID uint) error {
	return r.db.WithContext(ctx).Where("user_id = ?", userID).Delete(&models.UserTOTPModel{}).Error
}

func (r *mfaRepository) AdvanceStep(ctx context.Context, userID uint, step int64) (bool, error) {
	result := r.db.WithContext(ctx).Model(&models.UserTOTPModel{}).
		Where("user_id = ? AND enabled = true AND last_used_step < ?", userID, step).
		Update("last_used_step", step)
	return result.RowsAffected == 1, result.Error
}

func (r *mfaRepository) ConsumeRecoveryCode(ctx context.Context, userID uint, hash string) (bool, error) {
	consumed := false
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row models.UserTOTPModel
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("user_id = ? AND enabled = true", userID).First(&row).Error; err != nil {
			return translateError(err)
		}
		var hashes []string
		if err := json.Unmarshal([]byte(row.RecoveryCodeHashes), &hashes); err != nil {
			return err
		}
		i := slices.Index(hashes, hash)
		if i < 0 {
			return nil
		}
		encoded, err := json.Marshal(slices.Delete(hashes, i, i+1))
		if err != nil {
			return err
		}
		consumed = true
		return tx.Model(&models.UserTOTPModel{}).Where("user_id = ?", userID).Update("recovery_code_hashes", string(encoded)).Error
	})
	return consumed, err
}

func (r *mfaRepository) ReplaceRecoveryCodes(ctx context.Context, userID uint, hashes []string) error {
	encoded, err := json.Marshal(hashes)
	if err != nil {
		return err
	}
	return r.db.WithContext(ctx).Model(&models.UserTOTPModel{}).
		Where("user_id = ? AND enabled = true", userID).Update("recovery_code_hashes", string(encoded)).Error
}

func (r *mfaRepository) EnabledAmong(ctx context.Context, userIDs []uint) (map[uint]bool, error) {
	result := map[uint]bool{}
	if len(userIDs) == 0 {
		return result, nil
	}
	var ids []uint
	if err := r.db.WithContext(ctx).Model(&models.UserTOTPModel{}).
		Where("user_id IN ? AND enabled = true", userIDs).Pluck("user_id", &ids).Error; err != nil {
		return nil, err
	}
	for _, id := range ids {
		result[id] = true
	}
	return result, nil
}
