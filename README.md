# quick-feed

Monorepo for Quick Feed.

```
quick-feed/
├── frontend/          # React (placeholder — scaffold later)
├── backend/           # Go modular monolith (hexagonal)
└── docs/
    ├── patterns/      # architecture & scaffold
    └── adr/           # architecture decisions
```

## Backend

Quick Feed uses a local SQLite database and runs one backend instance. Docker Compose mounts `backend/data` on the host so the database survives container replacement. Keep this directory on a local filesystem; shared network filesystems and multiple backend instances are not supported.

```bash
make up              # build and run API on :8080; initializes SQLite with Goose
make down            # stop API
make migrate         # apply pending migrations to the local SQLite database
make migrate-status  # show Goose migration status
make migrate-down    # roll back one migration
make backend         # run Go API on :8080
make test-unit       # unit and adapter tests, no Docker
make test            # all Go tests
```

For local development, `make migrate` and `make backend` use `backend/data/quick-feed.db` by default. Set `DATABASE_PATH` to choose another persistent location. The Compose deployment uses `backend/data/quick-feed.db` on the host, mounted as `/data/quick-feed.db` in the container.

### Back up and restore

Install the `sqlite3` command-line tool on the host. The examples below use the Compose database path; for a local development database, substitute the value of `DATABASE_PATH` (by default, `backend/data/quick-feed.db`). Keep backups outside `backend/data`, which is the directory mounted into the container.

Create a backup directory and run SQLite's online backup command while the API is running:

```sh
mkdir -p backups
sqlite3 backend/data/quick-feed.db ".backup 'backups/quick-feed-backup.db'"
```

`.backup` uses SQLite's online backup API to produce a consistent snapshot while the database is open, including committed data that may still be in its write-ahead log (WAL). Do not make a live backup by copying only the `.db` file. The backup file is a standalone database; it does not need WAL or shared-memory sidecar files.

With the `sqlite3` CLI installed, `cd backend && go test ./internal/bootstrap -run TestSQLiteOnlineBackupRestoresAPIState -count=1` exercises this backup command against an open application database, restores the snapshot, and checks the restored API data and SQLite integrity.

To restore that snapshot, first stop the API so no process has the database open. Replace the database file and clear any stale sidecars while the app is stopped, then start the API:

```sh
make down
rm -f backend/data/quick-feed.db-wal backend/data/quick-feed.db-shm
cp backups/quick-feed-backup.db backend/data/quick-feed.db
make up
```

For local development, stop the `make backend` process with Ctrl-C and use the configured `DATABASE_PATH` in place of `backend/data/quick-feed.db`. Keep the database and its sidecars together in the persistent data directory during normal operation. Never remove or replace `-wal` or `-shm` files while the API is running; after it has stopped, removing stale sidecars prevents them from being applied to the restored database.

- Health: `GET http://localhost:8080/healthz`
- Sources: `POST/GET /api/v1/sources`

## Frontend

Folder reserved for a React app. Not scaffolded yet. `make frontend` expects `npm run dev` there.
