package auth

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// OAuthConfig configures the OAuth 2.1 bearer-token resource server. Point it at
// any standards-compliant provider (Auth0, Okta, Cognito, Clerk, Keycloak, ...).
type OAuthConfig struct {
	Issuer     string // expected "iss"
	Audience   string // expected "aud" (optional but recommended)
	JWKSURL    string // provider JWKS endpoint
	PlanClaim  string // claim carrying the plan name (default "plan")
	HTTPClient *http.Client
}

// OAuthAuthenticator validates bearer JWTs and maps them to a Principal.
type OAuthAuthenticator struct {
	cfg  OAuthConfig
	keys *jwksCache
}

func NewOAuthAuthenticator(cfg OAuthConfig) (*OAuthAuthenticator, error) {
	if cfg.Issuer == "" || cfg.JWKSURL == "" {
		return nil, errors.New("oauth requires BINDKIT_OAUTH_ISSUER and BINDKIT_OAUTH_JWKS_URL")
	}
	if cfg.PlanClaim == "" {
		cfg.PlanClaim = "plan"
	}
	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	return &OAuthAuthenticator{cfg: cfg, keys: newJWKSCache(cfg.JWKSURL, client)}, nil
}

// Authenticate validates a bearer token string and returns its Principal.
func (a *OAuthAuthenticator) Authenticate(ctx context.Context, token string) (Principal, error) {
	if token == "" {
		return Principal{}, ErrUnauthorized
	}
	opts := []jwt.ParserOption{
		jwt.WithValidMethods([]string{"RS256", "RS384", "RS512"}),
		jwt.WithIssuer(a.cfg.Issuer),
		jwt.WithExpirationRequired(),
	}
	if a.cfg.Audience != "" {
		opts = append(opts, jwt.WithAudience(a.cfg.Audience))
	}
	parsed, err := jwt.Parse(token, a.keyfunc(ctx), opts...)
	if err != nil || !parsed.Valid {
		return Principal{}, fmt.Errorf("%w: %v", ErrUnauthorized, err)
	}
	claims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok {
		return Principal{}, ErrUnauthorized
	}
	sub, _ := claims["sub"].(string)
	if sub == "" {
		return Principal{}, fmt.Errorf("%w: token missing sub", ErrUnauthorized)
	}
	plan, _ := claims[a.cfg.PlanClaim].(string)
	if plan == "" {
		plan = "free"
	}
	return Principal{Key: sub, Plan: plan}, nil
}

func (a *OAuthAuthenticator) keyfunc(ctx context.Context) jwt.Keyfunc {
	return func(token *jwt.Token) (any, error) {
		kid, _ := token.Header["kid"].(string)
		if kid == "" {
			return nil, errors.New("token missing kid header")
		}
		return a.keys.key(ctx, kid)
	}
}

// jwksCache fetches and caches a provider's RSA signing keys, refreshing on an
// unknown key id (rotation) but no more often than refreshInterval.
type jwksCache struct {
	url             string
	client          *http.Client
	refreshInterval time.Duration

	mu      sync.RWMutex
	keys    map[string]*rsa.PublicKey
	fetched time.Time
}

func newJWKSCache(url string, client *http.Client) *jwksCache {
	return &jwksCache{url: url, client: client, refreshInterval: 30 * time.Second, keys: map[string]*rsa.PublicKey{}}
}

func (c *jwksCache) key(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	c.mu.RLock()
	k := c.keys[kid]
	c.mu.RUnlock()
	if k != nil {
		return k, nil
	}
	if err := c.refresh(ctx); err != nil {
		return nil, err
	}
	c.mu.RLock()
	k = c.keys[kid]
	c.mu.RUnlock()
	if k == nil {
		return nil, fmt.Errorf("unknown signing key id %q", kid)
	}
	return k, nil
}

func (c *jwksCache) refresh(ctx context.Context) error {
	c.mu.RLock()
	fresh := len(c.keys) > 0 && time.Since(c.fetched) < c.refreshInterval
	c.mu.RUnlock()
	if fresh {
		return nil
	}
	// Fetch outside the lock so a slow or hanging JWKS endpoint cannot block
	// concurrent token validation (including requests that would hit the cache).
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url, nil)
	if err != nil {
		return err
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("fetch jwks: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("fetch jwks: status %d", resp.StatusCode)
	}
	var doc struct {
		Keys []jwk `json:"keys"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return fmt.Errorf("decode jwks: %w", err)
	}
	keys := map[string]*rsa.PublicKey{}
	for _, k := range doc.Keys {
		if k.Kty != "RSA" || k.Kid == "" {
			continue
		}
		pub, err := k.rsaPublicKey()
		if err != nil {
			continue
		}
		keys[k.Kid] = pub
	}
	if len(keys) == 0 {
		return errors.New("jwks contained no usable RSA keys")
	}
	c.mu.Lock()
	c.keys = keys
	c.fetched = time.Now()
	c.mu.Unlock()
	return nil
}

type jwk struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	N   string `json:"n"`
	E   string `json:"e"`
}

func (j jwk) rsaPublicKey() (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(j.N)
	if err != nil {
		return nil, err
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(j.E)
	if err != nil {
		return nil, err
	}
	e := 0
	for _, b := range eBytes {
		e = e<<8 | int(b)
	}
	if e == 0 {
		return nil, errors.New("invalid exponent")
	}
	return &rsa.PublicKey{N: new(big.Int).SetBytes(nBytes), E: e}, nil
}
