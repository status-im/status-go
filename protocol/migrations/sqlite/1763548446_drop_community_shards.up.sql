ALTER TABLE communities_communities DROP COLUMN shard_cluster;
ALTER TABLE communities_communities DROP COLUMN shard_index;

DROP TABLE IF EXISTS communities_shards;
