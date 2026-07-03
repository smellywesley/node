#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUTPUT="${OUTPUT:-bin/agentos}"
VERSION="${VERSION:-dev}"
COMMIT="${COMMIT:-}"
BUILT_AT="${BUILT_AT:-}"

if [[ -z "$COMMIT" ]]; then
  if [[ -d "$ROOT/.git" ]] && command -v git >/dev/null 2>&1; then
    COMMIT="$(git -C "$ROOT" rev-parse --short HEAD 2>/dev/null || true)"
  fi
  COMMIT="${COMMIT:-unknown}"
fi

if [[ -z "$BUILT_AT" ]]; then
  BUILT_AT="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
fi

GO_EXE="${GO_EXE:-}"
if [[ -z "$GO_EXE" ]]; then
  if command -v go >/dev/null 2>&1; then
    GO_EXE="$(command -v go)"
  elif [[ -x "$ROOT/work/toolchain/go/bin/go" ]]; then
    GO_EXE="$ROOT/work/toolchain/go/bin/go"
  else
    echo "problem: Go was not found." >&2
    echo "fix: Install Go 1.24+ or place the portable toolchain at work/toolchain/go." >&2
    exit 2
  fi
fi

mkdir -p "$ROOT/$(dirname "$OUTPUT")"
export GOTOOLCHAIN="${GOTOOLCHAIN:-local}"
export GOCACHE="${GOCACHE:-$ROOT/work/gocache}"
export GOMODCACHE="${GOMODCACHE:-$ROOT/work/gomodcache}"

"$GO_EXE" build -buildvcs=false -trimpath \
  -ldflags "-s -w -X main.version=$VERSION -X main.commit=$COMMIT -X main.builtAt=$BUILT_AT" \
  -o "$ROOT/$OUTPUT" ./cmd/agentos

echo "Built $ROOT/$OUTPUT"
