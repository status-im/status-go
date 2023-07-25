CREATE TABLE IF NOT EXISTS collectibles_ownership_cache (
    chain_id UNSIGNED BIGINT NOT NULL,
    contract_address VARCHAR NOT NULL,
    token_id BLOB NOT NULL,
    owner_address VARCHAR NOT NULL
);

CREATE INDEX IF NOT EXISTS collectibles_ownership_filter_entries ON collectibles_ownership_cache (chain_id, owner_address);
