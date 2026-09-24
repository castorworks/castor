package dto

import (
	"time"

	"github.com/castorworks/castor/internal/domain/shared"
	"github.com/castorworks/castor/internal/domain/sso"
)

// OIDCProviderReq 新建或更新身份提供方。ClientSecret 只写：更新时留空表示保持不变。
type OIDCProviderReq struct {
	Code          string      `json:"code" binding:"required,max=32"`
	Name          I18nTextReq `json:"name" binding:"required"`
	Issuer        string      `json:"issuer" binding:"required,max=500"`
	ClientID      string      `json:"clientId" binding:"required,max=255"`
	ClientSecret  string      `json:"clientSecret" binding:"max=512"`
	Scopes        []string    `json:"scopes" binding:"required,min=1,max=20"`
	UsernameClaim string      `json:"usernameClaim" binding:"max=64"`
	AutoRegister  bool        `json:"autoRegister"`
	IsEnabled     bool        `json:"isEnabled"`
	SortOrder     int         `json:"sortOrder"`
}

// OIDCProviderResp 管理端看到的身份提供方：不含客户端密钥，附带要在身份提供方登记的回调地址
type OIDCProviderResp struct {
	sso.Provider
	HasClientSecret bool   `json:"hasClientSecret"`
	CallbackURL     string `json:"callbackUrl"`
}

// OIDCProviderPublicResp 登录页上的身份提供方按钮
type OIDCProviderPublicResp struct {
	Code string          `json:"code"`
	Name shared.I18nText `json:"name"`
}

// UserIdentityResp 当前用户在一个身份提供方上的关联状态
type UserIdentityResp struct {
	ProviderCode    string          `json:"providerCode"`
	ProviderName    shared.I18nText `json:"providerName"`
	ProviderEnabled bool            `json:"providerEnabled"`
	Linked          bool            `json:"linked"`
	Email           string          `json:"email"`
	LinkedAt        *time.Time      `json:"linkedAt"`
	LastLoginAt     *time.Time      `json:"lastLoginAt"`
}

// SSOLinkResp 发起关联：前端把浏览器导航到 authorizeUrl
type SSOLinkResp struct {
	AuthorizeURL string `json:"authorizeUrl"`
}
