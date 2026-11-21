## Commands

From repo root via `Makefile` (`make help` for the full list):

- `make backend:up` — build and run the API with a persistent SQLite file at `backend/data/my-feed.db`
- `make backend:down` — stop the API container; data remains on the host
- `make backend:migrate` — Goose `up` on SQLite `migrations/`
- `make backend:migrate-status` — Goose status
- `make backend:migrate-down` — Goose `down` (one step)
- `make backend:backend` — API (:8080)
- `make backend:test-unit` — unit tests, no Docker
- `make backend:test` — all Go tests
- `make backend:check-arch` — verify module and layer dependencies
- `make backend:fmt-check` — check Go formatting
- `make backend:lint` — run Go linters
- `make backend:build` / `make backend:tidy` — binary / `go mod tidy`

Default `DATABASE_PATH` and `HTTP_ADDR`: `backend/.env.example`. Compose mounts `backend/data` at `/data` in the API container and runs Goose migrations before starting the API.

## Stack

Go + gin + SQLite. No ORM, no message broker. HTTP via gin; SQL via `database/sql` and modernc SQLite; schema via Goose + `migrations/` SQL files. Run one backend instance with its database on a local persistent filesystem. Business errors: `platform/apperr`.

## Architecture

Read and follow the [backend architecture pattern](../docs/patterns/architecture.md) when designing, implementing, or reviewing backend changes.

## Module scaffold

New bounded context / module layout and bootstrap wiring: `docs/patterns/module-scaffold.md`.

## Layer patterns

Reach the matching doc when changing that layer:

- Domain invariants, factories, transitions: `docs/patterns/domain.md`
- Module service or cross-module use case: `docs/patterns/application.md`
- Repository port, `WithTx`, SQLite adapter: `docs/patterns/persistence.md`
- Gin handlers, DTOs, `apperr` → HTTP: `docs/patterns/http.md`
- Shared `platform` kernel: `docs/patterns/platform.md`
- Tests, AAA, HTTP integration vs unit seams: `docs/patterns/testing.md`

## Domain language

Use root `GLOSSARY.md`. Bounded contexts live under `internal/modules/source` and `internal/modules/entry`. Glossary **Feed** is reserved for a future ranked stream — there is no `internal/feed` module.
