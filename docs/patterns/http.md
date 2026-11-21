# HTTP API pattern

For an existing endpoint, use its handler and route tests to identify the status codes, response fields, and pagination behavior clients depend on before editing it.

## Wire the route

Define resource routes in the module's `RegisterRoutes`. Bootstrap constructs the repository, service, and handler and supplies the route prefix.

## Handle the request

1. Bind path, query, and body values in the HTTP adapter. Keep request DTOs and binding tags in `request.go`. Use `httpx.BindJSON` for JSON and Gin's `ShouldBindQuery` for query parameters.
2. Pass `c.Request.Context()` and the mapped input to the service. The service applies operation rules and calls the repository port.
3. Map the result to the response DTO in `response.go`. Return `201 Created` for creation and `200 OK` for reads and commands that return the resulting resource.

## Write the response

| Outcome | Response contract |
| --- | --- |
| Single resource | `httpx.Success(dto)` returns `success: true` and `data`. |
| Cursor list | `httpx.Cursor(items, pageSize, nextCursor)` adds `meta.pageSize` and includes `meta.nextCursor` when another page exists. |
| Page-number list | `httpx.Page(items, page, pageSize, total)` adds `meta.page`, `meta.pageSize`, `meta.total`, and `meta.totalPages`. |

Pass an empty slice to either list helper to return `data: []`.

## Handle errors

See the [application error pattern](./errors.md) for how to define, construct, and wrap typed errors.

- JSON binding failures are reported by `httpx.BindJSON` through `httpx.InvalidRequest`. Return immediately when binding fails.
- Convert query binding failures to a typed invalid-input error and report it through `c.Error(err)`.
- For service errors, call `c.Error(err)` and return before writing the response. The router's `httpx.ErrorMiddleware` maps typed errors centrally, hides internal details on unexpected failures, and logs those failures server-side for investigation.

| Failure | HTTP status | Error code |
| --- | --- | --- |
| Invalid JSON body, query, operation input, or cursor | 400 Bad Request | `VALIDATION` |
| Unknown resource ID | 404 Not Found | `NOT_FOUND` |
| Application conflict | 409 Conflict | Code from the error definition |
| Unexpected failure | 500 Internal Server Error | `INTERNAL_SERVER_ERROR` |

For example, a domain conflict defined with code `CONFLICT` produces this envelope. The middleware includes `error.detail` when the typed error supplies it.

```json
{
  "success": false,
  "error": {
    "code": "CONFLICT",
    "message": "Resource already exists"
  }
}
```

## Paginate list requests

### Criteria naming

Name new criteria fields for the property and comparison they perform: use `NameLike` for a partial match and `NameEqual` for an exact match. Use the corresponding lower-camel-case names in query parameters, such as `nameLike` and `nameEqual`. Apply the same convention to other properties, for example `typeEqual`.

### Page numbers

Initialize `httpx.PageOption` with `httpx.NewPageOption()` before binding the query. The defaults are `page=1` and `pageSize=10`. Apply the same filters to the page query and the total count, and pass the total before pagination to `httpx.Page`. The helper computes `totalPages` from the total and page size.

The Source list accepts `nameLike` (case-insensitive literal substring), `typeEqual` (`rss`, `youtube`, or `manual`), and `activeEqual` (`true` or `false`). Without `activeEqual`, it returns Sources of both statuses. Set `sortBy` to `name`, `type`, `active`, or `createdAt`, and `sortDirection` to `asc` or `desc`; the default is `createdAt` ascending, with ID as the tie-breaker.

### Cursors

Initialize `httpx.CursorPageOption` with `httpx.NewCursorPageOption()` and embed it in the query DTO alongside the endpoint's filters and sort options. The repository filters and orders the result before seeking past the cursor. The client sends `meta.nextCursor` as `cursor` with the same filters and sort options.

For SQLite, derive the cursor from the query's ordering and its tie-breaker so rows that share a sort value can be paged consistently.

## Verify at the route boundary

Run requests through `httptest` and the module's `RegisterRoutes` as HTTP integration tests. Include the router's error middleware when checking the error envelope. Cover query binding and continuation across filtered, sorted pages. See the [testing pattern](./testing.md).
