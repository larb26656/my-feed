# quick-feed

Monorepo for Quick Feed.

```
quick-feed/
├── frontend/   # React (placeholder — scaffold later)
└── backend/    # Go API
```

## Commands

```bash
make backend    # Go API on :8080
make frontend   # React dev server (after scaffold)
make build      # build Go binary → backend/bin/server
make test       # Go tests
make tidy       # go mod tidy
```

Health check: `GET http://localhost:8080/health`

## Frontend

Folder reserved for a React app. Not scaffolded yet. `make frontend` expects `npm run dev` there.
