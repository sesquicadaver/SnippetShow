#!/usr/bin/env bash
# scripts/ci-local.sh — локальний CI-прогін у контрольованому середовищі.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

# Pinned release used by FIX-L4 / SS-28 (must match Actions install).
STATICCHECK_VERSION="${STATICCHECK_VERSION:-2026.2.2}"

ensure_staticcheck() {
  export PATH="$(go env GOPATH)/bin:${PATH}"
  if command -v staticcheck >/dev/null 2>&1; then
    return 0
  fi
  echo "==> installing staticcheck@${STATICCHECK_VERSION} (FIX-L4 required)"
  go install "honnef.co/go/tools/cmd/staticcheck@${STATICCHECK_VERSION}"
  export PATH="$(go env GOPATH)/bin:${PATH}"
  if ! command -v staticcheck >/dev/null 2>&1; then
    echo "ci-local: staticcheck is required but not on PATH after install" >&2
    exit 1
  fi
}

run_staticcheck() {
  ensure_staticcheck
  local target="$1"
  shift
  echo "==> staticcheck $* ${target}"
  staticcheck "$@" "$target"
}

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
  echo "==> FTS@3000 latency gate (TestFTSLatencyAt3000, budget ≤50ms)"
  go test ./internal/db -run '^TestFTSLatencyAt3000$' -count=1 -v
  run_staticcheck "./internal/..."
  if command -v node >/dev/null 2>&1; then
    echo "==> frontend unit tests (navigator/shell)"
    node --test frontend/src/ui/*_test.mjs
  fi
  echo "ci-local: OK (internal only)"
  exit 0
fi

if [[ -f frontend/package.json ]] && command -v npm >/dev/null 2>&1; then
  echo "==> frontend build (embed)"
  (cd frontend && npm ci --ignore-scripts >/dev/null && npm run build && touch dist/.gitkeep)
fi

if command -v node >/dev/null 2>&1; then
  echo "==> frontend unit tests (navigator/shell)"
  node --test frontend/src/ui/*_test.mjs
fi

echo "==> go vet ${GO_TAGS[*]:-} ./..."
go vet "${GO_TAGS[@]}" ./...

echo "==> go test ${GO_TAGS[*]:-} ./..."
go test "${GO_TAGS[@]}" ./...

echo "==> FTS@3000 latency gate (TestFTSLatencyAt3000, budget ≤50ms)"
go test ./internal/db -run '^TestFTSLatencyAt3000$' -count=1 -v

if [[ ${#GO_TAGS[@]} -gt 0 ]]; then
  run_staticcheck "./..." "${GO_TAGS[@]}"
else
  run_staticcheck "./..."
fi

echo "ci-local: OK"
