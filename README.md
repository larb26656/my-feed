# My feed

<p align="center">
  <img src="frontend/public/my-feed-logo.png" alt="My feed logo" width="180" />
</p>

A self-hosted reader that brings RSS Sources and links you collect into one place. Read Entries, track what is unread, and bookmark favorites in Saved.

![My feed showing the main stream and Source navigation](docs/images/my-feed.png)

*Captured from the running app at `http://localhost:3000/`. Source content varies over time.*

## Features

- **One reading stream** — browse Entries newest first, filter unread Entries, or open a single Source.
- **Read and Saved state** — mark Entries as read and bookmark them in Saved.
- **Save Link** — record a URL in the built-in Saved links Source.
- **Source management** — add RSS Sources, edit their configuration, deactivate them, and reactivate them later. Existing Entries remain available.
- **Fetch Jobs** — fetch automatically on a fixed interval or trigger a Source manually; inspect active work and job history.
- **Local persistence** — keep Entries, Sources, and Fetch Jobs in a SQLite database.

## Quick start

Run commands from the repository root. You need Node.js compatible with Vite 8 (20.19+ or 22.12+), pnpm, and Docker Compose for the backend option below. Local backend development requires Go 1.25.7 or newer and Make.

If upgrading from the former `backend/docker-compose.yml`, stop the old Compose project first (its default project name was `backend`):

```bash
docker compose -p backend down
```

This keeps the existing database in `backend/data`.

Start the application with Docker Compose:

```bash
make backend:up
```

Set `MY_FEED_PASSWORD` and `MY_FEED_JWT_SECRET` in your shell or a root `.env` file first. Compose builds one image containing the frontend and backend, applies Goose migrations, and exposes the app at **http://localhost:8080/**. SQLite data persists in `backend/data/my-feed.db` on the host. For deployment and MCP setup, see [Deployment](#deployment) and [MCP](#mcp).

For frontend development with hot reload, start Vite separately:

```bash
cd frontend
pnpm install
pnpm dev
```

Open **http://localhost:3000/**. The Vite dev server proxies `/api` to `http://localhost:8080`. Check backend health at `http://localhost:8080/healthz`.

To begin, open **Manage Sources** and add an RSS Source. View **Fetch Jobs** to inspect fetching, or use **Save Link** to record a URL yourself. A fresh database does not include the example Sources shown in the screenshot.

### Run the backend without Docker

```bash
make backend:migrate
make backend:backend
```

Both commands default to `backend/data/my-feed.db`. Stop the API with Ctrl-C. Stop the Compose backend with `make backend:down` before running a local backend on the same port.

## Configuration

The backend reads environment variables; defaults are listed in [backend/.env.example](backend/.env.example). For local `make backend:dev`, copy that file to `backend/.env` — the Makefile loads it before starting the API. You can also export overrides in your shell. For Docker, configure the service environment in [docker-compose.yml](docker-compose.yml).

| Variable | Default | Purpose |
| --- | --- | --- |
| `DATABASE_PATH` | `data/my-feed.db` | SQLite path relative to the backend working directory; Compose uses `/data/my-feed.db` |
| `HTTP_ADDR` | `:8080` | HTTP listen address |
| `WEB_ROOT` | _(empty)_ | Frontend build directory; the release image sets `/app/web` |
| `MY_FEED_USERNAME` | _(required)_ | Reader login username |
| `MY_FEED_PASSWORD` | _(required)_ | Reader login password |
| `MY_FEED_JWT_SECRET` | _(required)_ | HS256 secret for access JWTs |
| `MY_FEED_ACCESS_TTL` | `15m` | Access token lifetime |
| `MY_FEED_REFRESH_TTL` | `168h` | Refresh token / Session lifetime |
| `MY_FEED_API_KEY` | _(empty)_ | Optional shared Bearer for `/api/v1` and `/mcp` (scripts); empty = JWT login only |
| `FETCH_SCHEDULE_INTERVAL` | `15m` | Interval between automatic fetching rounds |
| `FETCH_RETRY_BACKOFF` | `1m` | Delay between Fetch Job retries |
| `FETCH_EXECUTOR_POLL` | `2s` | Interval for checking pending Fetch Jobs |
| `VITE_API_PROXY_TARGET` | `http://localhost:8080` | Backend target for the Vite development proxy |
| `VITE_API_BASE_URL` | `/api/v1` | Frontend API base URL; set before building when deploying elsewhere |
| `VITE_MY_FEED_API_KEY` | _(empty)_ | Optional: same secret as `MY_FEED_API_KEY` for non-login clients; the SPA normally uses JWT from `/login` |

## Deployment

The included Compose configuration builds and runs the application on a host with Docker and Docker Compose. Run one instance and keep `backend/data` on a persistent local filesystem; shared network filesystems are not supported.

1. Clone the repository on the host and enter its directory. Create a root `.env` file with your own credentials (do not commit it):

   ```dotenv
   MY_FEED_USERNAME=admin
   MY_FEED_PASSWORD=replace-with-a-strong-password
   MY_FEED_JWT_SECRET=replace-with-a-long-random-secret
   MY_FEED_API_KEY=replace-with-a-separate-random-secret
   ```

   `MY_FEED_API_KEY` is optional, but needed for the long-lived MCP bearer token example below. Generate independent secrets, for example with `openssl rand -hex 32`. The root `.env` is read by Compose; `backend/.env` is for local backend development.

2. Build and start the app, then check its status:

   ```bash
   docker compose up -d --build
   docker compose ps
   docker compose logs --tail=100 api
   curl -f http://localhost:8080/healthz
   ```

   The image runs pending Goose migrations before starting the server. Open `http://<host>:8080/` and sign in with the configured username and password. For access over an untrusted network, put the service behind an HTTPS reverse proxy and forward `/`, `/api`, and `/mcp` to port 8080. Restrict direct access to port 8080 as appropriate.

3. To update, back up the database, update the checkout to the desired revision, and rebuild:

   ```bash
   docker compose up -d --build
   docker compose logs --tail=100 api
   curl -f http://localhost:8080/healthz
   ```

   To stop the service, run `docker compose down`; the bind-mounted data remains in `backend/data`.

### Back up and restore

For a consistent backup while the app is running, use the SQLite CLI's online backup command on the host:

```bash
sqlite3 backend/data/my-feed.db ".backup 'my-feed-backup.db'"
```

Store the resulting file somewhere safe outside `backend/data`. To restore, stop the app, preserve the current data directory, and restore the backup before starting it again:

```bash
docker compose down
mv backend/data backend/data-before-restore
mkdir -p backend/data
cp my-feed-backup.db backend/data/my-feed.db
docker compose up -d
```

Ensure the restored file and directory are writable by the container. Startup applies any pending migrations, so keep a backup before upgrading or restoring.

## MCP

My feed serves a Streamable HTTP MCP endpoint at `/mcp`. Set `MY_FEED_API_KEY` in the deployment's root `.env`, then recreate the service with `docker compose up -d`. The same key grants bearer access to `/api/v1` as well as `/mcp`, so keep it secret. A JWT from `POST /api/v1/auth/login` can also authenticate MCP requests.

For example, add this to your Codex `config.toml`, replacing the URL with your deployment's HTTPS address (or use `http://localhost:8080/mcp` locally):

```toml
[mcp_servers.my-feed]
url = "https://your-host.example/mcp"
bearer_token_env_var = "MY_FEED_API_KEY"
```

Export `MY_FEED_API_KEY` in the environment that starts Codex; the server's `.env` file is not automatically loaded by the client. Restart Codex after changing its configuration. Ask it to save a link to My feed to use the `save_link` tool. The tool requires `title` and `link`, and accepts `description`, `thumbnail`, and `publishedAt` (RFC3339) as optional fields. It records an Entry in the manual **Saved links** Source; it does not bookmark the Entry in **Saved**. An existing link has its metadata refreshed, with `created: false` in the result.

## Development

To enable the repository's Git hooks, install [Lefthook](https://lefthook.dev/install/) and run `make hooks` from the repository root. Install frontend dependencies with `cd frontend && pnpm install` before committing frontend changes. When matching files are staged, the pre-commit hook checks backend formatting, lint, and architecture, and runs Biome on the matching frontend files. These checks read the working tree, including unstaged changes.

| Task | Command |
| --- | --- |
| Start frontend after installing dependencies | `make frontend` |
| Build backend binary | `make backend:build` |
| Run backend unit and adapter tests | `make backend:test-unit` |
| Run all backend tests | `make backend:test` |
| Apply migrations | `make backend:migrate` |
| Inspect migration status | `make backend:migrate-status` |
| Roll back one migration | `make backend:migrate-down` |
| Build frontend | `cd frontend && pnpm build` |
| Run frontend tests | `cd frontend && pnpm test` |
| Check frontend lint and formatting | `cd frontend && pnpm check` |

`make backend:build` writes `backend/bin/server`. The frontend build writes `frontend/dist`. The release image serves it with SPA route fallback from the Go server.

## Architecture

```text
my-feed/
├── frontend/          # React + TypeScript, Vite, TanStack Router/Query, Tailwind CSS
├── backend/           # Go modular monolith with hexagonal ports and adapters
│   ├── cmd/api/       # HTTP API and in-process Fetch Job scheduler/executor
│   ├── internal/      # Source, Entry, Fetch Job modules and platform code
│   ├── migrations/    # Goose SQL migrations
│   └── data/          # Persistent local SQLite database
├── bruno/my-feed/     # API request collection
└── docs/              # Architecture decisions, patterns, and agent guidance
```

The backend uses Gin, SQLite, and Goose. Fetch scheduling and execution run inside the API process; no separate worker or message broker is required. Each deployment supports **one backend instance** with its database on a **local filesystem**.

API resources live under `/api/v1/sources`, `/api/v1/entries`, and `/api/v1/fetch-jobs`. See the [Bruno collection](bruno/my-feed/README.md) for request examples.

## Project documentation

- [Domain glossary](GLOSSARY.md) — Entry, Source, Saved, and Fetch Job terminology.
- [Architecture decisions](docs/adr/) — module boundaries, fetching, and SQLite deployment.
- [Backend patterns](docs/patterns/architecture.md) — implementation conventions.

### Backing up Source Profile Images

Source Profile Images are stored in `UPLOADS_DIR` (default `data/uploads` when running the backend locally). Docker Compose sets it to `/data/uploads` in the same persistent volume as the SQLite database.

Stop the application and any other writers before copying the SQLite database and uploads directory together. Keep both copies as one backup. To restore, stop the application, replace both the database and uploads with the matching copies from that backup, and then restart the application. Restoring only one of the pair can leave images missing or unreferenced.
