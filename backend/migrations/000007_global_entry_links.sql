-- +goose Up
-- Goose runs this migration in a transaction. There are currently no foreign
-- keys referencing entries; see the migration runbook before adding any.
CREATE TABLE entries_new (
    id TEXT PRIMARY KEY,
    source_id TEXT NOT NULL REFERENCES sources (id),
    title TEXT NOT NULL,
    link TEXT,
    description TEXT NOT NULL DEFAULT '',
    thumbnail TEXT,
    published_at INTEGER,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    read_at INTEGER,
    saved_at INTEGER,
    CONSTRAINT entries_title_non_empty CHECK (length(trim(title)) > 0),
    CONSTRAINT entries_link_non_empty CHECK (link IS NULL OR length(link) > 0)
);

-- Match Go strings.TrimSpace's Unicode White Space set.
INSERT INTO entries_new
WITH normalized AS (
    SELECT *, NULLIF(trim(link, char(9,10,11,12,13,32,133,160,5760,8192,8193,8194,8195,8196,8197,8198,8199,8200,8201,8202,8232,8233,8239,8287,12288)), '') AS normalized_link
    FROM entries
), ranked AS (
    SELECT *, ROW_NUMBER() OVER (PARTITION BY normalized_link ORDER BY (saved_at IS NOT NULL) DESC, created_at, id) AS rank,
        MIN(read_at) OVER (PARTITION BY normalized_link) AS first_read,
        MIN(saved_at) OVER (PARTITION BY normalized_link) AS first_saved
    FROM normalized
)
SELECT id, source_id, title, normalized_link, description, thumbnail, published_at, created_at, updated_at,
       CASE WHEN normalized_link IS NULL THEN read_at ELSE first_read END,
       CASE WHEN normalized_link IS NULL THEN saved_at ELSE first_saved END
FROM ranked WHERE normalized_link IS NULL OR rank = 1;

DROP TABLE entries;
ALTER TABLE entries_new RENAME TO entries;
CREATE UNIQUE INDEX entries_link_unique ON entries(link) WHERE link IS NOT NULL;

-- +goose Down
-- Consolidated rows cannot be reconstructed. Restore a backup instead.
SELECT * FROM restore_pre_migration_backup_to_undo_entry_consolidation;
