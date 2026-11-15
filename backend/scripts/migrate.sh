#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
DATABASE_PATH="${DATABASE_PATH:-data/my-feed.db}"
GOOSE_VERSION="${GOOSE_VERSION:-v3.27.3}"
CMD="${1:-up}"
shift || true

mkdir -p "$(dirname "$DATABASE_PATH")"

exec go run "github.com/pressly/goose/v3/cmd/goose@${GOOSE_VERSION}" \
  -dir "$ROOT/migrations" \
  sqlite3 "$DATABASE_PATH" \
  "$CMD" \
  "$@"
