-- +goose Up
CREATE TABLE IF NOT EXISTS sources (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    type        TEXT NOT NULL,
    rss_url     TEXT,
    active      INTEGER NOT NULL DEFAULT 1 CHECK (active IN (0, 1)),
    created_at  INTEGER NOT NULL,
    updated_at  INTEGER NOT NULL,
    CONSTRAINT sources_name_non_empty CHECK (length(trim(name)) > 0),
    CONSTRAINT sources_type_valid CHECK (type IN ('rss', 'manual')),
    CONSTRAINT sources_rss_config CHECK (
        (type = 'rss' AND rss_url IS NOT NULL AND length(trim(rss_url)) > 0)
        OR (type = 'manual' AND rss_url IS NULL)
    )
);

CREATE UNIQUE INDEX IF NOT EXISTS sources_rss_url_unique
    ON sources (rss_url)
    WHERE rss_url IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS sources_manual_singleton
    ON sources (type)
    WHERE type = 'manual';

CREATE TABLE IF NOT EXISTS entries (
    id           TEXT PRIMARY KEY,
    source_id    TEXT NOT NULL REFERENCES sources (id),
    title        TEXT NOT NULL,
    link         TEXT NOT NULL,
    description  TEXT NOT NULL DEFAULT '',
    thumbnail    TEXT,
    published_at INTEGER,
    created_at   INTEGER NOT NULL,
    updated_at   INTEGER NOT NULL,
    read_at      INTEGER,
    saved_at     INTEGER,
    CONSTRAINT entries_source_link_unique UNIQUE (source_id, link),
    CONSTRAINT entries_title_non_empty CHECK (length(trim(title)) > 0),
    CONSTRAINT entries_link_non_empty CHECK (length(trim(link)) > 0)
);

-- +goose Down
DROP TABLE IF EXISTS entries;
DROP TABLE IF EXISTS sources;
