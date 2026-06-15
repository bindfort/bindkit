package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bindfort/bindkit/internal/auth"
	"github.com/bindfort/bindkit/internal/billing"
	"github.com/bindfort/bindkit/internal/mcp"
	"github.com/bindfort/bindkit/internal/metering"
	"github.com/bindfort/bindkit/internal/ratelimit"
)

func testHandler(t *testing.T, opts Options) (Handler, metering.Store) {
	t.Helper()
	registry := mcp.NewRegistry()
	err := registry.Register(mcp.Tool{Name: "echo", Description: "echo", InputSchema: map[string]any{"type": "object"}}, func(_ context.Context, call mcp.CallRequest) (mcp.CallResult, error) {
		value, _ := call.Arguments["text"].(string)
		return mcp.CallResult{Content: []mcp.Content{{Type: "text", Text: value}}}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if opts.Metering == nil {
		opts.Metering = metering.NewMemoryStore()
	}
	if opts.Limiter == nil {
		opts.Limiter = ratelimit.New(100)
	}
	if opts.Authenticator == nil {
		opts.Authenticator = auth.NewStaticAuthenticator(map[string]string{"test-key": "free"})
	}
	if opts.Quota == nil {
		opts.Quota = billing.NewQuotaChecker(opts.Metering, map[string]int{"free": 10})
	}
	return BuildHandler(mcp.NewDispatcher(registry, "test", "0"), opts), opts.Metering
}

func callReq(id string) *mcp.Request {
	params, _ := json.Marshal(map[string]any{
		"name":      "echo",
		"arguments": map[string]any{"text": "ok"},
	})
	return &mcp.Request{JSONRPC: "2.0", ID: json.RawMessage(id), Method: "tools/call", Params: params}
}

func callReqWithAPIKey(id, key string) *mcp.Request {
	params, _ := json.Marshal(map[string]any{
		"name":   "echo",
		"apiKey": key,
		"arguments": map[string]any{
			"text": "ok",
		},
	})
	return &mcp.Request{JSONRPC: "2.0", ID: json.RawMessage(id), Method: "tools/call", Params: params}
}

func TestDiscoveryWorksWithoutGates(t *testing.T) {
	handler, _ := testHandler(t, Options{AuthEnabled: true, BillingEnabled: true})
	response := handler(context.Background(), &mcp.Request{JSONRPC: "2.0", ID: json.RawMessage("1"), Method: "tools/list"})
	if response.Error != nil {
		t.Fatalf("expected discovery success, got %v", response.Error)
	}
}

func TestToolCallSucceedsWithGatesDisabled(t *testing.T) {
	handler, store := testHandler(t, Options{})
	response := handler(context.Background(), callReq("1"))
	if response.Error != nil {
		t.Fatalf("expected success, got %v", response.Error)
	}
	count, _ := store.Count(context.Background(), "anonymous")
	if count != 1 {
		t.Fatalf("expected metering count 1, got %d", count)
	}
}

func TestMissingKeyDeniedWhenAuthEnabled(t *testing.T) {
	handler, _ := testHandler(t, Options{AuthEnabled: true})
	response := handler(context.Background(), callReq("1"))
	if response.Error == nil || response.Error.Code != -32001 {
		t.Fatalf("expected auth denial, got %#v", response.Error)
	}
}

func TestAuthCanReadAPIKeyFromParams(t *testing.T) {
	handler, store := testHandler(t, Options{AuthEnabled: true})
	response := handler(context.Background(), callReqWithAPIKey("1", "test-key"))
	if response.Error != nil {
		t.Fatalf("expected success, got %v", response.Error)
	}
	count, _ := store.Count(context.Background(), "test-key")
	if count != 1 {
		t.Fatalf("expected authenticated metering count 1, got %d", count)
	}
}

func TestDeniedCallsDoNotIncrementMetering(t *testing.T) {
	handler, store := testHandler(t, Options{AuthEnabled: true})
	response := handler(context.Background(), callReq("1"))
	if response.Error == nil {
		t.Fatal("expected denied call")
	}
	count, _ := store.Count(context.Background(), "anonymous")
	if count != 0 {
		t.Fatalf("denied call should not increment anonymous count, got %d", count)
	}
}

func TestRateLimitDeniesBeforeDispatch(t *testing.T) {
	handler, _ := testHandler(t, Options{Limiter: ratelimit.New(1)})
	if response := handler(context.Background(), callReq("1")); response.Error != nil {
		t.Fatalf("first call should pass: %#v", response.Error)
	}
	response := handler(context.Background(), callReq("2"))
	if response.Error == nil || response.Error.Code != -32002 {
		t.Fatalf("expected rate denial, got %#v", response.Error)
	}
}

func TestQuotaDeniesBeforeDispatch(t *testing.T) {
	store := metering.NewMemoryStore()
	handler, _ := testHandler(t, Options{
		AuthEnabled:    true,
		BillingEnabled: true,
		Metering:       store,
		Quota:          billing.NewQuotaChecker(store, map[string]int{"free": 1}),
	})
	ctx := WithAPIKey(context.Background(), "test-key")
	if response := handler(ctx, callReq("1")); response.Error != nil {
		t.Fatalf("first call should pass: %#v", response.Error)
	}
	response := handler(ctx, callReq("2"))
	if response.Error == nil || response.Error.Code != -32003 {
		t.Fatalf("expected quota denial, got %#v", response.Error)
	}
}

func TestStdioTransport(t *testing.T) {
	handler, _ := testHandler(t, Options{})
	in := bytes.NewBufferString(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}` + "\n")
	var out bytes.Buffer
	if err := ServeStdio(context.Background(), handler, in, &out); err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(out.Bytes(), []byte(`"tools"`)) {
		t.Fatalf("expected tools list response, got %s", out.String())
	}
}

func TestStdioTransportReturnsParseErrorAndContinues(t *testing.T) {
	handler, _ := testHandler(t, Options{})
	in := bytes.NewBufferString("{bad json}\n" + `{"jsonrpc":"2.0","id":1,"method":"tools/list"}` + "\n")
	var out bytes.Buffer
	if err := ServeStdio(context.Background(), handler, in, &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), `"code":-32700`) || !strings.Contains(out.String(), `"tools"`) {
		t.Fatalf("expected parse error and later tools response, got %s", out.String())
	}
}

func TestHTTPHandlerHealthz(t *testing.T) {
	handler, _ := testHandler(t, Options{})
	server := httptest.NewServer(HTTPHandler(handler))
	defer server.Close()

	response, err := http.Get(server.URL + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", response.StatusCode)
	}
}

func TestHTTPHandlerAuthenticatedToolCall(t *testing.T) {
	handler, _ := testHandler(t, Options{AuthEnabled: true})
	server := httptest.NewServer(HTTPHandler(handler))
	defer server.Close()

	body := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"echo","arguments":{"text":"hello"}}}`
	req, err := http.NewRequest(http.MethodPost, server.URL+"/mcp", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer test-key")
	req.Header.Set("Content-Type", "application/json")
	response, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", response.StatusCode)
	}
	var decoded mcp.Response
	if err := json.NewDecoder(response.Body).Decode(&decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Error != nil {
		t.Fatalf("expected success, got %v", decoded.Error)
	}
}

func TestHTTPHandlerRejectsWrongMethod(t *testing.T) {
	handler, _ := testHandler(t, Options{})
	server := httptest.NewServer(HTTPHandler(handler))
	defer server.Close()

	response, err := http.Get(server.URL + "/mcp")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", response.StatusCode)
	}
}

func TestHTTPHandlerReturnsJSONRPCParseError(t *testing.T) {
	handler, _ := testHandler(t, Options{})
	server := httptest.NewServer(HTTPHandler(handler))
	defer server.Close()

	response, err := http.Post(server.URL+"/mcp", "application/json", strings.NewReader("{bad json}"))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	var decoded mcp.Response
	if err := json.NewDecoder(response.Body).Decode(&decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Error == nil || decoded.Error.Code != -32700 {
		t.Fatalf("expected parse error, got %#v", decoded.Error)
	}
}
