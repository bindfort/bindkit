#!/usr/bin/env bash
# Packages the Bindkit source into a clean, customer-ready zip in dist/.
# Works on macOS/Linux and Windows git-bash. Requires `zip`.
set -euo pipefail
cd "$(dirname "$0")/.."

version="$(grep -m1 'const version' cmd/server/main.go | sed -E 's/.*"(.*)".*/\1/')"
out="dist/bindkit-${version}.zip"
mkdir -p dist
rm -f "$out"

echo "Running tests before packaging..."
go test ./... >/dev/null

zip -r "$out" . \
  -x '*/.git/*' '.git/*' 'dist/*' '*.exe' '*.log' \
     'agent.md' 'handoff.json' 'tasks.md' 'architecture-map.html' \
     'landingpage/*' '*/.DS_Store' >/dev/null

echo "packaged -> $out"
unzip -l "$out" | tail -1
