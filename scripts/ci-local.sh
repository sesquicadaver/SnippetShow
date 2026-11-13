#!/usr/bin/env bash
# scripts/ci-local.sh — локальний CI-прогін у контрольованому середовищі.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

GO_TAGS=()
if pkg-config --exists webkit2gtk-4.1 2>/dev/null; then
  GO_TAGS=(-tags webkit2_41)
elif pkg-config --exists webkit2gtk-4.0 2>/dev/null; then
  GO_TAGS=()
else
  # Без webkit збираємо лише internal (Core) — типовий headless CI без GUI deps.
  echo "==> webkit2gtk not found; limiting to ./internal/..."
  echo "==> go vet ./internal/..."
  go vet ./internal/...
  echo "==> go test ./internal/..."
  go test ./internal/...
  if command -v staticcheck >/dev/null 2>&1; then
    echo "==> staticcheck ./internal/..."
    staticcheck ./internal/...
  else
    echo "==> staticcheck skipped (not installed)"
  fi
  echo "ci-local: OK (internal only)"
  exit 0
fi

if [[ -f frontend/package.json ]] && command -v npm >/dev/null 2>&1; then
  echo "==> frontend build (embed)"
  (cd frontend && npm ci --ignore-scripts >/dev/null && npm run build && touch dist/.gitkeep)
fi

echo "==> go vet ${GO_TAGS[*]:-} ./..."
go vet "${GO_TAGS[@]}" ./...

echo "==> go test ${GO_TAGS[*]:-} ./..."
go test "${GO_TAGS[@]}" ./...

if command -v staticcheck >/dev/null 2>&1; then
  echo "==> staticcheck ./..."
  staticcheck ./...
else
  echo "==> staticcheck skipped (not installed)"
fi

echo "ci-local: OK"
