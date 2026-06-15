package server

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bindfort/bindkit/internal/mcp"
)

func echoHandler() Handler {
	return func(_ context.Context, req *mcp.Request) *mcp.Response {
		return mcp.Success(req.ID, map[string]any{"ok": true})
	}
}

func TestMCPStreamsSSEWhenRequested(t *testing.T) {
	srv := httptest.NewServer(HTTPHandler(echoHandler()))
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/mcp", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`))
	req.Header.Set("Accept", "text/event-stream")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/event-stream") {
		t.Fatalf("expected text/event-stream, got %q", ct)
	}
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "data: ") || !strings.Contains(string(body), `"ok":true`) {
		t.Fatalf("expected an SSE data frame with the result, got: %s", body)
	}
}

func TestMCPReturnsJSONByDefault(t *testing.T) {
	srv := httptest.NewServer(HTTPHandler(echoHandler()))
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/mcp", "application/json", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Fatalf("expected application/json, got %q", ct)
	}
}

func TestWebhookRouteMountedWhenProvided(t *testing.T) {
	called := false
	wh := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})
	srv := httptest.NewServer(HTTPHandler(echoHandler(), wh))
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/stripe/webhook", "application/json", strings.NewReader("{}"))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if !called {
		t.Fatal("expected webhook handler to be mounted at /stripe/webhook")
	}
}
