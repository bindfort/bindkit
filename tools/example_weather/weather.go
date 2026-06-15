package example_weather

import (
	"context"
	"fmt"

	"github.com/bindfort/bindkit/internal/mcp"
)

func Register(registry *mcp.Registry) error {
	return registry.Register(mcp.Tool{
		Name:        "weather.current",
		Description: "Returns a deterministic demo weather summary for a city.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"city": map[string]any{"type": "string", "description": "City name"},
			},
			"required": []string{"city"},
		},
	}, current)
}

func current(_ context.Context, call mcp.CallRequest) (mcp.CallResult, error) {
	city, _ := call.Arguments["city"].(string)
	if city == "" {
		city = "Wroclaw"
	}
	return mcp.CallResult{
		Content: []mcp.Content{{
			Type: "text",
			Text: fmt.Sprintf("%s: 21C, clear enough for a Bindkit demo.", city),
		}},
	}, nil
}
