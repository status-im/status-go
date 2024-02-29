CREATE INDEX IF NOT EXISTS idx_group_timestamp_desc
ON hash_ratchet_encryption (group_id, key_timestamp DESC);

CREATE INDEX IF NOT EXISTS idx_hash_ratchet_encryption_keys_key_id
ON hash_ratchet_encryption (key_id);

CREATE INDEX IF NOT EXISTS idx_hash_ratchet_encryption_keys_deprecated_key_id
ON hash_ratchet_encryption (deprecated_key_id);

CREATE INDEX IF NOT EXISTS idx_hash_ratchet_cache_group_id_key_id_seq_no
ON hash_ratchet_encryption_cache (group_id, key_id, seq_no DESC);
