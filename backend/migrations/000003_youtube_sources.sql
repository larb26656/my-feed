-- +goose NO TRANSACTION
-- +goose Up
PRAGMA foreign_keys = OFF;
BEGIN;
CREATE TABLE sources_new (
 id TEXT PRIMARY KEY, name TEXT NOT NULL CHECK (length(trim(name)) > 0),
 type TEXT NOT NULL CHECK (type IN ('rss', 'manual', 'youtube')),
 rss_url TEXT, youtube_channel_id TEXT,
 active INTEGER NOT NULL DEFAULT 1 CHECK (active IN (0, 1)),
 created_at INTEGER NOT NULL, updated_at INTEGER NOT NULL,
 CHECK ((type = 'rss' AND rss_url IS NOT NULL AND length(trim(rss_url)) > 0 AND youtube_channel_id IS NULL)
 OR (type = 'manual' AND rss_url IS NULL AND youtube_channel_id IS NULL)
 OR (type = 'youtube' AND rss_url IS NULL AND youtube_channel_id IS NOT NULL AND length(youtube_channel_id) = 24 AND substr(youtube_channel_id,1,2) = 'UC' AND youtube_channel_id NOT GLOB '*[^A-Za-z0-9_-]*'))
);
INSERT INTO sources_new (id,name,type,rss_url,active,created_at,updated_at) SELECT id,name,type,rss_url,active,created_at,updated_at FROM sources;
DROP TABLE sources;
ALTER TABLE sources_new RENAME TO sources;
CREATE UNIQUE INDEX sources_rss_url_unique ON sources(rss_url) WHERE rss_url IS NOT NULL;
CREATE UNIQUE INDEX sources_manual_singleton ON sources(type) WHERE type = 'manual';
CREATE UNIQUE INDEX sources_youtube_channel_unique ON sources(youtube_channel_id) WHERE youtube_channel_id IS NOT NULL;
COMMIT;
PRAGMA foreign_keys = ON;

-- +goose Down
-- YouTube Sources cannot be represented by the previous schema.
SELECT * FROM youtube_source_schema_cannot_be_downgraded;
