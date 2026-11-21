# Application Service Pattern

## Module service

Place module services in `internal/{module}/service`. A service loads an aggregate, invokes its domain method, and persists the resulting state.

```go
type Service struct {
    repo repository.Repository
}

func (s *Service) Publish(ctx context.Context, id uuid.UUID) (*domain.Product, error) {
    p, err := s.repo.GetByID(ctx, id)
    if err != nil {
        return nil, fmt.Errorf("load product %s for publishing: %w", id, err)
    }
    if err := p.Publish(); err != nil {
        return nil, fmt.Errorf("publish product %s: %w", id, err)
    }
    if err := s.repo.Update(ctx, p); err != nil {
        return nil, fmt.Errorf("save published product %s: %w", id, err)
    }
    return p, nil
}
```

Define operation inputs such as `CreateInput` in the service package. The HTTP adapter maps request DTOs to those inputs. Add operation context with `fmt.Errorf` and `%w` when returning errors. Wrapping preserves typed errors for callers that use `errors.As`. See the [application error pattern](./errors.md) for translating errors at a layer that knows their application meaning.

## Module-to-module calls

The providing module declares the interface and exposes the operations and data the caller needs. The caller accepts that interface as its dependency and calls it for reads and writes owned by the provider. For example, Fetch uses a Source contract to read fetchable Sources and an Entry contract to ingest Entries. Keep repository access inside the provider for these calls.

Wire the implementation in bootstrap. Reuse the provider's existing service when its method signatures and behavior satisfy the contract. The provider may add methods or supply an adapter when the contract calls for different behavior or types. See [ADR 0004](../adr/0004-module-api-contracts.md).

## Cross-module use case

Place cross-module orchestration in `internal/{usecase}/service`. When repository operations across modules must succeed together, bind their ports to the same transaction.

```go
err := s.db.WithinTransaction(ctx, func(ctx context.Context, tx database.Transaction) error {
    products := s.products.WithTx(tx)
    coupons := s.coupons.WithTx(tx)
    orders := s.orders.WithTx(tx)

    // Load aggregates and validate the operation through domain methods.
    // Apply the writes through the transaction-bound repositories.
    // Return any error so the transaction helper rolls back all writes.
    return nil
})
```

### Orchestration rules

- Inject repository ports from the participating modules and keep SQL in their adapters.
- Use `WithTx(tx)` for each repository operation that belongs to the transaction. The transaction helper owns commit and rollback.
- Validate business rules through domain methods before writing, and return failures that should abort the operation.

## Idempotency

When the use case accepts an idempotency key, look up an existing result before doing expensive work. Enforce key uniqueness in the database to handle concurrent requests. If a request loses the race on that constraint, roll its transaction back before reading and returning the committed result. See the [architecture decision record](../adr/0001-architecture.md).

## Cancellation and compensation

Model cancellation as its own use case. When it must reverse effects across aggregates, such as restoring stock, releasing a coupon, and cancelling an order, apply those changes in a single transaction so the relevant invariants hold after cancellation.
