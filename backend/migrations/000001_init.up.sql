CREATE TABLE IF NOT EXISTS feeds (
    id          UUID PRIMARY KEY,
    slug        TEXT NOT NULL,
    title       TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    status      TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT feeds_slug_unique UNIQUE (slug),
    CONSTRAINT feeds_title_non_empty CHECK (char_length(trim(title)) > 0),
    CONSTRAINT feeds_slug_non_empty CHECK (char_length(trim(slug)) > 0),
    CONSTRAINT feeds_status_valid CHECK (status IN ('DRAFT', 'PUBLISHED', 'ARCHIVED'))
);
