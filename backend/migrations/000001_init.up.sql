CREATE TABLE IF NOT EXISTS sources (
    id          UUID PRIMARY KEY,
    name        TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT sources_name_non_empty CHECK (char_length(trim(name)) > 0)
);

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
