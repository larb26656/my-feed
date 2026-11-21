-- +goose Up
ALTER TABLE sources ADD COLUMN profile_image_id TEXT;
CREATE UNIQUE INDEX sources_profile_image_id_unique ON sources(profile_image_id) WHERE profile_image_id IS NOT NULL;

-- +goose Down
DROP INDEX sources_profile_image_id_unique;
ALTER TABLE sources DROP COLUMN profile_image_id;
