CREATE TABLE IF NOT EXISTS collection_socials_cache (
    chain_id UNSIGNED BIGINT NOT NULL,
    contract_address VARCHAR NOT NULL,
    provider VARCHAR NOT NULL,
    website VARCHAR NOT NULL,
    twitter_handle VARCHAR NOT NULL,
    FOREIGN KEY(chain_id, contract_address) REFERENCES collection_data_cache(chain_id, contract_address) ON DELETE CASCADE
);
