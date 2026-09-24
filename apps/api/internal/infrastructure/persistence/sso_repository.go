package persistence

import (
	"context"
	"time"

	"github.com/castorworks/castor/internal/domain/sso"
	"github.com/castorworks/castor/internal/infrastructure/persistence/models"
	"gorm.io/gorm"
)

type ssoRepository struct{ db *gorm.DB }

func NewSSORepository(db *gorm.DB) sso.Repository {
	return &ssoRepository{db: db}
}

func (r *ssoRepository) ListProviders(ctx context.Context) ([]sso.Provider, error) {
	var rows []models.OIDCProviderModel
	if err := r.db.WithContext(ctx).Order("sort_order, id").Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make([]sso.Provider, len(rows))
	for i := range rows {
		result[i] = rows[i].ToEntity()
	}
	return result, nil
}

func (r *ssoRepository) getProvider(ctx context.Context, query string, arg any) (*sso.Provider, error) {
	var row models.OIDCProviderModel
	if err := translateError(r.db.WithContext(ctx).Where(query, arg).First(&row).Error); err != nil {
		return nil, err
	}
	p := row.ToEntity()
	return &p, nil
}

func (r *ssoRepository) GetProvider(ctx context.Context, id uint) (*sso.Provider, error) {
	return r.getProvider(ctx, "id = ?", id)
}

func (r *ssoRepository) GetProviderByCode(ctx context.Context, code string) (*sso.Provider, error) {
	return r.getProvider(ctx, "code = ?", code)
}

func (r *ssoRepository) SaveProvider(ctx context.Context, p *sso.Provider) error {
	row := models.OIDCProviderModelFromEntity(p)
	if err := translateError(r.db.WithContext(ctx).Save(row).Error); err != nil {
		return err
	}
	saved, err := r.GetProvider(ctx, row.ID)
	if err != nil {
		return err
	}
	*p = *saved
	return nil
}

func (r *ssoRepository) DeleteProvider(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&models.OIDCProviderModel{}, id).Error
}

func (r *ssoRepository) FindIdentity(ctx context.Context, providerID uint, subject string) (*sso.Identity, error) {
	var row models.UserIdentityModel
	if err := translateError(r.db.WithContext(ctx).Where("provider_id = ? AND subject = ?", providerID, subject).First(&row).Error); err != nil {
		return nil, err
	}
	identity := row.ToEntity()
	return &identity, nil
}

func (r *ssoRepository) ListIdentities(ctx context.Context, userID uint) ([]sso.Identity, error) {
	var rows []models.UserIdentityModel
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("id").Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make([]sso.Identity, len(rows))
	for i := range rows {
		result[i] = rows[i].ToEntity()
	}
	return result, nil
}

func (r *ssoRepository) CreateIdentity(ctx context.Context, identity *sso.Identity) error {
	row := models.UserIdentityModel{UserID: identity.UserID, ProviderID: identity.ProviderID, Subject: identity.Subject, Email: identity.Email, LastLoginAt: identity.LastLoginAt}
	if err := translateError(r.db.WithContext(ctx).Create(&row).Error); err != nil {
		return err
	}
	*identity = row.ToEntity()
	return nil
}

func (r *ssoRepository) DeleteIdentity(ctx context.Context, userID, providerID uint) (bool, error) {
	result := r.db.WithContext(ctx).Where("user_id = ? AND provider_id = ?", userID, providerID).Delete(&models.UserIdentityModel{})
	return result.RowsAffected > 0, result.Error
}

func (r *ssoRepository) TouchIdentity(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Model(&models.UserIdentityModel{}).Where("id = ?", id).Update("last_login_at", time.Now()).Error
}
