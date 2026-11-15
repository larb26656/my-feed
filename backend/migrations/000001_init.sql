-- +goose Up
CREATE TABLE IF NOT EXISTS sources (
    id          UUID PRIMARY KEY,
    name        TEXT NOT NULL,
    type        TEXT NOT NULL,
    rss_url     TEXT,
    active      BOOLEAN NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT sources_name_non_empty CHECK (char_length(trim(name)) > 0),
    CONSTRAINT sources_type_valid CHECK (type IN ('rss', 'manual')),
    CONSTRAINT sources_rss_config CHECK (
        (type = 'rss' AND rss_url IS NOT NULL AND char_length(trim(rss_url)) > 0)
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
    id           UUID PRIMARY KEY,
    source_id    UUID NOT NULL REFERENCES sources (id),
    title        TEXT NOT NULL,
    link         TEXT NOT NULL,
    description  TEXT NOT NULL DEFAULT '',
    thumbnail    TEXT,
    published_at TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    read_at      TIMESTAMPTZ,
    saved_at     TIMESTAMPTZ,
    CONSTRAINT entries_source_link_unique UNIQUE (source_id, link),
    CONSTRAINT entries_title_non_empty CHECK (char_length(trim(title)) > 0),
    CONSTRAINT entries_link_non_empty CHECK (char_length(trim(link)) > 0)
);

-- +goose Down
DROP TABLE IF EXISTS entries;
DROP TABLE IF EXISTS sources;
