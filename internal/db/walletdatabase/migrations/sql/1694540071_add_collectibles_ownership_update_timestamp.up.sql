CREATE TABLE IF NOT EXISTS collectibles_ownership_update_timestamps (
    owner_address VARCHAR NOT NULL,
    chain_id UNSIGNED BIGINT NOT NULL,
    timestamp UNSIGNED BIGINT NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS collectibles_ownership_update_timestamps_identify_entry ON collectibles_ownership_update_timestamps (owner_address, chain_id);
