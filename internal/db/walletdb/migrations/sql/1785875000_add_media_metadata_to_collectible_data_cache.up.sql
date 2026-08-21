ALTER TABLE collectible_data_cache ADD COLUMN thumbnail_url TEXT NOT NULL DEFAULT "";
ALTER TABLE collectible_data_cache ADD COLUMN image_size INTEGER NOT NULL DEFAULT 0;
ALTER TABLE collectible_data_cache ADD COLUMN thumbnail_size INTEGER NOT NULL DEFAULT 0;
ALTER TABLE collectible_data_cache ADD COLUMN animation_size INTEGER NOT NULL DEFAULT 0;
ALTER TABLE collectible_data_cache ADD COLUMN metadata_version INTEGER NOT NULL DEFAULT 0;
