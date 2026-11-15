# Entry and Source test strategy

Business workflows are exercised through `httptest`, the production module Chi
routes, real services, and SQLite with every production Goose migration applied.
`internal/testkit` owns isolated temporary databases and small JSON request/response
helpers. `internal/testkit/api` wires the two modules without starting fetch workers.
It mounts the same module handlers at their production paths; bootstrap integration
tests remain responsible for middleware, fetch hooks, and worker wiring.

Keep business assertions in the scenario. Use domain unit tests for invariants and
focused service stubs for storage failures and another-writer conflict recovery.
The stubs embed their ports and implement only the relevant methods; unexpected
calls fail immediately rather than silently simulating a database.

Repository tests retain checks for database constraints and reopening persistent
files. They share the migration helper rather than maintaining partial schema setup.
Domain timestamp tests start from a fixed historical `UpdatedAt` instead of sleeping
until the clock advances. No production code or dependencies changed.

## Validation and coverage

Validated with Go 1.25.7:

```sh
cd backend
go test ./...
go test -race ./...
go test ./internal/entry/... ./internal/source/... \
  -coverpkg=./internal/entry/...,./internal/source/... -coverprofile=/tmp/modules.cover
go tool cover -func=/tmp/modules.cover
```

Both full ordinary and race suites passed on the final implementation. No races
were reported. Coverage compares unchanged main (`cda595b`) with this refactor,
using the same module-only command on both checkouts. Instrumenting both modules
counts HTTP integration execution in the service and repository packages too.
Testkit code is excluded from the coverage denominator.

| Production package | Before | After |
| --- | ---: | ---: |
| Entry HTTP | 24.2% | 95.0% |
| Entry SQLite | 78.1% | 89.0% |
| Entry domain | 100.0% | 100.0% |
| Entry service | 76.1% | 93.5% |
| Source HTTP | 0.0% | 94.7% |
| Source SQLite | 81.2% | 82.8% |
| Source domain | 92.3% | 97.4% |
| Source service | 79.6% | 94.4% |
| **Entry + Source statements** | **58.9%** | **93.1%** |

The scenarios cover ingestion identity and metadata refresh, preserved read/saved
choices, manual/RSS identity separation, independent reader state changes, source
lifecycle and validation, duplicate URLs with unchanged data after failed edits,
combined filters, cursor criteria binding, timestamp ties, arrivals between pages,
creation-date fallback, default page sizes, and validation/not-found/precondition/
conflict/internal HTTP responses.

Remaining gaps include driver scan/query failures, transaction adapter branches,
several service and handler error propagation branches, and malformed cursor payload
variants. Another-writer recovery is forced deterministically at the service port;
these tests do not prove concurrent ingestion is serializable. Fetch and bootstrap
remain outside this refactor's coverage comparison.

Full-suite coverage runs on unchanged main and this branch exposed existing
bootstrap scheduler flakes. They are recorded separately in
[bootstrap-scheduler-flake.md](bootstrap-scheduler-flake.md). Module coverage above
comes from passing runs and avoids those background workers.
