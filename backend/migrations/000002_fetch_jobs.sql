-- +goose Up
CREATE TABLE IF NOT EXISTS fetch_jobs (
    id              TEXT PRIMARY KEY,
    source_id       TEXT NOT NULL REFERENCES sources (id),
    status          TEXT NOT NULL CHECK (status IN ('pending', 'running', 'succeeded', 'failed')),
    attempt         INTEGER NOT NULL DEFAULT 0 CHECK (attempt >= 0),
    max_attempts    INTEGER NOT NULL DEFAULT 3 CHECK (max_attempts > 0),
    entries_created INTEGER NOT NULL DEFAULT 0 CHECK (entries_created >= 0),
    next_attempt_at TEXT,
    created_at      TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at      TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);

CREATE UNIQUE INDEX IF NOT EXISTS fetch_jobs_one_unfinished_per_source
    ON fetch_jobs (source_id)
    WHERE status IN ('pending', 'running');

CREATE INDEX IF NOT EXISTS fetch_jobs_claimable_pending
    ON fetch_jobs (next_attempt_at)
    WHERE status = 'pending';

-- +goose Down
DROP INDEX IF EXISTS fetch_jobs_claimable_pending;
DROP INDEX IF EXISTS fetch_jobs_one_unfinished_per_source;
DROP TABLE IF EXISTS fetch_jobs;
