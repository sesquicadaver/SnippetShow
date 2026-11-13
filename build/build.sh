#!/usr/bin/env bash
# build/build.sh — Linux-only обгортка wails build для SnippetShow.
# Windows/macOS scaffold (build/windows, build/darwin) навмисно відсутні: продукт лише для Linux.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

export PATH="${PATH}:$(go env GOPATH)/bin"

TAGS=()
if pkg-config --exists webkit2gtk-4.1 2>/dev/null; then
  TAGS+=(-tags webkit2_41)
  echo "Using webkit2gtk-4.1 (build tag webkit2_41)"
elif pkg-config --exists webkit2gtk-4.0 2>/dev/null; then
  echo "Using webkit2gtk-4.0"
else
  echo "error: webkit2gtk-4.0 or webkit2gtk-4.1 not found via pkg-config" >&2
  exit 1
fi

if [[ ! -d frontend/node_modules ]]; then
  (cd frontend && npm install)
fi

if command -v wails >/dev/null 2>&1; then
  wails build "${TAGS[@]}" "$@"
else
  echo "wails CLI not found; falling back to go build (GUI tags may still apply)" >&2
  mkdir -p build/bin
  (cd frontend && npm run build)
  go build "${TAGS[@]}" -o build/bin/SnippetShow .
fi

echo "OK: build/bin/SnippetShow (або шлях з wails.json)"
