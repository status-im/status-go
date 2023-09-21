ALTER TABLE collectible_data_cache ADD COLUMN community_id TEXT NOT NULL DEFAULT "";
UPDATE collectible_data_cache SET community_id = "";

ALTER TABLE collection_data_cache ADD COLUMN community_id TEXT NOT NULL DEFAULT "";
UPDATE collection_data_cache SET community_id = "";
