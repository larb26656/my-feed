# Persistence pattern

## Repository port

Define the module's persistence contract in `repository/`. Give each operation a method that expresses its intent. Use names such as `ReserveStock` or `Redeem` for operations that must enforce a rule under concurrent writes.

```go
type Repository interface {
    Create(ctx context.Context, p *domain.Product) error
    GetByID(ctx context.Context, id uuid.UUID) (*domain.Product, error)
    Update(ctx context.Context, p *domain.Product) error
    ReserveStock(ctx context.Context, id uuid.UUID, qty int) error
    RestoreStock(ctx context.Context, id uuid.UUID, qty int) error
    WithTx(tx database.Transaction) Repository
}
```

## List contracts and file organization

Keep repository method signatures in `repository.go`. Place input types such as list criteria, sort options, and cursor positions in `request.go`. Place dedicated result types in `response.go` when needed; methods can return domain types directly without a separate response file.

Criteria describe filters only. Define sort options as a separate type and pass criteria and sort separately to list methods. Keep cursor and pagination inputs separate as well. The service applies defaults and validates sort values before calling the repository.

## SQLite adapter

Place the implementation in `internal/modules/{module}/adapter/sqlite/`.

1. Use a `Querier` interface for the execution and query methods shared by the database handle and a transaction.
2. Construct the adapter with `NewRepository(pool)` for ordinary requests.
3. Have `WithTx(tx)` return a new repository instance bound to the supplied transaction. Return the port type so callers can use the same contract inside and outside the transaction.

Keep SQL in the adapter, use `database/sql`, and route service persistence calls through the port. Use the platform transaction helper when coordinating multiple operations in one transaction. Keep SQL and error translation with the owning module.

## Error translation

Translate recognized storage failures into typed application errors at the adapter boundary. Check shared definitions first, then use a domain definition when callers need to distinguish a specific condition. Use `Definition.Wrap` to preserve the original error in the chain and supply a client-safe message. See the [application error pattern](./errors.md).

For example, a lookup that requires a row can translate `sql.ErrNoRows` into `apperr.ResourceNotFound`.

```go
if errors.Is(err, sql.ErrNoRows) {
    return nil, apperr.ResourceNotFound.Wrap(err, "Product not found")
}
if err != nil {
    return nil, fmt.Errorf("get product by ID: %w", err)
}
```

For a recognized uniqueness violation, check that it matches the constraint for the operation before returning the appropriate conflict definition with `Wrap`. For failures without an application mapping, return `fmt.Errorf("operation: %w", err)` to add context and preserve the cause. When a successful statement reports zero affected rows and the operation requires a change, use `Definition.New` for the application failure.

## Enforce invariants in the domain and database

| Layer | Responsibility |
| --- | --- |
| Domain | Validate quantities and allowed state transitions before writing. |
| Database | Enforce invariants under concurrent writes with `UNIQUE`, `CHECK`, and conditional updates. |

For example, reserve stock only when the product is available and has sufficient stock.

```sql
UPDATE products
SET stock = stock - $2
WHERE id = $1 AND stock >= $2 AND status = 'AVAILABLE'
```

Check `RowsAffected()` after the update. If it returns zero, return the application error required by the repository contract so the enclosing transaction can roll back. The predicate combines multiple conditions, so identifying a specific reason may require further information.
