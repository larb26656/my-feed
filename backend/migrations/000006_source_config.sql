-- +goose NO TRANSACTION
-- +goose Up
PRAGMA foreign_keys = OFF;
BEGIN;
CREATE TABLE sources_new (
 id TEXT PRIMARY KEY,
 name TEXT NOT NULL CHECK (length(trim(name)) > 0),
 type TEXT NOT NULL,
 config TEXT NOT NULL CHECK (json_valid(config)),
 key TEXT NOT NULL,
 rss_url TEXT,
 youtube_channel_id TEXT,
 active INTEGER NOT NULL DEFAULT 1 CHECK (active IN (0, 1)),
 created_at INTEGER NOT NULL,
 updated_at INTEGER NOT NULL,
 profile_image_id TEXT
);
INSERT INTO sources_new (id,name,type,config,key,rss_url,youtube_channel_id,active,created_at,updated_at,profile_image_id)
SELECT id,name,type,
 CASE type WHEN 'rss' THEN json_object('url',rss_url) WHEN 'youtube' THEN json_object('channelId',youtube_channel_id) ELSE json_object() END,
 CASE type WHEN 'rss' THEN rss_url WHEN 'youtube' THEN youtube_channel_id ELSE 'singleton' END,
 rss_url,youtube_channel_id,active,created_at,updated_at,profile_image_id FROM sources;
DROP TABLE sources;
ALTER TABLE sources_new RENAME TO sources;
CREATE UNIQUE INDEX sources_type_key_unique ON sources(type,key);
CREATE UNIQUE INDEX sources_manual_singleton ON sources(type) WHERE type = 'manual';
CREATE UNIQUE INDEX sources_profile_image_id_unique ON sources(profile_image_id) WHERE profile_image_id IS NOT NULL;
COMMIT;
PRAGMA foreign_keys = ON;

-- +goose Down
SELECT * FROM source_config_schema_cannot_be_downgraded;
