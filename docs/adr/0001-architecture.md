# ADR 0001 — Modular monolith + hexagonal

## Status

Accepted

## Context

Quick Feed backend needs clear module boundaries without splitting into microservices yet.

## Decision

- Go modular monolith with hexagonal ports & adapters
- No ORM, no message broker at init
- Domain does not import adapters; modules must not import another module's `adapter/sqlite`
- Repository ports expose explicit operations + `WithTx`
- Cross-module use cases (when added) orchestrate via injected ports inside `WithinTransaction`
- Business errors use `platform/apperr`
- Money is `int64` minor units when introduced

Patterns live in `docs/patterns/`.
