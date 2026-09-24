// Package sso 通过外部 OpenID Connect 身份提供方登录。
package sso

import (
	"time"

	"github.com/castorworks/castor/internal/domain/shared"
)

// Provider 一个 OIDC 身份提供方（如 Keycloak、Okta、Azure AD、Google）。
type Provider struct {
	ID uint `json:"id"`
	// Code 出现在登录与回调地址里的标识（小写字母、数字与 -）
	Code string `json:"code"`
	// Name 登录按钮上的名称，四种语言
	Name     shared.I18nText `json:"name"`
	Issuer   string          `json:"issuer"`
	ClientID string          `json:"clientId"`
	// ClientSecretCiphertext 经 secretbox 加密的客户端密钥；从不返回给前端
	ClientSecretCiphertext string   `json:"-"`
	Scopes                 []string `json:"scopes"`
	// UsernameClaim 自动注册时取用户名的声明，缺省 preferred_username
	UsernameClaim string `json:"usernameClaim"`
	// AutoRegister 没有关联账号的人首次登录时自动创建账号；关闭时只能登录已关联的账号
	AutoRegister bool      `json:"autoRegister"`
	IsEnabled    bool      `json:"isEnabled"`
	SortOrder    int       `json:"sortOrder"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

// Identity 本系统用户与某个提供方账号（subject）的关联。
type Identity struct {
	ID          uint       `json:"id"`
	UserID      uint       `json:"userId"`
	ProviderID  uint       `json:"providerId"`
	Subject     string     `json:"subject"`
	Email       string     `json:"email"`
	CreatedAt   time.Time  `json:"createdAt"`
	LastLoginAt *time.Time `json:"lastLoginAt"`
}
