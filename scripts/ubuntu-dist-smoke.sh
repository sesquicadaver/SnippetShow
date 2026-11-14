#!/usr/bin/env bash
# scripts/ubuntu-dist-smoke.sh — Ubuntu dist smoke (FIX-M9-Ubuntu / P4-16 / D12).
#
# Modes (env):
#   SMOKE_SKIP_CI=1       — не ганяти ./scripts/ci-local.sh
#   SMOKE_SKIP_BUILD=1    — не збирати ELF (очікує build/bin/SnippetShow)
#   SMOKE_SKIP_GUI=1      — не запускати ELF / не міряти cold start+RSS
#   SMOKE_KEEP_APP=1      — лишити SnippetShow запущеним після вимірів
#   SMOKE_ARTIFACT_DIR=…  — каталог логів (default: build/smoke-artifacts)
#
# Exit:
#   0 — core (+ build якщо webkit) OK; GUI виміри optional (SKIP не валить)
#   1 — критичний fail (немає go, ci-local fail, build fail при наявному webkit)
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

ARTIFACT_DIR="${SMOKE_ARTIFACT_DIR:-$ROOT/build/smoke-artifacts}"
mkdir -p "$ARTIFACT_DIR"
STAMP="$(date -u +%Y%m%dT%H%M%SZ)"
LOG="$ARTIFACT_DIR/ubuntu-smoke-${STAMP}.log"
SUMMARY="$ARTIFACT_DIR/ubuntu-smoke-latest.md"

log() { printf '%s\n' "$*" | tee -a "$LOG"; }
section() { log ""; log "## $*"; }

HAVE_WEBKIT=0
if pkg-config --exists webkit2gtk-4.1 2>/dev/null || pkg-config --exists webkit2gtk-4.0 2>/dev/null; then
  HAVE_WEBKIT=1
fi

HAVE_DISPLAY=0
if [[ -n "${DISPLAY:-}" ]] || [[ -n "${WAYLAND_DISPLAY:-}" ]]; then
  HAVE_DISPLAY=1
fi

HOST_PRETTY="unknown"
if [[ -r /etc/os-release ]]; then
  # shellcheck disable=SC1091
  . /etc/os-release
  HOST_PRETTY="${PRETTY_NAME:-$NAME}"
fi

: >"$LOG"
log "# SnippetShow Ubuntu dist smoke"
log "- stamp_utc: $STAMP"
log "- host: $HOST_PRETTY"
log "- kernel: $(uname -r)"
log "- go: $(go version 2>/dev/null || echo missing)"
log "- webkit_pkgconfig: $HAVE_WEBKIT"
if pkg-config --exists webkit2gtk-4.1 2>/dev/null; then
  log "- webkit2gtk-4.1: $(pkg-config --modversion webkit2gtk-4.1)"
fi
log "- display: ${DISPLAY:-}${WAYLAND_DISPLAY:+ wayland=$WAYLAND_DISPLAY}"
log "- have_display: $HAVE_DISPLAY"

CORE_STATUS="pending"
BUILD_STATUS="pending"
GUI_LAUNCH_STATUS="pending"
COLD_MS=""
IDLE_RSS_KIB=""

section "1) Core CI (internal / full with webkit)"
if [[ "${SMOKE_SKIP_CI:-0}" == "1" ]]; then
  CORE_STATUS="skipped"
  log "SKIP ci-local (SMOKE_SKIP_CI=1)"
else
  if ./scripts/ci-local.sh >>"$LOG" 2>&1; then
    CORE_STATUS="pass"
    log "PASS ci-local"
  else
    CORE_STATUS="fail"
    log "FAIL ci-local — див. $LOG"
    exit 1
  fi
fi

section "2) ELF build (webkit required)"
BIN="$ROOT/build/bin/SnippetShow"
if [[ "$HAVE_WEBKIT" -eq 0 ]]; then
  BUILD_STATUS="blocked-no-webkit"
  log "LIMITATION: webkit2gtk not found via pkg-config."
  log "Install (Ubuntu): sudo apt install build-essential pkg-config libgtk-3-dev libwebkit2gtk-4.1-dev"
  log "Then re-run: ./scripts/ubuntu-dist-smoke.sh"
  log "Core-only smoke (ci-local/internal) is the maximum evidence on this host."
elif [[ "${SMOKE_SKIP_BUILD:-0}" == "1" ]]; then
  if [[ -x "$BIN" ]]; then
    BUILD_STATUS="skipped-existing-bin"
    log "SKIP build; using existing $BIN ($(file -b "$BIN" 2>/dev/null || echo bin))"
  else
    BUILD_STATUS="fail-missing-bin"
    log "FAIL: SMOKE_SKIP_BUILD=1 but $BIN missing"
    exit 1
  fi
else
  export PATH="${PATH}:$(go env GOPATH)/bin"
  if ./build/build.sh >>"$LOG" 2>&1; then
    BUILD_STATUS="pass"
    log "PASS build → $BIN"
    log "file: $(file -b "$BIN")"
    log "size: $(du -h "$BIN" | awk '{print $1}')"
  else
    BUILD_STATUS="fail"
    log "FAIL ./build/build.sh"
    exit 1
  fi
fi

section "3) GUI launch + NFR samples (optional)"
if [[ "${SMOKE_SKIP_GUI:-0}" == "1" ]]; then
  GUI_LAUNCH_STATUS="skipped"
  log "SKIP GUI (SMOKE_SKIP_GUI=1)"
elif [[ "$HAVE_WEBKIT" -eq 0 || ! -x "$BIN" ]]; then
  GUI_LAUNCH_STATUS="skipped-no-bin"
  log "SKIP GUI: no ELF"
elif [[ "$HAVE_DISPLAY" -eq 0 ]]; then
  GUI_LAUNCH_STATUS="skipped-no-display"
  log "LIMITATION: no DISPLAY/WAYLAND — ELF not launched here."
  log "On a desktop Ubuntu session run the P4-16 checklist in build/dist-notes-ubuntu.md"
else
  # Kill stale instances from prior smokes (best-effort).
  pkill -x SnippetShow 2>/dev/null || true
  sleep 0.3
  START_NS=$(date +%s%N)
  "$BIN" >>"$ARTIFACT_DIR/SnippetShow-${STAMP}.stdout" 2>>"$ARTIFACT_DIR/SnippetShow-${STAMP}.stderr" &
  APP_PID=$!
  COLD_MS=""
  for _ in $(seq 1 50); do
    sleep 0.1
    if ! kill -0 "$APP_PID" 2>/dev/null; then
      GUI_LAUNCH_STATUS="fail-exited"
      log "FAIL: process exited early; stderr:"
      tail -n 40 "$ARTIFACT_DIR/SnippetShow-${STAMP}.stderr" | tee -a "$LOG" || true
      break
    fi
    RSS=$(ps -o rss= -p "$APP_PID" 2>/dev/null | tr -d ' ' || true)
    if [[ -n "${RSS:-}" && "$RSS" -gt 80000 ]]; then
      END_NS=$(date +%s%N)
      COLD_MS=$(( (END_NS - START_NS) / 1000000 ))
      GUI_LAUNCH_STATUS="pass"
      log "PASS launch pid=$APP_PID cold_start_to_rss80MiB_ms=$COLD_MS"
      break
    fi
  done
  if [[ "$GUI_LAUNCH_STATUS" == "pending" ]]; then
    GUI_LAUNCH_STATUS="fail-timeout"
    log "FAIL: timed out waiting for RSS>80MiB"
  fi
  sleep 2
  if kill -0 "$APP_PID" 2>/dev/null; then
    IDLE_RSS_KIB=$(ps -o rss= -p "$APP_PID" 2>/dev/null | tr -d ' ' || true)
    log "idle_rss_kib=${IDLE_RSS_KIB:-n/a} (~$(( ${IDLE_RSS_KIB:-0} / 1024 )) MiB)"
  fi
  if [[ "${SMOKE_KEEP_APP:-0}" != "1" ]]; then
    kill "$APP_PID" 2>/dev/null || true
    wait "$APP_PID" 2>/dev/null || true
    log "app stopped"
  else
    log "app left running pid=$APP_PID (SMOKE_KEEP_APP=1) — finish P4-16 UI checklist manually"
  fi
fi

section "4) Checklist mapping (P4-16)"
log "Automated here:"
log "  - deps/webkit detect"
log "  - ci-local / vet+test → $CORE_STATUS"
log "  - wails/go ELF build → $BUILD_STATUS"
log "  - ELF launch + cold/RSS sample → $GUI_LAUNCH_STATUS"
log "Manual / GUI interactive (mark in build/dist-notes-ubuntu.md):"
log "  1. Запуск ELF / wails dev"
log "  2. First-run scan (EnsureIndexed)"
log "  3. Пошук відомої утиліти"
log "  4. «Оновити індекс»"
log "  5. Сніпет → Copy → Run"
log "  6. Script Builder export .sh"
log "  7. JSON Export / Import"

section "Summary"
{
  echo "# Ubuntu smoke summary ($STAMP)"
  echo
  echo "| Step | Status |"
  echo "|------|--------|"
  echo "| Core CI | \`$CORE_STATUS\` |"
  echo "| ELF build | \`$BUILD_STATUS\` |"
  echo "| GUI launch | \`$GUI_LAUNCH_STATUS\` |"
  echo "| Cold start → RSS≥80 MiB | \`${COLD_MS:-n/a} ms\` |"
  echo "| Idle RSS | \`${IDLE_RSS_KIB:-n/a} KiB\` |"
  echo
  echo "Host: \`$HOST_PRETTY\`. Log: \`$LOG\`."
  echo
  if [[ "$HAVE_WEBKIT" -eq 0 ]]; then
    echo "**Limitation:** no webkit2gtk — ELF build/GUI skipped; core-only evidence."
  elif [[ "$HAVE_DISPLAY" -eq 0 && "$GUI_LAUNCH_STATUS" == skipped-no-display ]]; then
    echo "**Limitation:** headless host — complete interactive checklist on a desktop session."
  fi
} | tee "$SUMMARY" | tee -a "$LOG"

log "OK: artifacts in $ARTIFACT_DIR"
exit 0
