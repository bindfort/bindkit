package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const testKID = "test-key-1"

func newOAuthFixture(t *testing.T) (*OAuthAuthenticator, *rsa.PrivateKey, func()) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	jwks := map[string]any{"keys": []map[string]string{{
		"kty": "RSA",
		"kid": testKID,
		"n":   base64.RawURLEncoding.EncodeToString(key.N.Bytes()),
		"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes()),
	}}}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(jwks)
	}))
	authn, err := NewOAuthAuthenticator(OAuthConfig{
		Issuer:   "https://issuer.test",
		Audience: "bindkit",
		JWKSURL:  srv.URL,
	})
	if err != nil {
		t.Fatal(err)
	}
	return authn, key, srv.Close
}

func sign(t *testing.T, key *rsa.PrivateKey, claims jwt.MapClaims) string {
	t.Helper()
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tok.Header["kid"] = testKID
	s, err := tok.SignedString(key)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestOAuthAcceptsValidToken(t *testing.T) {
	authn, key, done := newOAuthFixture(t)
	defer done()

	token := sign(t, key, jwt.MapClaims{
		"iss":  "https://issuer.test",
		"aud":  "bindkit",
		"sub":  "user_42",
		"plan": "pro",
		"exp":  time.Now().Add(time.Hour).Unix(),
	})
	p, err := authn.Authenticate(context.Background(), token)
	if err != nil {
		t.Fatalf("expected valid token: %v", err)
	}
	if p.Key != "user_42" || p.Plan != "pro" {
		t.Fatalf("unexpected principal: %+v", p)
	}
}

func TestOAuthRejectsExpiredWrongAudienceAndIssuer(t *testing.T) {
	authn, key, done := newOAuthFixture(t)
	defer done()

	cases := map[string]jwt.MapClaims{
		"expired":      {"iss": "https://issuer.test", "aud": "bindkit", "sub": "u", "exp": time.Now().Add(-time.Hour).Unix()},
		"wrong aud":    {"iss": "https://issuer.test", "aud": "someone-else", "sub": "u", "exp": time.Now().Add(time.Hour).Unix()},
		"wrong issuer": {"iss": "https://evil.test", "aud": "bindkit", "sub": "u", "exp": time.Now().Add(time.Hour).Unix()},
		"no sub":       {"iss": "https://issuer.test", "aud": "bindkit", "exp": time.Now().Add(time.Hour).Unix()},
	}
	for name, claims := range cases {
		token := sign(t, key, claims)
		if _, err := authn.Authenticate(context.Background(), token); err == nil {
			t.Fatalf("%s: expected rejection", name)
		}
	}
}

func TestOAuthRejectsTokenSignedByUnknownKey(t *testing.T) {
	authn, _, done := newOAuthFixture(t)
	defer done()

	attacker, _ := rsa.GenerateKey(rand.Reader, 2048)
	token := sign(t, attacker, jwt.MapClaims{
		"iss": "https://issuer.test", "aud": "bindkit", "sub": "u",
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	if _, err := authn.Authenticate(context.Background(), token); err == nil {
		t.Fatal("expected rejection of token signed by an untrusted key")
	}
}
