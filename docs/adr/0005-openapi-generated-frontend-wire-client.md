# ADR 0005 — OpenAPI-generated frontend wire client

## Status

Accepted

## Context

The backend already treats the committed OpenAPI document (`backend/internal/platform/apidocs/swagger.json`, produced by `make openapi` and checked by `make check-openapi`) as the HTTP API description. The frontend still used a hand-written `fetch` wrapper and hand-written types, which had already drifted from the real envelopes in `docs/patterns/http.md` (success/`data`/`meta`, error `{ code, message, detail? }`, pagination). We need a FE↔BE wire seam that stays typed without giving up control of TanStack Query.

## Decision

- Treat the committed OpenAPI document as the sole HTTP wire contract the frontend may depend on.
- Generate TypeScript types (and path/operation typings) from that document with `openapi-typescript`; commit the generated output under the frontend.
- Keep a small hand-rolled thin client on top of those types: base URL, JSON, unwrap successful `{ data, meta? }`, and throw a typed error from the fail envelope. Do not adopt `openapi-fetch` or Orval.
- Leave React Query hooks hand-written; they call the thin client and own cache/UX concerns.
- Sync via a repo script (OpenAPI → generate FE artifacts) and CI that fails when generated files drift from `swagger.json`.
- Land in two phases: (1) fix OpenAPI annotation fidelity so the spec matches real handlers (especially Sources list), then (2) introduce codegen and migrate all production hooks in one pass.
- Defer stronger CI that validates live responses against OpenAPI schemas (beyond existing path+method coverage); revisit if annotation drift recurs.

## Considered Options

- **Keep hand-written FE types/`fetch`**: rejected — drift already happened and will keep happening as the Gin API evolves.
- **Orval (or other full SDK generators), including React Query output**: rejected — the API surface is small and we want Query control in hand-written hooks.
- **`openapi-fetch` as the thin layer**: rejected — envelope unwrap and typed errors are project-specific enough that one explicit wrapper is clearer.
- **Generate-on-the-fly only (no committed FE artifacts)**: rejected — mirrors how we already commit `swagger.json`; review and local builds should see the contract without requiring codegen everywhere.
