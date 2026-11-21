# API prototype (throwaway)

A separate, in-memory Bookmark module for discussing API layering and the new
`platform/http` success, error and cursor response shapes. It is registered in the main application; restarting it clears its data.

Run the main API using `make backend:backend` from the repository root. The default
address is `localhost:8080`.

For Bruno, open the `bruno/my-feed` collection, select the `Local` environment,
and use the requests in `API prototype`. Copy a created bookmark's `data.id`
to `prototypeBookmarkId` in that environment for get and mark-read requests.
To try the next page, copy `meta.nextCursor` to `prototypeCursor` and enable
the cursor query parameter in `List bookmarks`.

```sh
curl -i -X POST localhost:8080/api/prototype/bookmarks \
  -H 'Content-Type: application/json' \
  -d '{"title":"Example","url":"https://example.com"}'
curl -i 'localhost:8080/api/prototype/bookmarks?pageSize=1'
curl -i localhost:8080/api/prototype/bookmarks/unknown
# Replace ID with the id returned by create:
curl -i localhost:8080/api/prototype/bookmarks/ID
curl -i -X POST localhost:8080/api/prototype/bookmarks/ID/read
```

Create two bookmarks and use `meta.nextCursor` as the `cursor` query parameter
to explore pagination. Invalid input returns 400, an unknown ID returns 404,
and a repeated URL returns 409. Unknown errors return 500 without details.
The cursor implementation is deliberately in-memory and is not a production
pagination design. `nextCursor` is omitted on the final page, as defined by the
current new `httpx.Cursor` helper. These choices are starting points for discussion.
