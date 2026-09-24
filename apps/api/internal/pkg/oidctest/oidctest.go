// Package oidctest 是测试用的最小 OpenID Connect 身份提供方：发现文档、JWKS、授权与令牌端点。
// 授权端点不显示登录页，直接以预设的用户"同意"并带着授权码跳回；令牌端点校验客户端密钥与 PKCE。
package oidctest

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
)

// User 授权时"登录"身份提供方的人
type User struct {
	Subject           string
	Email             string
	EmailVerified     bool
	PreferredUsername string
	Name              string
}

// Provider 测试身份提供方
type Provider struct {
	*httptest.Server
	ClientID     string
	ClientSecret string

	mu    sync.Mutex
	key   *rsa.PrivateKey
	user  User
	codes map[string]grant
	// NonceOverride 非空时写进 ID token 的 nonce（模拟被篡改或重放的令牌）
	NonceOverride string
	// Audience 非空时替代 ClientID 作为 ID token 的 aud
	Audience string
}

type grant struct {
	user          User
	nonce         string
	challenge     string
	redirectURI   string
	codeChallenge string
}

// New 启动测试身份提供方
func New(clientID, clientSecret string) *Provider {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err)
	}
	p := &Provider{ClientID: clientID, ClientSecret: clientSecret, key: key, codes: map[string]grant{}}
	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/openid-configuration", p.discovery)
	mux.HandleFunc("/jwks", p.jwks)
	mux.HandleFunc("/authorize", p.authorize)
	mux.HandleFunc("/token", p.token)
	p.Server = httptest.NewServer(mux)
	return p
}

// SignIn 设定下一次授权时登录的用户
func (p *Provider) SignIn(u User) {
	p.mu.Lock()
	p.user = u
	p.mu.Unlock()
}

func (p *Provider) discovery(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, map[string]any{
		"issuer": p.URL, "authorization_endpoint": p.URL + "/authorize", "token_endpoint": p.URL + "/token",
		"jwks_uri": p.URL + "/jwks", "response_types_supported": []string{"code"},
		"subject_types_supported": []string{"public"}, "id_token_signing_alg_values_supported": []string{"RS256"},
	})
}

func (p *Provider) jwks(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, jose.JSONWebKeySet{Keys: []jose.JSONWebKey{{Key: &p.key.PublicKey, KeyID: "test", Algorithm: "RS256", Use: "sig"}}})
}

// authorize 直接同意，并带着授权码与原样的 state 跳回 redirect_uri
func (p *Provider) authorize(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	if q.Get("client_id") != p.ClientID || q.Get("response_type") != "code" || q.Get("code_challenge_method") != "S256" {
		http.Error(w, "bad authorization request", http.StatusBadRequest)
		return
	}
	code := randomString()
	p.mu.Lock()
	p.codes[code] = grant{user: p.user, nonce: q.Get("nonce"), redirectURI: q.Get("redirect_uri"), codeChallenge: q.Get("code_challenge")}
	p.mu.Unlock()
	target, _ := url.Parse(q.Get("redirect_uri"))
	values := target.Query()
	values.Set("code", code)
	values.Set("state", q.Get("state"))
	target.RawQuery = values.Encode()
	http.Redirect(w, r, target.String(), http.StatusFound)
}

func (p *Provider) token(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	id, secret, ok := r.BasicAuth()
	if !ok {
		id, secret = r.PostForm.Get("client_id"), r.PostForm.Get("client_secret")
	}
	if id != p.ClientID || secret != p.ClientSecret {
		http.Error(w, `{"error":"invalid_client"}`, http.StatusUnauthorized)
		return
	}
	p.mu.Lock()
	g, found := p.codes[r.PostForm.Get("code")]
	delete(p.codes, r.PostForm.Get("code"))
	nonce, aud := g.nonce, p.ClientID
	if p.NonceOverride != "" {
		nonce = p.NonceOverride
	}
	if p.Audience != "" {
		aud = p.Audience
	}
	p.mu.Unlock()
	sum := sha256.Sum256([]byte(r.PostForm.Get("code_verifier")))
	if !found || g.redirectURI != r.PostForm.Get("redirect_uri") || base64.RawURLEncoding.EncodeToString(sum[:]) != g.codeChallenge {
		http.Error(w, `{"error":"invalid_grant"}`, http.StatusBadRequest)
		return
	}
	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS256, Key: p.key}, (&jose.SignerOptions{}).WithHeader("kid", "test"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	now := time.Now()
	claims := map[string]any{
		"iss": p.URL, "sub": g.user.Subject, "aud": aud, "exp": now.Add(5 * time.Minute).Unix(), "iat": now.Unix(),
		"nonce": nonce, "email": g.user.Email, "email_verified": g.user.EmailVerified,
		"preferred_username": g.user.PreferredUsername, "name": g.user.Name,
	}
	idToken, err := jwt.Signed(signer).Claims(claims).Serialize()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]any{"access_token": randomString(), "token_type": "Bearer", "expires_in": 300, "id_token": idToken})
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func randomString() string {
	buf := make([]byte, 16)
	_, _ = rand.Read(buf)
	return base64.RawURLEncoding.EncodeToString(buf)
}
