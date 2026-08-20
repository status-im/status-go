CREATE TABLE IF NOT EXISTS community_encryption_keys_requests (
    community_id BLOB NOT NULL,
    channel_id TEXT,
    requested_at INTEGER NOT NULL,
    requested_count INTEGER NOT NULL,
    PRIMARY KEY (community_id, channel_id)
);
