CREATE TABLE IF NOT EXISTS collectible_data_cache (
    chain_id UNSIGNED BIGINT NOT NULL,
    contract_address VARCHAR NOT NULL,
    token_id BLOB NOT NULL,
    provider VARCHAR NOT NULL,
    name VARCHAR NOT NULL,
    description VARCHAR NOT NULL,
    permalink VARCHAR NOT NULL,
    image_url VARCHAR NOT NULL,
    animation_url VARCHAR NOT NULL,
    animation_media_type VARCHAR NOT NULL,
    background_color VARCHAR NOT NULL,
    token_uri VARCHAR NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS collectible_data_identify_entry ON collectible_data_cache (chain_id, contract_address, token_id);

CREATE TABLE IF NOT EXISTS collectible_traits_cache (
    chain_id UNSIGNED BIGINT NOT NULL,
    contract_address VARCHAR NOT NULL,
    token_id BLOB NOT NULL,
    trait_type VARCHAR NOT NULL,
    trait_value VARCHAR NOT NULL,
    display_type VARCHAR NOT NULL,
    max_value VARCHAR NOT NULL,
    FOREIGN KEY(chain_id, contract_address, token_id) REFERENCES collectible_data_cache(chain_id, contract_address, token_id) 
      ON UPDATE CASCADE 
      ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS collection_data_cache (
    chain_id UNSIGNED BIGINT NOT NULL,
    contract_address VARCHAR NOT NULL,
    provider VARCHAR NOT NULL,
    name VARCHAR NOT NULL,
    slug VARCHAR NOT NULL,
    image_url VARCHAR NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS collection_data_identify_entry ON collection_data_cache (chain_id, contract_address);

CREATE TABLE IF NOT EXISTS collection_traits_cache (
    chain_id UNSIGNED BIGINT NOT NULL,
    contract_address VARCHAR NOT NULL,
    trait_type VARCHAR NOT NULL,
    min REAL NOT NULL,
    max REAL NOT NULL,
    FOREIGN KEY(chain_id, contract_address) REFERENCES collection_data_cache(chain_id, contract_address) 
      ON UPDATE CASCADE 
      ON DELETE CASCADE
);