ALTER TABLE communities_communities ADD COLUMN shard_cluster INT DEFAULT NULL;
ALTER TABLE communities_communities ADD COLUMN shard_index INT DEFAULT NULL;
