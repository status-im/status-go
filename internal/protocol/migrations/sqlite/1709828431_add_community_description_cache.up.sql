CREATE TABLE IF NOT EXISTS encrypted_community_description_cache (
  community_id TEXT PRIMARY KEY,
  clock UINT64,
  description BLOB,
  UNIQUE(community_id) ON CONFLICT REPLACE
  );

CREATE TABLE IF NOT EXISTS encrypted_community_description_missing_keys (
        community_id TEXT,
        key_id TEXT,
        PRIMARY KEY (community_id, key_id),
        FOREIGN KEY (community_id) REFERENCES encrypted_community_description_cache(community_id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS encrypted_community_description_id_and_clock ON encrypted_community_description_cache(community_id, clock);
CREATE INDEX IF NOT EXISTS encrypted_community_description_key_ids ON encrypted_community_description_missing_keys(key_id);
