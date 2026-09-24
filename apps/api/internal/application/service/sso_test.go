package service

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/castorworks/castor/internal/application/apperror"
	"github.com/castorworks/castor/internal/application/dto"
	"github.com/castorworks/castor/internal/domain/shared"
	"github.com/castorworks/castor/internal/domain/sso"
	"github.com/castorworks/castor/internal/domain/user"
	"github.com/castorworks/castor/internal/pkg/constant"
	"github.com/castorworks/castor/internal/pkg/oidctest"
	"github.com/castorworks/castor/internal/pkg/rediskey"
	"github.com/castorworks/castor/internal/pkg/secretbox"
)

// memSSORepo 是 sso.Repository 的内存实现
type memSSORepo struct {
	mu         sync.Mutex
	providers  map[uint]*sso.Provider
	identities []sso.Identity
	nextID     uint
}

func newMemSSORepo() *memSSORepo { return &memSSORepo{providers: map[uint]*sso.Provider{}} }

func (r *memSSORepo) ListProviders(context.Context) ([]sso.Provider, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []sso.Provider
	for _, p := range r.providers {
		out = append(out, *p)
	}
	return out, nil
}

func (r *memSSORepo) GetProvider(_ context.Context, id uint) (*sso.Provider, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if p, ok := r.providers[id]; ok {
		c := *p
		return &c, nil
	}
	return nil, shared.ErrNotFound
}

func (r *memSSORepo) GetProviderByCode(_ context.Context, code string) (*sso.Provider, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, p := range r.providers {
		if p.Code == code {
			c := *p
			return &c, nil
		}
	}
	return nil, shared.ErrNotFound
}

func (r *memSSORepo) SaveProvider(_ context.Context, p *sso.Provider) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for id, other := range r.providers {
		if other.Code == p.Code && id != p.ID {
			return shared.ErrDuplicate
		}
	}
	if p.ID == 0 {
		r.nextID++
		p.ID = r.nextID
	}
	c := *p
	r.providers[p.ID] = &c
	return nil
}

func (r *memSSORepo) DeleteProvider(_ context.Context, id uint) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.providers, id)
	return nil
}

func (r *memSSORepo) FindIdentity(_ context.Context, providerID uint, subject string) (*sso.Identity, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, identity := range r.identities {
		if identity.ProviderID == providerID && identity.Subject == subject {
			c := identity
			return &c, nil
		}
	}
	return nil, shared.ErrNotFound
}

func (r *memSSORepo) ListIdentities(_ context.Context, userID uint) ([]sso.Identity, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []sso.Identity
	for _, identity := range r.identities {
		if identity.UserID == userID {
			out = append(out, identity)
		}
	}
	return out, nil
}

func (r *memSSORepo) CreateIdentity(_ context.Context, identity *sso.Identity) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, other := range r.identities {
		if (other.ProviderID == identity.ProviderID && other.Subject == identity.Subject) ||
			(other.ProviderID == identity.ProviderID && other.UserID == identity.UserID) {
			return shared.ErrDuplicate
		}
	}
	r.nextID++
	identity.ID, identity.CreatedAt = r.nextID, time.Now()
	r.identities = append(r.identities, *identity)
	return nil
}

func (r *memSSORepo) DeleteIdentity(_ context.Context, userID, providerID uint) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i, identity := range r.identities {
		if identity.UserID == userID && identity.ProviderID == providerID {
			r.identities = append(r.identities[:i], r.identities[i+1:]...)
			return true, nil
		}
	}
	return false, nil
}

func (r *memSSORepo) TouchIdentity(context.Context, uint) error { return nil }

// federatedUserService 把自动注册的用户写进 mockUserRepo
type federatedUserService struct {
	UserService
	repo *mockUserRepo
}

func (s *federatedUserService) CreateFederated(ctx context.Context, u *user.User) error {
	u.Enable = true
	u.AccountExpireDate = time.Now().Add(time.Hour)
	return s.repo.Create(ctx, u)
}

type ssoFixture struct {
	svc   *ssoService
	repo  *memSSORepo
	users *mockUserRepo
	idp   *oidctest.Provider
}

func newSSOFixture(t *testing.T, autoRegister bool) *ssoFixture {
	t.Helper()
	idp := oidctest.New("castor-client", "client-secret")
	t.Cleanup(idp.Close)
	_, rdb := newJobTestRedis(t)
	ring, _ := secretbox.NewKeyring(strings.Repeat("k", 32))
	repo := newMemSSORepo()
	users := newMockUserRepo()
	svcAny, err := NewSSOService(repo, users, &federatedUserService{repo: users}, ring, rdb, rediskey.Namespace("test"),
		SSOPolicy{PublicURL: "https://castor.test", AllowInsecureIssuer: true})
	if err != nil {
		t.Fatal(err)
	}
	svc := svcAny.(*ssoService)
	if _, err := svc.SaveProvider(context.Background(), 0, &dto.OIDCProviderReq{
		Code: "corp", Name: dto.I18nTextReq{En: "Corp", Zh: "公司", Ja: "会社", Ko: "회사"}, Issuer: idp.URL,
		ClientID: "castor-client", ClientSecret: "client-secret", Scopes: []string{"openid profile", "email"},
		AutoRegister: autoRegister, IsEnabled: true,
	}); err != nil {
		t.Fatalf("SaveProvider: %v", err)
	}
	return &ssoFixture{svc: svc, repo: repo, users: users, idp: idp}
}

// run 走一遍浏览器的往返：Begin → 身份提供方授权 → 回调参数 → Complete
func (f *ssoFixture) run(t *testing.T, begin SSOBegin) (*SSOResult, error) {
	t.Helper()
	target, err := f.svc.Begin(context.Background(), "corp", begin)
	if err != nil {
		t.Fatalf("Begin: %v", err)
	}
	state, code := f.authorize(t, target)
	return f.svc.Complete(context.Background(), "corp", state, code)
}

func (f *ssoFixture) authorize(t *testing.T, target string) (string, string) {
	t.Helper()
	client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := client.Get(target)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	callback, err := url.Parse(resp.Header.Get("Location"))
	if err != nil || resp.StatusCode != http.StatusFound {
		t.Fatalf("authorize answered %d %q", resp.StatusCode, resp.Header.Get("Location"))
	}
	if callback.String()[:len("https://castor.test/api/v1/auth/oidc/corp/callback")] != "https://castor.test/api/v1/auth/oidc/corp/callback" {
		t.Fatalf("callback URL = %s", callback)
	}
	return callback.Query().Get("state"), callback.Query().Get("code")
}

func TestSSO_ProviderSettings(t *testing.T) {
	f := newSSOFixture(t, true)
	ctx := context.Background()
	items, _ := f.svc.ListProviders(ctx)
	if len(items) != 1 || !items[0].HasClientSecret || items[0].CallbackURL != "https://castor.test/api/v1/auth/oidc/corp/callback" ||
		strings.Join(items[0].Scopes, " ") != "openid profile email" || items[0].UsernameClaim != ssoDefaultUsernameClaim {
		t.Fatalf("provider = %+v", items)
	}
	stored := f.repo.providers[items[0].ID]
	if strings.Contains(stored.ClientSecretCiphertext, "client-secret") {
		t.Fatal("the client secret must be stored encrypted")
	}
	// 更新时不填密钥：沿用已存的。
	req := &dto.OIDCProviderReq{Code: "corp", Name: dto.I18nTextReq{En: "Corp SSO", Zh: "公司", Ja: "会社", Ko: "회사"},
		Issuer: f.idp.URL, ClientID: "castor-client", Scopes: []string{"openid"}, IsEnabled: true}
	if _, err := f.svc.SaveProvider(ctx, items[0].ID, req); err != nil || f.repo.providers[items[0].ID].ClientSecretCiphertext != stored.ClientSecretCiphertext {
		t.Fatalf("update without secret: %v", err)
	}
	bad := []dto.OIDCProviderReq{
		{Code: "Bad Code", Name: req.Name, Issuer: f.idp.URL, ClientID: "c", ClientSecret: "s", Scopes: []string{"openid"}},
		{Code: "other", Name: dto.I18nTextReq{En: "Only English"}, Issuer: f.idp.URL, ClientID: "c", ClientSecret: "s", Scopes: []string{"openid"}},
		{Code: "other", Name: req.Name, Issuer: f.idp.URL, ClientID: "c", ClientSecret: "s", Scopes: []string{"profile"}},
		{Code: "other", Name: req.Name, Issuer: "not a url", ClientID: "c", ClientSecret: "s", Scopes: []string{"openid"}},
		{Code: "other", Name: req.Name, Issuer: f.idp.URL, ClientID: "c", Scopes: []string{"openid"}},
	}
	for i, b := range bad {
		if _, err := f.svc.SaveProvider(ctx, 0, &b); !errors.Is(err, apperror.ErrInvalidOIDCProvider) {
			t.Errorf("bad provider %d err = %v", i, err)
		}
	}
	dup := *req
	dup.ClientSecret = "s"
	if _, err := f.svc.SaveProvider(ctx, 0, &dup); !errors.Is(err, apperror.ErrOIDCProviderConflict) {
		t.Errorf("duplicate code err = %v", err)
	}
	unreachable := dto.OIDCProviderReq{Code: "down", Name: req.Name, Issuer: "http://127.0.0.1:1", ClientID: "c", ClientSecret: "s", Scopes: []string{"openid"}, IsEnabled: true}
	if _, err := f.svc.SaveProvider(ctx, 0, &unreachable); !errors.Is(err, apperror.ErrOIDCDiscoveryFailed) {
		t.Errorf("enabling an unreachable issuer err = %v", err)
	}
	public, _ := f.svc.EnabledProviders(ctx)
	if len(public) != 1 || public[0].Code != "corp" {
		t.Errorf("EnabledProviders = %+v", public)
	}
}

func TestSSO_LoginRegistersThenSignsInTheSameAccount(t *testing.T) {
	f := newSSOFixture(t, true)
	f.users.users["alice"] = &user.User{ID: 50, Username: "alice", Enable: true}
	f.idp.SignIn(oidctest.User{Subject: "sub-1", Email: "alice@corp.test", EmailVerified: true, PreferredUsername: "alice", Name: "Alice Corp"})

	first, err := f.run(t, SSOBegin{Mode: SSOModeLogin, Redirect: "/dashboard/users", Remember: true})
	if err != nil {
		t.Fatalf("first login: %v", err)
	}
	if !first.Created || first.User.Username != "alice-2" || first.User.AccountSource != constant.ACCOUNT_SOURCE_OIDC ||
		first.User.Email != "alice@corp.test" || !first.User.EmailVerified || first.User.Name != "Alice Corp" || first.User.Password != "" {
		t.Fatalf("registered user = %+v (existing 'alice' must not be taken over)", first.User)
	}
	if first.Redirect != "/dashboard/users" || !first.Remember {
		t.Fatalf("state round trip lost redirect/remember: %+v", first)
	}
	second, err := f.run(t, SSOBegin{Mode: SSOModeLogin})
	if err != nil || second.Created || second.User.ID != first.User.ID {
		t.Fatalf("second login must find the linked account: %+v, %v", second, err)
	}

	// 未验证的邮箱不进账号；别人已占用的邮箱也不进。
	f.idp.SignIn(oidctest.User{Subject: "sub-2", Email: "bob@corp.test", EmailVerified: false, PreferredUsername: "bob"})
	bob, err := f.run(t, SSOBegin{Mode: SSOModeLogin})
	if err != nil || bob.User.Email != "" {
		t.Fatalf("unverified email must be ignored: %+v, %v", bob.User, err)
	}
	f.idp.SignIn(oidctest.User{Subject: "sub-3", Email: "alice@corp.test", EmailVerified: true, PreferredUsername: "mallory"})
	mallory, err := f.run(t, SSOBegin{Mode: SSOModeLogin})
	if err != nil || mallory.User.Email != "" || mallory.User.ID == first.User.ID {
		t.Fatalf("an email already in use must not be reused or linked: %+v, %v", mallory.User, err)
	}

	// 被禁用的账号不能再通过外部身份登录。
	f.users.users[first.User.Username].Enable = false
	f.idp.SignIn(oidctest.User{Subject: "sub-1"})
	if _, err := f.run(t, SSOBegin{Mode: SSOModeLogin}); !errors.Is(err, apperror.ErrUserDisabled) {
		t.Fatalf("disabled account err = %v", err)
	}
}

func TestSSO_WithoutAutoRegisterOnlyLinkedAccountsSignIn(t *testing.T) {
	f := newSSOFixture(t, false)
	f.users.users["carol"] = &user.User{ID: 60, Username: "carol", Enable: true, Email: "carol@corp.test", EmailVerified: true, Password: "hash"}
	f.idp.SignIn(oidctest.User{Subject: "sub-carol", Email: "carol@corp.test", EmailVerified: true, PreferredUsername: "carol"})

	// 邮箱相同也不自动关联。
	if _, err := f.run(t, SSOBegin{Mode: SSOModeLogin}); !errors.Is(err, apperror.ErrOIDCAccountNotLinked) {
		t.Fatalf("unlinked login err = %v", err)
	}
	linked, err := f.run(t, SSOBegin{Mode: SSOModeLink, UserID: 60, Redirect: "/dashboard/profile"})
	if err != nil || linked.User.ID != 60 {
		t.Fatalf("link: %+v, %v", linked, err)
	}
	login, err := f.run(t, SSOBegin{Mode: SSOModeLogin})
	if err != nil || login.User.ID != 60 {
		t.Fatalf("login after linking: %+v, %v", login, err)
	}
	// 同一提供方账号不能再关联到另一个用户。
	f.users.users["dave"] = &user.User{ID: 61, Username: "dave", Enable: true}
	if _, err := f.run(t, SSOBegin{Mode: SSOModeLink, UserID: 61}); !errors.Is(err, apperror.ErrOIDCIdentityInUse) {
		t.Fatalf("linking a taken identity err = %v", err)
	}
	identities, _ := f.svc.ListIdentities(context.Background(), 60)
	if len(identities) != 1 || !identities[0].Linked || identities[0].Email != "carol@corp.test" {
		t.Fatalf("identities = %+v", identities)
	}
	// 有密码的账号可以解除关联。
	if err := f.svc.Unlink(context.Background(), f.users.users["carol"], "corp"); err != nil {
		t.Fatal(err)
	}
}

func TestSSO_RejectsForgedOrReplayedResponses(t *testing.T) {
	f := newSSOFixture(t, true)
	ctx := context.Background()
	f.idp.SignIn(oidctest.User{Subject: "sub-x", PreferredUsername: "xavier"})

	target, _ := f.svc.Begin(ctx, "corp", SSOBegin{Mode: SSOModeLogin})
	state, code := f.authorize(t, target)
	if _, err := f.svc.Complete(ctx, "corp", "forged-state", code); !errors.Is(err, apperror.ErrOIDCStateInvalid) {
		t.Fatalf("unknown state err = %v", err)
	}
	if _, err := f.svc.Complete(ctx, "corp", state, code); err != nil {
		t.Fatalf("genuine callback: %v", err)
	}
	if _, err := f.svc.Complete(ctx, "corp", state, code); !errors.Is(err, apperror.ErrOIDCStateInvalid) {
		t.Fatalf("a replayed callback must be rejected, err = %v", err)
	}

	f.idp.NonceOverride = "attacker-nonce"
	if _, err := f.run(t, SSOBegin{Mode: SSOModeLogin}); !errors.Is(err, apperror.ErrOIDCAuthFailed) {
		t.Fatalf("nonce mismatch err = %v", err)
	}
	f.idp.NonceOverride, f.idp.Audience = "", "another-client"
	if _, err := f.run(t, SSOBegin{Mode: SSOModeLogin}); !errors.Is(err, apperror.ErrOIDCAuthFailed) {
		t.Fatalf("token for another audience err = %v", err)
	}
	f.idp.Audience = ""
	if _, err := f.svc.Begin(ctx, "missing", SSOBegin{Mode: SSOModeLogin}); !errors.Is(err, apperror.ErrOIDCProviderNotFound) {
		t.Fatalf("unknown provider err = %v", err)
	}
}

func TestSSO_UnlinkKeepsAWayToSignIn(t *testing.T) {
	f := newSSOFixture(t, true)
	f.idp.SignIn(oidctest.User{Subject: "sub-only", PreferredUsername: "erin"})
	result, err := f.run(t, SSOBegin{Mode: SSOModeLogin})
	if err != nil {
		t.Fatal(err)
	}
	if err := f.svc.Unlink(context.Background(), result.User, "corp"); !errors.Is(err, apperror.ErrOIDCLastSignInMethod) {
		t.Fatalf("unlinking the only sign-in method err = %v", err)
	}
}
