CREATE TABLE IF NOT EXISTS token_historical_ownership (
    owner_address VARCHAR NOT NULL,
    chain_id UNSIGNED BIGINT NOT NULL,
    token_address VARCHAR NOT NULL,
    first_detected_timestamp UNSIGNED BIGINT NOT NULL,
    last_detected_timestamp UNSIGNED BIGINT NOT NULL,
    PRIMARY KEY (owner_address, chain_id, token_address)
);

CREATE INDEX IF NOT EXISTS token_historical_ownership_owner_address ON token_historical_ownership(owner_address);
