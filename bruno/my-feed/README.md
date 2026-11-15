# My feed API

Open this folder as a collection in Bruno and select the `Local` environment. The API defaults to `http://localhost:8080`.

Requests that need an existing resource use `sourceId`, `entryId`, or `fetchJobId` from the environment. Replace the placeholder values with UUIDs returned by the create requests. To fetch another entries page, add a `cursor` query parameter using the previous response's `nextCursor` value.

Start the API from the repository root with `make backend` or `make up`.
