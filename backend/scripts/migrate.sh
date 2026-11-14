#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
DATABASE_URL="${DATABASE_URL:-postgres://quickfeed:quickfeed@localhost:54330/quick_feed?sslmode=disable}"

psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f "$ROOT/migrations/000001_init.up.sql"
echo "migrations applied"
