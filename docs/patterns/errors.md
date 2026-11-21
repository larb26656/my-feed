# Application error pattern

Use [`platform/apperr`](../../backend/internal/platform/apperr/apperr.go) for application errors that callers need to distinguish. Each `apperr.Definition` pairs a shared classification (`Kind`) with a stable identifier (`Code`).

## Choose an error definition

Check [`generic_error.go`](../../backend/internal/platform/apperr/generic_error.go) first. Reuse a shared definition when its kind and code express the failure. Add a module-specific definition in `domain/error.go` when callers need to distinguish a domain-specific condition.

## Error code naming

Define every new error code in `UPPER_SNAKE_CASE`, for example `CAPACITY_EXCEEDED`. Keep existing codes stable for clients that depend on them.

## Create and wrap errors

- Call `Definition.New(message)` when the failure has no underlying error.
- Call `Definition.Wrap(cause, message)` to preserve the underlying error for Go callers through `Unwrap`. Use a client-safe message for errors that reach the API.
- Use `WithDetail(detail)` for structured, client-safe details, such as validation fields.
- Translate adapter-specific failures at the layer that knows their application meaning. For example, a service can translate an invalid repository cursor into a shared invalid-input error.

## Custom error with structured details

Define a domain-specific conflict in `domain/error.go`.

```go
var ErrCapacityExceeded = apperr.Definition{
    Kind: apperr.KindConflict,
    Code: "CAPACITY_EXCEEDED",
}
```

Create an error for each occurrence and attach structured details the client can use.

```go
return domain.ErrCapacityExceeded.New("Capacity exceeded").WithDetail(struct {
    Limit int `json:"limit"`
}{
    Limit: limit,
})
```

The API includes the limit as `error.detail.limit`.

## HTTP mapping

The HTTP adapter maps `Kind` to a status and returns `Code` in the response. See [HTTP error handling](./http.md#handle-errors) for the middleware, response shape, and status examples.
