-- +goose Up
CREATE TABLE IF NOT EXISTS auth_sessions (
    id          TEXT PRIMARY KEY,
    created_at  INTEGER NOT NULL,
    revoked_at  INTEGER
);

CREATE TABLE IF NOT EXISTS auth_refresh_tokens (
    id              TEXT PRIMARY KEY,
    session_id      TEXT NOT NULL REFERENCES auth_sessions (id),
    token_hash      TEXT NOT NULL UNIQUE,
    expires_at      INTEGER NOT NULL,
    replaced_by_id  TEXT REFERENCES auth_refresh_tokens (id),
    created_at      INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS auth_refresh_tokens_session_id
    ON auth_refresh_tokens (session_id);

-- +goose Down
DROP INDEX IF EXISTS auth_refresh_tokens_session_id;
DROP TABLE IF EXISTS auth_refresh_tokens;
DROP TABLE IF EXISTS auth_sessions;
