CREATE TABLE IF NOT EXISTS collection_socials_cache (
    chain_id UNSIGNED BIGINT NOT NULL,
    contract_address VARCHAR NOT NULL,
    website VARCHAR NOT NULL,
    twitter_handle VARCHAR NOT NULL
);
