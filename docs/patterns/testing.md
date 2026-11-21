# Testing pattern

Write tests as business scenarios: name the behavior and expected outcome, keep setup small, and assert observable results rather than implementation details. Prefer readability over coverage percentage.

## Test layers

| What you are changing | Default test kind | How |
| --- | --- | --- |
| Gin handler, route, DTO binding, error envelope | **HTTP integration** | `httptest` through production `RegisterRoutes`, real module service, real SQLite |
| Domain invariant or transition | **Unit** | Domain package only; no database |
| SQL constraints, ordering, reopen persistence | **Persistence** | Repository against SQLite with production migrations |
| Storage failure or conflict-recovery branch | **Narrow port stub** | Real service + stub that embeds the repository port and implements only the methods under test; unexpected calls fail |
| Middleware, fetch hooks, worker wiring | **Bootstrap** | Separate from module HTTP fixtures |

HTTP adapter tests under `adapter/http` are integration tests by default. Handlers take the concrete `*service.Service`. Do not introduce a module-wide service interface so handlers can be unit-tested with mocks. When a failure path needs a fake dependency, stub the **repository port** (or another seam that already has multiple adapters), then construct the real service.

For Entry and Source module routes, use `internal/testkit/api.New(t)`; it mounts production handlers without starting fetch workers.

## One behavior per case

Each test or named subtest specifies one primary behavior and runs independently. A sorting case checks order; pagination, filtering, validation, state changes, and persistence belong in separate cases. Multiple assertions are fine when they describe the same outcome.

Use table-driven subtests for variations of one rule, with fresh mutable fixtures per case. Keep several steps together only when necessary to prove one behavior, such as reingestion preserving read/saved choices; put prerequisite actions in Arrange and persisted-state verification with the outcome. Prefer a small fixture helper or a little explicit setup duplication over combining unrelated checks to reuse setup.

## AAA sections

Use `// Arrange`, `// Act`, and `// Assert` comments to separate fixture/input setup, the operation under test, and its expected outcomes. Aim for one AAA sequence per case; repeated Act/Assert blocks are a signal to split independent behaviors. Add a short business description when useful, such as `// Act: ingest the same article again.`

Table-driven subtests may share immutable Arrange data outside the subtest; label local setup when present. Setup checks such as `require.NoError` belong with Arrange, and request helpers may validate transport status during Act. For a trivial getter asserted inline, `// Act & Assert` is sufficient. Label phases, not every statement; avoid consecutive Act comments for one business action.

## Shared setup

Use `internal/testkit.NewDatabase(t)` for an isolated temporary SQLite file with production connection settings, all production migrations, and automatic cleanup. Use `testkit.Migrate(t, db)` when a persistence test needs to own and reopen its database. Do not copy schemas or build large fake repositories.

## HTTP assertions

Use `testkit.Request` and `testkit.Decode[T]` for transport boilerplate; keep business actions and assertions visible in each test. Check response fields and persisted state, not only status codes. For errors, check the business code and useful message; internal errors must not leak infrastructure details. Include the router's error middleware when checking the error envelope.

## Determinism

Use explicit timestamps and fixed historical state where practical instead of `time.Sleep`. Coordinate concurrent work explicitly. Keep background workers out of module fixtures: `NewWithConfig` starts fetch loops and the scheduler may enqueue on startup even with a long interval, which races module HTTP writes in bootstrap tests.

## After changes

After backend test or code changes, run `go test ./...` and `go test -race ./...` from `backend/`. For coverage comparisons, instrument the same production packages before and after, including code reached through HTTP integration tests; exclude testkit helpers and report remaining gaps. Documentation-only edits do not require rerunning the Go suites.

Do not change production logic merely to satisfy tests; document discovered bugs separately.
