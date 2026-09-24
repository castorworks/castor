package sso

import "context"

// Repository 身份提供方与账号关联的持久化。
type Repository interface {
	ListProviders(ctx context.Context) ([]Provider, error)
	GetProvider(ctx context.Context, id uint) (*Provider, error)
	GetProviderByCode(ctx context.Context, code string) (*Provider, error)
	// SaveProvider 新建或更新；Code 重复返回 shared.ErrDuplicate
	SaveProvider(ctx context.Context, p *Provider) error
	// DeleteProvider 删除提供方及其全部账号关联
	DeleteProvider(ctx context.Context, id uint) error

	// FindIdentity 按提供方与 subject 查关联；没有时返回 shared.ErrNotFound
	FindIdentity(ctx context.Context, providerID uint, subject string) (*Identity, error)
	ListIdentities(ctx context.Context, userID uint) ([]Identity, error)
	// CreateIdentity 新建关联；同一提供方账号或同一用户在同一提供方已有关联时返回 shared.ErrDuplicate
	CreateIdentity(ctx context.Context, identity *Identity) error
	DeleteIdentity(ctx context.Context, userID, providerID uint) (bool, error)
	TouchIdentity(ctx context.Context, id uint) error
}
