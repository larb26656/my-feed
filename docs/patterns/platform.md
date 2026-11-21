# Platform (Shared Kernel)

Shared packages available to every module — **small and free of domain-specific business rules**.

```
internal/platform/
  apperr/       # typed application errors
  database/     # SQLite database/sql connection + WithinTransaction
  http/         # Gin binding, response envelopes, error middleware, pagination
```

## `apperr`

See the [application error pattern](./errors.md) for shared and module-specific error definitions.

## `database`

- `NewPool(ctx, databasePath)` — opens a SQLite file, pings it, and configures the busy timeout and foreign keys.
- `WithinTransaction(ctx, fn)` — begins a transaction, runs `fn`, and commits; errors and panics trigger a deferred rollback.
- A cross-module use case calls `WithinTransaction` and passes `tx` to `WithTx`.

## `http` (package `httpx`)

- Independent of any module.
- `ErrorMiddleware` uses `errors.As` to find an `*apperr.AppError` and map it to an HTTP error response.
- Do not log details here without a requirement; keep this package small.

## Bootstrap vs platform

| `platform` | `bootstrap` |
|------------|-------------|
| Reusable primitives | Assembles this application's dependency graph |
| Has no knowledge of Product or Order | Creates repositories, services, and handlers, and mounts routes |

`cmd/api/main.go` only sets up the signal context and calls `bootstrap.Run`.
