# Domain pattern

## Design principles

1. The aggregate root owns its state and enforces the rules for changing it.
2. Validate required inputs in the factory and initialize the aggregate in a valid state.
3. Expose methods named for business actions, such as `Publish` or `ReserveStock`. Keep fields such as status, stock, and usage private when changing them directly could bypass an invariant.
4. Return typed `apperr` errors for business failures. See the [application error pattern](./errors.md) for shared definitions, module-specific codes, and structured details.

## Example aggregate

```go
// domain/error.go
var ErrProductNotDraft = apperr.Definition{
    Kind: apperr.KindConflict,
    Code: "PRODUCT_NOT_DRAFT",
}

// domain/product.go
type Status string

const (
    StatusDraft     Status = "DRAFT"
    StatusAvailable Status = "AVAILABLE"
)

type Product struct {
    ID        uuid.UUID
    status    Status
    CreatedAt time.Time
    UpdatedAt time.Time
}

func (p *Product) Status() Status {
    return p.status
}

func (p *Product) Publish() error {
    if p.status != StatusDraft {
        return ErrProductNotDraft.New("Only draft products can be published")
    }
    p.status = StatusAvailable
    p.UpdatedAt = time.Now().UTC()
    return nil
}
```

A `NewProduct` factory validates required attributes and sets the ID, timestamps, and initial status. Use a shared invalid-input definition when it describes the failure.

## Domain checklist

- [ ] Express invariants in domain methods so their rules can be read directly from the Go code.
- [ ] Define status constants and check the current state before applying a transition.
- [ ] Represent money in integer minor units (`int64`) for prices and discounts.
- [ ] Store historical snapshots in the aggregate that owns the record, such as the unit price on an order line.
- [ ] Cover successful behavior and rejected transitions in `domain/*_test.go`.

## Layer boundaries

Keep transactions and repository calls in the layers that coordinate persistence. Map HTTP DTOs and database rows in their adapters. Add event publishing when a concrete workflow requires it.
