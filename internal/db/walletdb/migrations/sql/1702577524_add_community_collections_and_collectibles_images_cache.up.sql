-- Add columns
ALTER TABLE community_data_cache ADD COLUMN image_payload BLOB;
ALTER TABLE collection_data_cache ADD COLUMN image_payload BLOB;
ALTER TABLE collectible_data_cache ADD COLUMN image_payload BLOB;
