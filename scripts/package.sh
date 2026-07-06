#!/usr/bin/env bash
# Packages the BindKit source into a clean release archive in dist/.
# Works on macOS/Linux and Windows git-bash. Requires `zip`.
set -euo pipefail
cd "$(dirname "$0")/.."

version="$(grep -m1 'const version' cmd/server/main.go | sed -E 's/.*"(.*)".*/\1/')"
out="dist/bindkit-${version}.zip"
mkdir -p dist
rm -f "$out"

echo "Running tests before packaging..."
go test ./... >/dev/null

git ls-files --cached --others --exclude-standard -z |
  while IFS= read -r -d '' file; do
    [ -f "$file" ] && zip -q "$out" "$file"
  done

echo "packaged -> $out"
unzip -l "$out" | tail -1
