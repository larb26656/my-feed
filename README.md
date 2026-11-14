# quick-feed

Monorepo for Quick Feed.

```
quick-feed/
├── frontend/          # React (placeholder — scaffold later)
├── backend/           # Go modular monolith (hexagonal)
└── docs/
    ├── patterns/      # architecture & scaffold patterns
    └── adr/           # architecture decisions
```

## Backend

Stack: Go + chi + pgx (no ORM). First bounded context: **feed** (Source + Entry; module folder name only).

```bash
make up         # Postgres on :54330
make migrate    # apply migrations
make backend    # API on :8080
make test-unit  # domain/unit tests, no Docker
```

- Health: `GET http://localhost:8080/healthz`
- Sources: `POST/GET /api/v1/sources`

Default `DATABASE_URL`: `postgres://quickfeed:quickfeed@localhost:54330/quick_feed?sslmode=disable`

## Frontend

Folder reserved for a React app. Not scaffolded yet. `make frontend` expects `npm run dev` there.
