-- +goose Up
CREATE TABLE IF NOT EXISTS fetch_jobs (
    id              UUID PRIMARY KEY,
    source_id       UUID NOT NULL REFERENCES sources (id),
    status          TEXT NOT NULL,
    attempt         INT NOT NULL DEFAULT 0,
    max_attempts    INT NOT NULL DEFAULT 3,
    entries_created INT NOT NULL DEFAULT 0,
    next_attempt_at TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fetch_jobs_status_valid CHECK (
        status IN ('pending', 'running', 'succeeded', 'failed')
    ),
    CONSTRAINT fetch_jobs_attempt_non_negative CHECK (attempt >= 0),
    CONSTRAINT fetch_jobs_max_attempts_positive CHECK (max_attempts > 0),
    CONSTRAINT fetch_jobs_entries_created_non_negative CHECK (entries_created >= 0)
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
