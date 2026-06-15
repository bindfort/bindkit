package urlcheck

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bindfort/bindkit/internal/mcp"
)

func TestCheckReportsStatusAndSecurityHeaders(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Strict-Transport-Security", "max-age=63072000")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	tool := New(true) // allow loopback for the test server
	res, err := tool.Handle(context.Background(), mcp.CallRequest{Arguments: map[string]any{"url": srv.URL}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	text := res.Content[0].Text
	for _, want := range []string{"-> 200", "[+] Strict-Transport-Security", "[-] Content-Security-Policy (missing)", "score: 2/5"} {
		if !strings.Contains(text, want) {
			t.Fatalf("report missing %q\n%s", want, text)
		}
	}
}

func TestCheckRejectsInvalidURL(t *testing.T) {
	tool := New(true)
	for _, bad := range []string{"", "not a url", "ftp://example.com", "example.com"} {
		if _, err := tool.Handle(context.Background(), mcp.CallRequest{Arguments: map[string]any{"url": bad}}); err == nil {
			t.Fatalf("expected error for %q", bad)
		}
	}
}

func TestCheckBlocksPrivateAddressesWhenGuarded(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	tool := New(false) // SSRF guard on: loopback test server must be refused
	_, err := tool.Handle(context.Background(), mcp.CallRequest{Arguments: map[string]any{"url": srv.URL}})
	if err == nil || !strings.Contains(err.Error(), "non-public address") {
		t.Fatalf("expected SSRF guard to block loopback, got %v", err)
	}
}
