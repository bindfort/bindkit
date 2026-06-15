package mcp

import (
	"context"
	"errors"
	"testing"
)

func TestRegistryRejectsInvalidRegistration(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Register(Tool{}, func(context.Context, CallRequest) (CallResult, error) {
		return CallResult{}, nil
	}); err == nil {
		t.Fatal("expected missing-name error")
	}
	if err := registry.Register(Tool{Name: "x"}, nil); err == nil {
		t.Fatal("expected missing-handler error")
	}
}

func TestRegistryRejectsDuplicateTool(t *testing.T) {
	registry := NewRegistry()
	handler := func(context.Context, CallRequest) (CallResult, error) {
		return CallResult{}, nil
	}
	if err := registry.Register(Tool{Name: "x"}, handler); err != nil {
		t.Fatal(err)
	}
	if err := registry.Register(Tool{Name: "x"}, handler); err == nil {
		t.Fatal("expected duplicate registration error")
	}
}

func TestRegistryListIsSorted(t *testing.T) {
	registry := NewRegistry()
	handler := func(context.Context, CallRequest) (CallResult, error) {
		return CallResult{}, nil
	}
	for _, name := range []string{"zeta", "alpha", "middle"} {
		if err := registry.Register(Tool{Name: name}, handler); err != nil {
			t.Fatal(err)
		}
	}
	tools := registry.List()
	if got := []string{tools[0].Name, tools[1].Name, tools[2].Name}; got[0] != "alpha" || got[1] != "middle" || got[2] != "zeta" {
		t.Fatalf("tools not sorted: %#v", got)
	}
}

func TestRegistryCallUnknownTool(t *testing.T) {
	registry := NewRegistry()
	_, err := registry.Call(context.Background(), CallRequest{Name: "missing"})
	if !errors.Is(err, ErrToolNotFound) {
		t.Fatalf("expected ErrToolNotFound, got %v", err)
	}
}
