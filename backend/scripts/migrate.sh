#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
DATABASE_URL="${DATABASE_URL:-postgres://quickfeed:quickfeed@localhost:54330/quick_feed?sslmode=disable}"
GOOSE_VERSION="${GOOSE_VERSION:-v3.27.3}"
CMD="${1:-up}"
shift || true

exec go run "github.com/pressly/goose/v3/cmd/goose@${GOOSE_VERSION}" \
  -dir "$ROOT/migrations" \
  postgres "$DATABASE_URL" \
  "$CMD" \
  "$@"
