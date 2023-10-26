-- Only populated if communty_id is not empty
ALTER TABLE collectible_data_cache ADD COLUMN community_privileges_level UNSIGNED INT;

-- Holds community  metadata
CREATE TABLE IF NOT EXISTS community_data_cache (
    id TEXT PRIMARY KEY NOT NULL,
    name TEXT NOT NULL,
    color TEXT NOT NULL,
    image TEXT NOT NULL
);
