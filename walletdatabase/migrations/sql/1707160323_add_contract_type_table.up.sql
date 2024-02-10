CREATE TABLE IF NOT EXISTS contract_type_cache (
    chain_id UNSIGNED BIGINT NOT NULL,
    contract_address VARCHAR NOT NULL,
    contract_type INTEGER NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS contract_type_identify_entry ON contract_type_cache (chain_id, contract_address);
