# ADR 0002 — In-process fetch module + Source Type registry

## Status

Accepted

## Context

Fetchable Sources need a Fetch Job lifecycle (see `GLOSSARY.md`) and a way to obtain origin content per Source Type. A separate `cmd/worker` was the obvious split, but the product is still a single-user modular monolith (ADR 0001) with no message broker.

## Decision

- Own Fetch Jobs in a new `backend/internal/fetch` module; orchestrate `source` / `entry` via injected ports
- Run the fixed-rate scheduler and Fetch Job executor **in-process** with `cmd/api` (started from bootstrap) — no separate worker process for now
- Fixed-rate rounds and manual API triggers both create Fetch Jobs onto the same queue; the executor claims work by moving `pending` → `running` in persistence
- Register obtain adapters by Source Type at bootstrap wiring (`rss` first); adapters return Entry-ready fields only — `fetch` records through the entry port; adapters do not write the database
- Use a single global fixed-rate interval (config; default 15 minutes), not per-Source schedules; fixed retry backoff (config; default 1 minute) on the same Fetch Job

## Considered Options

- **Separate `cmd/worker`**: clearer isolation and scaling later, but extra process/ops cost before we need it — deferred until in-process becomes a problem
- **Message broker for the job queue**: rejected for now; stays aligned with ADR 0001 (no broker at init)
