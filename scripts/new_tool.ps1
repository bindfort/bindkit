param(
  [Parameter(Mandatory = $true)]
  [string]$Name
)

$slug = $Name.ToLowerInvariant() -replace '[^a-z0-9_]', '_'
$dir = Join-Path "tools" $slug
New-Item -ItemType Directory -Force -Path $dir | Out-Null

@"
package $slug

import (
	"context"

	"github.com/bindfort/bindkit/internal/mcp"
)

func Register(registry *mcp.Registry) error {
	return registry.Register(mcp.Tool{
		Name:        "$slug.run",
		Description: "TODO: describe $slug.",
		InputSchema: map[string]any{"type": "object"},
	}, run)
}

func run(_ context.Context, _ mcp.CallRequest) (mcp.CallResult, error) {
	return mcp.CallResult{Content: []mcp.Content{{Type: "text", Text: "$slug ok"}}}, nil
}
"@ | Set-Content -Encoding utf8 (Join-Path $dir "$slug.go")

@"
package $slug

import (
	"context"
	"testing"

	"github.com/bindfort/bindkit/internal/mcp"
)

func TestRun(t *testing.T) {
	registry := mcp.NewRegistry()
	if err := Register(registry); err != nil {
		t.Fatal(err)
	}
	result, err := registry.Call(context.Background(), mcp.CallRequest{Name: "$slug.run"})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Content) != 1 {
		t.Fatalf("expected one content item, got %d", len(result.Content))
	}
}
"@ | Set-Content -Encoding utf8 (Join-Path $dir "${slug}_test.go")

Write-Host "created $dir"
