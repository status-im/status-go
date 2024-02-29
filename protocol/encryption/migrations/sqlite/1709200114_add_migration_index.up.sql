CREATE INDEX idx_group_timestamp_desc
ON hash_ratchet_encryption (group_id, key_timestamp DESC);
