## Commands

From repo root via `Makefile` (`make help` for the full list):

- `make up` — build and run the API with a persistent SQLite file at `backend/data/my-feed.db`
- `make down` — stop the API container; data remains on the host
- `make migrate` — Goose `up` on SQLite `migrations/`
- `make migrate-status` — Goose status
- `make migrate-down` — Goose `down` (one step)
- `make backend` — API (:8080)
- `make test-unit` — unit tests, no Docker
- `make test` — all Go tests
- `make build` / `make tidy` — binary / `go mod tidy`

Default `DATABASE_PATH` and `HTTP_ADDR`: `backend/.env.example`. Compose mounts `backend/data` at `/data` in the API container and runs Goose migrations before starting the API.

## Stack

Go + chi + SQLite. No ORM, no message broker. HTTP via chi; SQL via `database/sql` and modernc SQLite; schema via Goose + `migrations/` SQL files. Run one backend instance with its database on a local persistent filesystem. Business errors: `platform/apperr`. Overview: `docs/patterns/README.md`.

## Architecture

Hexagonal modular monolith (ports & adapters). Dependency direction and cut-outs: `docs/patterns/architecture.md`. Decision record: `docs/adr/0001-architecture.md`.

## Module scaffold

New bounded context / module layout and bootstrap wiring: `docs/patterns/module-scaffold.md`.

## Layer patterns

Reach the matching doc when changing that layer:

- Domain invariants, factories, transitions: `docs/patterns/domain.md`
- Module service or cross-module use case: `docs/patterns/application.md`
- Repository port, `WithTx`, SQLite adapter: `docs/patterns/persistence.md`
- Chi handlers, DTOs, `apperr` → HTTP: `docs/patterns/http.md`
- Shared `platform` kernel: `docs/patterns/platform.md`

## Domain language

Use root `GLOSSARY.md`. Bounded contexts live under `internal/source` and `internal/entry`. Glossary **Feed** is reserved for a future ranked stream — there is no `internal/feed` module.
