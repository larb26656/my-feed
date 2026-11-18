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

Go + gin + SQLite. No ORM, no message broker. HTTP via gin; SQL via `database/sql` and modernc SQLite; schema via Goose + `migrations/` SQL files. Run one backend instance with its database on a local persistent filesystem. Business errors: `platform/apperr`. Overview: `docs/patterns/README.md`.

## Architecture

Hexagonal modular monolith (ports & adapters). Dependency direction and cut-outs: `docs/patterns/architecture.md`. Decision record: `docs/adr/0001-architecture.md`.

## Module scaffold

New bounded context / module layout and bootstrap wiring: `docs/patterns/module-scaffold.md`.

## Layer patterns

Reach the matching doc when changing that layer:

- Domain invariants, factories, transitions: `docs/patterns/domain.md`
- Module service or cross-module use case: `docs/patterns/application.md`
- Repository port, `WithTx`, SQLite adapter: `docs/patterns/persistence.md`
- Gin handlers, DTOs, `apperr` → HTTP: `docs/patterns/http.md`
- Shared `platform` kernel: `docs/patterns/platform.md`

## Domain language

Use root `GLOSSARY.md`. Bounded contexts live under `internal/source` and `internal/entry`. Glossary **Feed** is reserved for a future ranked stream — there is no `internal/feed` module.

## Testing pattern

Write tests as business scenarios: name the behavior and expected outcome, keep setup small, and assert observable results rather than implementation details. Prefer readability over coverage percentage. See [Entry and Source test strategy](../docs/testing/backend-tests.md) for examples and scope.

- **One behavior per case:** each test or named subtest should specify one primary behavior and run independently. A sorting case checks order; pagination, filtering, validation, state changes, and persistence belong in separate cases. Multiple assertions are fine when they describe the same outcome. Use table-driven subtests for variations of one rule, with fresh mutable fixtures per case. Keep several steps together only when necessary to prove one behavior, such as reingestion preserving read/saved choices; put prerequisite actions in Arrange and persisted-state verification with the outcome. Do not combine unrelated checks merely to reuse setup; prefer a small fixture helper or a little explicit setup duplication.
- **AAA sections:** use `// Arrange`, `// Act`, and `// Assert` comments to separate fixture/input setup, the operation under test, and its expected outcomes. Aim for one AAA sequence per case; repeated Act/Assert blocks are a signal to split independent behaviors. Add a short business description when useful, such as `// Act: ingest the same article again.` Table-driven subtests may share immutable Arrange data outside the subtest; label local setup when present. Setup checks such as `require.NoError` belong with Arrange, and request helpers may validate transport status during Act. For a trivial getter asserted inline, `// Act & Assert` is sufficient. Label phases, not every statement; avoid consecutive Act comments for one business action.
- **Business workflows:** prefer `httptest` through the production module Gin routes, real services, and real SQLite. For Entry and Source, use `internal/testkit/api.New(t)`; it mounts the module routes without starting fetch workers. Bootstrap tests cover middleware, cross-module fetch hooks, and worker wiring separately.
- **Shared setup:** use `internal/testkit.NewDatabase(t)` for an isolated temporary SQLite file with production connection settings, all production migrations, and automatic cleanup. Use `testkit.Migrate(t, db)` when a persistence test needs to own and reopen its database. Do not copy schemas or build large fake repositories.
- **HTTP assertions:** use `testkit.Request` and `testkit.Decode[T]` for transport boilerplate; keep business actions and assertions visible in each test. Check response fields and persisted state, not only status codes. For errors, check the business code and useful message; internal errors must not leak infrastructure details.
- **Focused tests:** retain domain unit tests for invariants and transitions, and repository tests for SQL constraints, ordering, and persistence across reopen. Use small service stubs for difficult storage-error and conflict-recovery branches. Implement only the relevant port methods and make unexpected calls fail; avoid reimplementing repository behavior or introducing a test DSL.
- **Determinism:** use explicit timestamps and fixed historical state where practical instead of `time.Sleep`. Coordinate concurrent work explicitly. Keep background workers out of module fixtures; a long worker interval does not prevent a startup round. See the [existing bootstrap scheduler flake](../docs/testing/bootstrap-scheduler-flake.md).
- **Behavior to protect:** cover idempotent ingestion, metadata refresh with read/saved preservation, source validation and lifecycle, filters, pagination ties and cursor boundaries, and error responses when changing those paths. Do not change production logic merely to satisfy tests; document discovered bugs separately.

After backend test or code changes, run `go test ./...` and `go test -race ./...` from `backend/`. For coverage comparisons, instrument the same production packages before and after, including code reached through integration tests; exclude testkit helpers and report remaining gaps. Documentation-only edits do not require rerunning the Go suites.
