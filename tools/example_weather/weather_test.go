package example_weather

import (
	"context"
	"strings"
	"testing"

	"github.com/bindfort/bindkit/internal/mcp"
)

func TestWeatherToolUsesProvidedCity(t *testing.T) {
	registry := mcp.NewRegistry()
	if err := Register(registry); err != nil {
		t.Fatal(err)
	}
	result, err := registry.Call(context.Background(), mcp.CallRequest{
		Name:      "weather.current",
		Arguments: map[string]any{"city": "Warsaw"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Content) != 1 || !strings.Contains(result.Content[0].Text, "Warsaw") {
		t.Fatalf("unexpected weather result: %#v", result)
	}
}

func TestWeatherToolDefaultsCity(t *testing.T) {
	result, err := current(context.Background(), mcp.CallRequest{Arguments: map[string]any{}})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.Content[0].Text, "Wroclaw") {
		t.Fatalf("expected default city in result: %#v", result)
	}
}
