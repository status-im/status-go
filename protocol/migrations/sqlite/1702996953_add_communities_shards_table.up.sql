CREATE TABLE IF NOT EXISTS communities_shards (
    community_id BLOB NOT NULL PRIMARY KEY ON CONFLICT REPLACE,
    shard_cluster INT DEFAULT NULL,
    shard_index INT DEFAULT NULL,
    clock INT DEFAULT 0
);