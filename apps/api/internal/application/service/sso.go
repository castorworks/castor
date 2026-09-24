package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/castorworks/castor/internal/application/apperror"
	"github.com/castorworks/castor/internal/application/dto"
	"github.com/castorworks/castor/internal/domain/shared"
	"github.com/castorworks/castor/internal/domain/sso"
	"github.com/castorworks/castor/internal/domain/user"
	"github.com/castorworks/castor/internal/pkg/constant"
	"github.com/castorworks/castor/internal/pkg/rediskey"
	"github.com/castorworks/castor/internal/pkg/secretbox"
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/redis/go-redis/v9"
	"golang.org/x/oauth2"
)

const (
	// ssoStateTTL 从跳转到身份提供方到回调的期限
	ssoStateTTL = 10 * time.Minute
	// ssoDiscoveryTTL 发现文档与签名公钥的缓存时长（go-oidc 另会在密钥轮换时自动刷新）
	ssoDiscoveryTTL = time.Hour
	// ssoDefaultUsernameClaim 自动注册取用户名的缺省声明
	ssoDefaultUsernameClaim = "preferred_username"
)

// SSO 登录流程的用途
const (
	SSOModeLogin = "login"
	SSOModeLink  = "link"
)

var providerCodePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,31}$`)

// SSOBegin 发起一次 OIDC 跳转
type SSOBegin struct {
	Mode string
	// Redirect 完成后回到的站内路径（已校验为相对路径）
	Redirect string
	Remember bool
	// UserID 关联模式下是当前登录用户
	UserID uint
}

// SSOResult 回调处理结果
type SSOResult struct {
	Mode     string
	User     *user.User
	Provider *sso.Provider
	Redirect string
	Remember bool
	// Created 本次自动注册了新账号
	Created bool
}

type ssoState struct {
	ProviderID uint   `json:"providerId"`
	Mode       string `json:"mode"`
	UserID     uint   `json:"userId"`
	Redirect   string `json:"redirect"`
	Remember   bool   `json:"remember"`
	Nonce      string `json:"nonce"`
	Verifier   string `json:"verifier"`
}

// SSOService OIDC 单点登录：身份提供方管理、登录/关联跳转与回调、账号关联。
type SSOService interface {
	ListProviders(ctx context.Context) ([]dto.OIDCProviderResp, error)
	SaveProvider(ctx context.Context, id uint, req *dto.OIDCProviderReq) (*dto.OIDCProviderResp, error)
	DeleteProvider(ctx context.Context, id uint) (*sso.Provider, error)
	// EnabledProviders 登录页展示的身份提供方
	EnabledProviders(ctx context.Context) ([]dto.OIDCProviderPublicResp, error)

	// Begin 生成跳转到身份提供方的地址
	Begin(ctx context.Context, providerCode string, begin SSOBegin) (string, error)
	// Complete 处理回调：校验 state 与 ID token，找到（或注册、关联）本系统用户
	Complete(ctx context.Context, providerCode, state, code string) (*SSOResult, error)

	ListIdentities(ctx context.Context, userID uint) ([]dto.UserIdentityResp, error)
	// Unlink 解除关联；这是账号唯一的登录方式时拒绝
	Unlink(ctx context.Context, u *user.User, providerCode string) error
}

type discovered struct {
	provider *oidc.Provider
	issuer   string
	expires  time.Time
}

type ssoService struct {
	repo   sso.Repository
	users  user.Repository
	create UserService
	box    *secretbox.Box
	redis  redis.UniversalClient
	keys   rediskey.Namespace
	policy SSOPolicy

	mu        sync.Mutex
	discovery map[uint]discovered
}

// NewSSOService 创建 OIDC 单点登录服务
func NewSSOService(repo sso.Repository, users user.Repository, userService UserService, keyring *secretbox.Keyring,
	rdb redis.UniversalClient, ns rediskey.Namespace, policy SSOPolicy) (SSOService, error) {
	box, err := keyring.Box("oidc")
	if err != nil {
		return nil, err
	}
	return &ssoService{repo: repo, users: users, create: userService, box: box, redis: rdb, keys: ns, policy: policy,
		discovery: map[uint]discovered{}}, nil
}

func (s *ssoService) callbackURL(code string) string {
	if s.policy.PublicURL == "" {
		return ""
	}
	return strings.TrimRight(s.policy.PublicURL, "/") + "/api/v1/auth/oidc/" + code + "/callback"
}

func (s *ssoService) toResp(p *sso.Provider) dto.OIDCProviderResp {
	return dto.OIDCProviderResp{Provider: *p, HasClientSecret: p.ClientSecretCiphertext != "", CallbackURL: s.callbackURL(p.Code)}
}

func (s *ssoService) ListProviders(ctx context.Context) ([]dto.OIDCProviderResp, error) {
	items, err := s.repo.ListProviders(ctx)
	if err != nil {
		return nil, err
	}
	resp := make([]dto.OIDCProviderResp, len(items))
	for i := range items {
		resp[i] = s.toResp(&items[i])
	}
	return resp, nil
}

func (s *ssoService) validate(req *dto.OIDCProviderReq) error {
	if !providerCodePattern.MatchString(req.Code) || !req.Name.ToI18nText().IsComplete() || strings.TrimSpace(req.ClientID) == "" {
		return apperror.ErrInvalidOIDCProvider
	}
	issuer, err := url.Parse(req.Issuer)
	if err != nil || issuer.Host == "" || (issuer.Scheme != "https" && !(issuer.Scheme == "http" && s.policy.AllowInsecureIssuer)) {
		return apperror.ErrInvalidOIDCProvider
	}
	if !slices.Contains(req.Scopes, oidc.ScopeOpenID) {
		return apperror.ErrInvalidOIDCProvider
	}
	return nil
}

func (s *ssoService) SaveProvider(ctx context.Context, id uint, req *dto.OIDCProviderReq) (*dto.OIDCProviderResp, error) {
	req.Code = strings.TrimSpace(req.Code)
	req.Issuer = strings.TrimRight(strings.TrimSpace(req.Issuer), "/")
	req.Scopes = normalizeScopes(req.Scopes)
	if err := s.validate(req); err != nil {
		return nil, err
	}
	p := &sso.Provider{}
	if id != 0 {
		existing, err := s.repo.GetProvider(ctx, id)
		if errors.Is(err, shared.ErrNotFound) {
			return nil, apperror.ErrOIDCProviderNotFound
		}
		if err != nil {
			return nil, err
		}
		p = existing
	}
	p.Code, p.Name, p.Issuer, p.ClientID, p.Scopes = req.Code, req.Name.ToI18nText(), req.Issuer, strings.TrimSpace(req.ClientID), req.Scopes
	p.UsernameClaim = strings.TrimSpace(req.UsernameClaim)
	if p.UsernameClaim == "" {
		p.UsernameClaim = ssoDefaultUsernameClaim
	}
	p.AutoRegister, p.IsEnabled, p.SortOrder = req.AutoRegister, req.IsEnabled, req.SortOrder
	// 客户端密钥只写不读：更新时留空表示不变。
	if req.ClientSecret != "" {
		sealed, err := s.box.Seal(req.ClientSecret)
		if err != nil {
			return nil, err
		}
		p.ClientSecretCiphertext = sealed
	}
	if p.ClientSecretCiphertext == "" {
		return nil, apperror.ErrInvalidOIDCProvider
	}
	if p.IsEnabled {
		if s.policy.PublicURL == "" {
			return nil, apperror.ErrOIDCNotConfigured
		}
		// 启用前确认 issuer 真的是一个 OIDC 提供方，免得登录页挂出一个点了就报错的按钮。
		if _, err := oidc.NewProvider(oidc.ClientContext(ctx, nil), p.Issuer); err != nil {
			return nil, fmt.Errorf("%w: %v", apperror.ErrOIDCDiscoveryFailed, err)
		}
	}
	if err := s.repo.SaveProvider(ctx, p); err != nil {
		if errors.Is(err, shared.ErrDuplicate) {
			return nil, apperror.ErrOIDCProviderConflict
		}
		return nil, err
	}
	s.forget(p.ID)
	resp := s.toResp(p)
	return &resp, nil
}

func (s *ssoService) DeleteProvider(ctx context.Context, id uint) (*sso.Provider, error) {
	p, err := s.repo.GetProvider(ctx, id)
	if errors.Is(err, shared.ErrNotFound) {
		return nil, apperror.ErrOIDCProviderNotFound
	}
	if err != nil {
		return nil, err
	}
	if err := s.repo.DeleteProvider(ctx, id); err != nil {
		return nil, err
	}
	s.forget(id)
	return p, nil
}

func (s *ssoService) EnabledProviders(ctx context.Context) ([]dto.OIDCProviderPublicResp, error) {
	items, err := s.repo.ListProviders(ctx)
	if err != nil {
		return nil, err
	}
	resp := []dto.OIDCProviderPublicResp{}
	if s.policy.PublicURL == "" {
		return resp, nil
	}
	for _, p := range items {
		if p.IsEnabled {
			resp = append(resp, dto.OIDCProviderPublicResp{Code: p.Code, Name: p.Name})
		}
	}
	return resp, nil
}

func (s *ssoService) forget(id uint) {
	s.mu.Lock()
	delete(s.discovery, id)
	s.mu.Unlock()
}

// providerFor 读取启用的提供方及其发现文档（缓存）
func (s *ssoService) providerFor(ctx context.Context, code string) (*sso.Provider, *oidc.Provider, error) {
	p, err := s.repo.GetProviderByCode(ctx, code)
	if errors.Is(err, shared.ErrNotFound) || (err == nil && !p.IsEnabled) {
		return nil, nil, apperror.ErrOIDCProviderNotFound
	}
	if err != nil {
		return nil, nil, err
	}
	if s.policy.PublicURL == "" {
		return nil, nil, apperror.ErrOIDCNotConfigured
	}
	s.mu.Lock()
	cached, ok := s.discovery[p.ID]
	s.mu.Unlock()
	if ok && cached.issuer == p.Issuer && time.Now().Before(cached.expires) {
		return p, cached.provider, nil
	}
	// 发现文档与之后的 JWKS 请求不能绑在单个请求的 ctx 上：provider 会被缓存复用。
	provider, err := oidc.NewProvider(context.WithoutCancel(ctx), p.Issuer)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: %v", apperror.ErrOIDCDiscoveryFailed, err)
	}
	s.mu.Lock()
	s.discovery[p.ID] = discovered{provider: provider, issuer: p.Issuer, expires: time.Now().Add(ssoDiscoveryTTL)}
	s.mu.Unlock()
	return p, provider, nil
}

func (s *ssoService) oauthConfig(p *sso.Provider, provider *oidc.Provider) (*oauth2.Config, error) {
	secret, err := s.box.Open(p.ClientSecretCiphertext)
	if err != nil {
		return nil, err
	}
	return &oauth2.Config{
		ClientID: p.ClientID, ClientSecret: secret, Endpoint: provider.Endpoint(),
		RedirectURL: s.callbackURL(p.Code), Scopes: p.Scopes,
	}, nil
}

func (s *ssoService) stateKey(state string) string {
	return s.keys.Key("oidc:state:" + state)
}

func randomToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func (s *ssoService) Begin(ctx context.Context, providerCode string, begin SSOBegin) (string, error) {
	p, provider, err := s.providerFor(ctx, providerCode)
	if err != nil {
		return "", err
	}
	cfg, err := s.oauthConfig(p, provider)
	if err != nil {
		return "", err
	}
	state, err := randomToken()
	if err != nil {
		return "", err
	}
	nonce, err := randomToken()
	if err != nil {
		return "", err
	}
	verifier := oauth2.GenerateVerifier()
	payload, err := json.Marshal(ssoState{
		ProviderID: p.ID, Mode: begin.Mode, UserID: begin.UserID, Redirect: begin.Redirect, Remember: begin.Remember,
		Nonce: nonce, Verifier: verifier,
	})
	if err != nil {
		return "", err
	}
	if err := s.redis.Set(ctx, s.stateKey(state), payload, ssoStateTTL).Err(); err != nil {
		return "", err
	}
	return cfg.AuthCodeURL(state, oidc.Nonce(nonce), oauth2.S256ChallengeOption(verifier)), nil
}

// ssoClaims ID token 里用到的声明
type ssoClaims struct {
	Subject       string `json:"sub"`
	Email         string `json:"email"`
	EmailVerified any    `json:"email_verified"`
	Name          string `json:"name"`
}

func (c ssoClaims) emailVerified() bool {
	switch v := c.EmailVerified.(type) {
	case bool:
		return v
	case string:
		return v == "true"
	}
	return false
}

func (s *ssoService) Complete(ctx context.Context, providerCode, state, code string) (*SSOResult, error) {
	if state == "" || code == "" {
		return nil, apperror.ErrOIDCStateInvalid
	}
	// state 只能用一次：GETDEL 保证同一回调地址重放无效。
	payload, err := s.redis.GetDel(ctx, s.stateKey(state)).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, apperror.ErrOIDCStateInvalid
	}
	if err != nil {
		return nil, err
	}
	var st ssoState
	if err := json.Unmarshal(payload, &st); err != nil {
		return nil, apperror.ErrOIDCStateInvalid
	}
	p, provider, err := s.providerFor(ctx, providerCode)
	if err != nil {
		return nil, err
	}
	if p.ID != st.ProviderID {
		return nil, apperror.ErrOIDCStateInvalid
	}
	cfg, err := s.oauthConfig(p, provider)
	if err != nil {
		return nil, err
	}
	token, err := cfg.Exchange(ctx, code, oauth2.VerifierOption(st.Verifier))
	if err != nil {
		return nil, fmt.Errorf("%w: token exchange: %v", apperror.ErrOIDCAuthFailed, err)
	}
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok || rawIDToken == "" {
		return nil, fmt.Errorf("%w: no id_token in the token response", apperror.ErrOIDCAuthFailed)
	}
	idToken, err := provider.Verifier(&oidc.Config{ClientID: p.ClientID}).Verify(ctx, rawIDToken)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", apperror.ErrOIDCAuthFailed, err)
	}
	if idToken.Nonce != st.Nonce {
		return nil, fmt.Errorf("%w: nonce mismatch", apperror.ErrOIDCAuthFailed)
	}
	var claims ssoClaims
	var all map[string]any
	if err := idToken.Claims(&claims); err != nil || claims.Subject == "" {
		return nil, fmt.Errorf("%w: invalid claims", apperror.ErrOIDCAuthFailed)
	}
	_ = idToken.Claims(&all)

	result := &SSOResult{Mode: st.Mode, Provider: p, Redirect: st.Redirect, Remember: st.Remember}
	identity, err := s.repo.FindIdentity(ctx, p.ID, claims.Subject)
	if err != nil && !errors.Is(err, shared.ErrNotFound) {
		return nil, err
	}

	if st.Mode == SSOModeLink {
		if identity != nil && identity.UserID != st.UserID {
			return result, apperror.ErrOIDCIdentityInUse
		}
		u, err := s.users.Get(ctx, st.UserID)
		if err != nil {
			return result, apperror.ErrOIDCStateInvalid
		}
		result.User = u
		if identity != nil {
			return result, nil
		}
		err = s.repo.CreateIdentity(ctx, &sso.Identity{UserID: u.ID, ProviderID: p.ID, Subject: claims.Subject, Email: claims.Email})
		if errors.Is(err, shared.ErrDuplicate) {
			// 同一提供方已关联了另一个账号，或同一账号并发关联。
			return result, apperror.ErrOIDCIdentityInUse
		}
		return result, err
	}

	if identity != nil {
		u, err := s.users.Get(ctx, identity.UserID)
		if err != nil {
			return result, apperror.ErrOIDCAccountNotLinked
		}
		result.User = u
		_ = s.repo.TouchIdentity(ctx, identity.ID)
		return result, ValidateUserStatus(u)
	}
	// 没有关联：只有提供方允许自动注册时才建账号。不按邮箱自动关联已有账号——
	// 否则任何能在身份提供方上注册同一邮箱的人都能接管这个账号。
	if !p.AutoRegister {
		return result, apperror.ErrOIDCAccountNotLinked
	}
	u, err := s.register(ctx, p, claims, all)
	if err != nil {
		return result, err
	}
	now := time.Now()
	if err := s.repo.CreateIdentity(ctx, &sso.Identity{UserID: u.ID, ProviderID: p.ID, Subject: claims.Subject, Email: claims.Email, LastLoginAt: &now}); err != nil {
		// 并发的首次登录已经为这个 subject 建了账号：以已建的为准。
		if errors.Is(err, shared.ErrDuplicate) {
			if existing, findErr := s.repo.FindIdentity(ctx, p.ID, claims.Subject); findErr == nil {
				_ = s.users.Delete(ctx, u.ID)
				if winner, getErr := s.users.Get(ctx, existing.UserID); getErr == nil {
					result.User = winner
					return result, ValidateUserStatus(winner)
				}
			}
		}
		return result, err
	}
	result.User, result.Created = u, true
	return result, nil
}

// register 为首次登录的提供方账号创建本系统用户
func (s *ssoService) register(ctx context.Context, p *sso.Provider, claims ssoClaims, all map[string]any) (*user.User, error) {
	base := ""
	if v, ok := all[p.UsernameClaim].(string); ok {
		base = strings.TrimSpace(v)
	}
	if base == "" {
		base = strings.TrimSpace(strings.Split(claims.Email, "@")[0])
	}
	if base == "" {
		base = p.Code + "-user"
	}
	if utf8.RuneCountInString(base) < 3 {
		base = p.Code + "-" + base
	}
	if runes := []rune(base); len(runes) > 90 {
		base = string(runes[:90])
	}
	username := ""
	for i := 0; i < 20; i++ {
		candidate := base
		if i > 0 {
			candidate = fmt.Sprintf("%s-%d", base, i+1)
		}
		if candidate == constant.CASTOR_RESERVED_USER_ROLE_NAME {
			continue
		}
		if _, err := s.users.GetByUsername(ctx, candidate); errors.Is(err, shared.ErrNotFound) {
			username = candidate
			break
		} else if err != nil {
			return nil, err
		}
	}
	if username == "" {
		suffix, err := randomToken()
		if err != nil {
			return nil, err
		}
		username = base + "-" + strings.ToLower(suffix[:8])
	}
	name := strings.TrimSpace(claims.Name)
	if utf8.RuneCountInString(name) > 100 {
		name = string([]rune(name)[:100])
	}
	u := &user.User{Username: username, Name: name, AccountSource: constant.ACCOUNT_SOURCE_OIDC}
	// 只采用身份提供方确认过、且没被别的账号占用的邮箱。
	if claims.Email != "" && claims.emailVerified() && validateContact(ContactTypeEmail, claims.Email) == nil {
		if _, err := s.users.GetByEmail(ctx, claims.Email); errors.Is(err, shared.ErrNotFound) {
			u.Email, u.EmailVerified = claims.Email, true
		}
	}
	if err := s.create.CreateFederated(ctx, u); err != nil {
		return nil, err
	}
	return u, nil
}

func (s *ssoService) ListIdentities(ctx context.Context, userID uint) ([]dto.UserIdentityResp, error) {
	identities, err := s.repo.ListIdentities(ctx, userID)
	if err != nil {
		return nil, err
	}
	providers, err := s.repo.ListProviders(ctx)
	if err != nil {
		return nil, err
	}
	linked := map[uint]sso.Identity{}
	for _, identity := range identities {
		linked[identity.ProviderID] = identity
	}
	resp := []dto.UserIdentityResp{}
	for _, p := range providers {
		identity, isLinked := linked[p.ID]
		// 停用的提供方只在已关联时列出（方便解除），不再提供新的关联入口。
		if !p.IsEnabled && !isLinked {
			continue
		}
		item := dto.UserIdentityResp{ProviderCode: p.Code, ProviderName: p.Name, Linked: isLinked, ProviderEnabled: p.IsEnabled}
		if isLinked {
			item.Email, item.LinkedAt, item.LastLoginAt = identity.Email, &identity.CreatedAt, identity.LastLoginAt
		}
		resp = append(resp, item)
	}
	return resp, nil
}

func (s *ssoService) Unlink(ctx context.Context, u *user.User, providerCode string) error {
	p, err := s.repo.GetProviderByCode(ctx, providerCode)
	if errors.Is(err, shared.ErrNotFound) {
		return apperror.ErrOIDCProviderNotFound
	}
	if err != nil {
		return err
	}
	identities, err := s.repo.ListIdentities(ctx, u.ID)
	if err != nil {
		return err
	}
	// 没有密码、又只剩这一个关联：解除后就再也登录不了。
	if u.Password == "" && len(identities) <= 1 {
		return apperror.ErrOIDCLastSignInMethod
	}
	deleted, err := s.repo.DeleteIdentity(ctx, u.ID, p.ID)
	if err != nil {
		return err
	}
	if !deleted {
		return apperror.ErrOIDCProviderNotFound
	}
	return nil
}

func normalizeScopes(scopes []string) []string {
	out := []string{}
	for _, scope := range scopes {
		for _, field := range strings.Fields(scope) {
			if !slices.Contains(out, field) {
				out = append(out, field)
			}
		}
	}
	return out
}
