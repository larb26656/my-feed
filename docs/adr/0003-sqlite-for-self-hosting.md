# ADR 0003 — SQLite for self-hosting

## Status

Accepted

## Context

My feed is a personal, single-user application. Running PostgreSQL as a separate service adds operational work to a self-hosted installation. Each deployment runs one backend instance, including the in-process Fetch Job executor (ADR 0002).

## Decision

- Use SQLite as the only supported database, replacing PostgreSQL, and continue using Goose for schema migrations.
- Store the database on a configurable persistent volume or directory on the host's local filesystem, so it survives container replacement. Shared network filesystems are outside the supported deployment scope.
- Preserve existing API and Fetch Job behavior, including safe concurrent access and at most one unfinished Fetch Job per Source.
- Document a backup and restore procedure that produces a consistent backup while the database is in use. Automated backups are outside the current scope.
- Start with a fresh SQLite database; existing PostgreSQL data is experimental and does not require migration.

## Consequences

SQLite reduces installation and operational overhead, at the cost of limiting a deployment to one backend instance and requiring care around concurrent writes, transactions, and backups. Existing PostgreSQL-specific migrations and persistence code must be adapted to SQLite.
