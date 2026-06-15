package auth

import (
	"context"
	"errors"
)

var ErrUnauthorized = errors.New("missing or invalid api key")

type Principal struct {
	Key  string
	Plan string
}

type contextKey struct{}

func WithPrincipal(ctx context.Context, principal Principal) context.Context {
	return context.WithValue(ctx, contextKey{}, principal)
}

func PrincipalFromContext(ctx context.Context) (Principal, bool) {
	principal, ok := ctx.Value(contextKey{}).(Principal)
	return principal, ok
}

type Authenticator interface {
	Authenticate(ctx context.Context, apiKey string) (Principal, error)
}

type StaticAuthenticator struct {
	keys map[string]Principal
}

func NewStaticAuthenticator(keys map[string]string) *StaticAuthenticator {
	principals := map[string]Principal{}
	for key, plan := range keys {
		if key == "" {
			continue
		}
		if plan == "" {
			plan = "free"
		}
		principals[key] = Principal{Key: key, Plan: plan}
	}
	return &StaticAuthenticator{keys: principals}
}

func (a *StaticAuthenticator) Authenticate(_ context.Context, apiKey string) (Principal, error) {
	if apiKey == "" {
		return Principal{}, ErrUnauthorized
	}
	principal, ok := a.keys[apiKey]
	if !ok {
		return Principal{}, ErrUnauthorized
	}
	return principal, nil
}
