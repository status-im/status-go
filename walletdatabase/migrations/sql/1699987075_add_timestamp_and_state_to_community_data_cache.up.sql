-- Holds community metadata state
CREATE TABLE IF NOT EXISTS community_data_cache_state (
    id TEXT PRIMARY KEY NOT NULL,
    last_update_timestamp UNSIGNED BIGINT NOT NULL,
    last_update_successful BOOLEAN NOT NULL
);

INSERT INTO community_data_cache_state (id, last_update_timestamp, last_update_successful)
  SELECT id, 1, 1 FROM community_data_cache;

-- Recreate community_data_cache with state constraints
ALTER TABLE community_data_cache RENAME TO community_data_cache_old;

CREATE TABLE IF NOT EXISTS community_data_cache (
    id TEXT PRIMARY KEY NOT NULL,
    name TEXT NOT NULL,
    color TEXT NOT NULL,
    image TEXT NOT NULL,
    FOREIGN KEY(id) REFERENCES community_data_cache_state(id) ON DELETE CASCADE
);

INSERT INTO community_data_cache
  SELECT id, name, color, image
  FROM community_data_cache_old;

DROP TABLE community_data_cache_old;
