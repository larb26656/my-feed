# Module scaffold

Use this layout when adding a module under `backend/internal/modules/`. Add packages and files as their responsibilities appear.

## Directory layout

```text
internal/modules/{module}/
  api/                   # Provider-owned contracts for other modules
  domain/
    {entity}.go          # Aggregate state, invariants, and transitions
    {entity}_test.go     # Domain tests with no database dependency
    error.go             # Module-specific application error definitions
  repository/
    {entity}.go          # Repository port and query types
  service/
    {entity}.go          # Application operations and service inputs
  adapter/
    http/
      handler.go         # Gin routes and request handling
      request.go         # Request DTOs, query options, and binding tags
      response.go        # Response DTOs and domain-to-response mapping
    sqlite/
      {entity}.go        # SQL implementation of the repository port
```

## Package responsibilities

| Package | Responsibility |
| --- | --- |
| `domain` | Own aggregate invariants and transitions. Define custom errors in `error.go` after checking the shared definitions. See [Application error pattern](./errors.md). |
| `repository` | Declare the persistence operations and query types that the module uses. See [Persistence pattern](./persistence.md). |
| `service` | Orchestrate domain operations through repository ports. Declare typed operation inputs here. See [Application service pattern](./application.md). |
| `adapter/http` | Register routes in `handler.go`, bind transport inputs defined in `request.go`, and serialize DTOs defined in `response.go`. See [HTTP API pattern](./http.md). |
| `adapter/sqlite` | Implement the repository port, map rows to domain values, and translate recognized storage errors. |

Use domain terms for entity types and filenames. Name the port `repository.Repository` for a module with a single repository. Use unexported HTTP DTOs such as `createProductRequest` and `productResponse`.

## Bootstrap wiring

Wire modules in `internal/bootstrap/app.go`.

1. Build the SQLite repository with the application's database handle.
2. Pass the repository port to the module service.
3. For module-to-module calls, wire the implementation of the provider-owned API interface into the calling module. Construct cross-module use cases with the ports and transaction helper they need. See the [application service pattern](./application.md#module-to-module-calls).
4. Construct the handler with the service and call `RegisterRoutes` on the router group for that API.

Keep a cross-module use case in `internal/modules/{usecase}/service/` when it coordinates multiple modules. Depend on other modules only through `api` contracts. Use the transaction helper to bind participating repository ports to the same transaction when their writes must succeed together.

## Migrations

Add Goose SQL migrations under `migrations/` when the module introduces or changes persisted data. Include the applicable `-- +goose Up` and `-- +goose Down` sections. Use primary keys and unique constraints to enforce identity and uniqueness, and use `CHECK` constraints for invariants such as nonnegative stock. Apply and inspect migrations with `make migrate` and `make migrate-status`.
