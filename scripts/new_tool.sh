#!/usr/bin/env sh
set -eu

name="${1:-}"
if [ -z "$name" ]; then
  echo "usage: new_tool.sh my_tool" >&2
  exit 1
fi

slug="$(printf '%s' "$name" | tr '[:upper:]' '[:lower:]' | sed 's/[^a-z0-9_]/_/g')"
dir="tools/$slug"
mkdir -p "$dir"

cat > "$dir/$slug.go" <<EOF
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
EOF

cat > "$dir/${slug}_test.go" <<EOF
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
EOF

echo "created $dir"

