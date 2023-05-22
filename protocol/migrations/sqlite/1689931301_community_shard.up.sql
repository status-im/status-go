ALTER TABLE communities_communities ADD COLUMN shard_cluster INT DEFAULT NULL;
ALTER TABLE communities_communities ADD COLUMN shard_index INT DEFAULT NULL;

UPDATE communities_communities SET shard_cluster = NULL; -- TODO confirm this
UPDATE communities_communities SET shard_index = NULL; -- TODO confirm this
