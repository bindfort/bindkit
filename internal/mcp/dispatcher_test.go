package mcp

import (
	"context"
	"encoding/json"
	"testing"
)

func TestDispatcherInitialize(t *testing.T) {
	dispatcher := NewDispatcher(NewRegistry(), "bindkit-test", "0.0.1")
	response := dispatcher.Handle(context.Background(), &Request{JSONRPC: "2.0", ID: json.RawMessage("1"), Method: "initialize"})
	if response.Error != nil {
		t.Fatalf("expected initialize success, got %v", response.Error)
	}
	result, ok := response.Result.(map[string]any)
	if !ok || result["protocolVersion"] == "" {
		t.Fatalf("unexpected initialize result: %#v", response.Result)
	}
}

func TestDispatcherUnknownMethod(t *testing.T) {
	dispatcher := NewDispatcher(NewRegistry(), "bindkit-test", "0.0.1")
	response := dispatcher.Handle(context.Background(), &Request{JSONRPC: "2.0", ID: json.RawMessage("1"), Method: "unknown"})
	if response.Error == nil || response.Error.Code != -32601 {
		t.Fatalf("expected method-not-found error, got %#v", response.Error)
	}
}

func TestDispatcherInvalidToolCallParams(t *testing.T) {
	dispatcher := NewDispatcher(NewRegistry(), "bindkit-test", "0.0.1")
	response := dispatcher.Handle(context.Background(), &Request{JSONRPC: "2.0", ID: json.RawMessage("1"), Method: "tools/call", Params: json.RawMessage(`{"name":`)})
	if response.Error == nil || response.Error.Code != -32602 {
		t.Fatalf("expected invalid params error, got %#v", response.Error)
	}
}

func TestDispatcherToolHandlerErrorReturnsToolResult(t *testing.T) {
	registry := NewRegistry()
	err := registry.Register(Tool{Name: "broken"}, func(context.Context, CallRequest) (CallResult, error) {
		return CallResult{}, errBoom{}
	})
	if err != nil {
		t.Fatal(err)
	}
	params, _ := json.Marshal(CallRequest{Name: "broken"})
	response := NewDispatcher(registry, "bindkit-test", "0.0.1").Handle(context.Background(), &Request{JSONRPC: "2.0", ID: json.RawMessage("1"), Method: "tools/call", Params: params})
	result, ok := response.Result.(CallResult)
	if response.Error != nil || !ok || !result.IsError {
		t.Fatalf("expected successful MCP response with tool error payload, got response=%#v result=%#v", response, result)
	}
}

type errBoom struct{}

func (errBoom) Error() string { return "boom" }
