#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
DATABASE_URL="${DATABASE_URL:-postgres://quickfeed:quickfeed@localhost:54330/quick_feed?sslmode=disable}"

for f in "$ROOT"/migrations/*.up.sql; do
  psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f "$f"
done
echo "migrations applied"
