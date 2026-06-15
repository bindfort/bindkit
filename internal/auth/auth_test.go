package auth

import (
	"context"
	"errors"
	"testing"
)

func TestStaticAuthenticator(t *testing.T) {
	authenticator := NewStaticAuthenticator(map[string]string{"abc": "pro", "fallback": ""})
	principal, err := authenticator.Authenticate(context.Background(), "abc")
	if err != nil {
		t.Fatal(err)
	}
	if principal.Key != "abc" || principal.Plan != "pro" {
		t.Fatalf("unexpected principal: %#v", principal)
	}
	principal, err = authenticator.Authenticate(context.Background(), "fallback")
	if err != nil {
		t.Fatal(err)
	}
	if principal.Plan != "free" {
		t.Fatalf("expected default free plan, got %q", principal.Plan)
	}
}

func TestStaticAuthenticatorDeniesUnknown(t *testing.T) {
	authenticator := NewStaticAuthenticator(map[string]string{"abc": "pro"})
	_, err := authenticator.Authenticate(context.Background(), "missing")
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized, got %v", err)
	}
}

func TestPrincipalContextRoundTrip(t *testing.T) {
	want := Principal{Key: "abc", Plan: "pro"}
	got, ok := PrincipalFromContext(WithPrincipal(context.Background(), want))
	if !ok || got != want {
		t.Fatalf("principal did not round trip: %#v %t", got, ok)
	}
}
