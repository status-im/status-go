CREATE TABLE IF NOT EXISTS fetched_alchemy_transfers (
    transfer JSON NOT NULL,
    chain_id UNSIGNED INTEGER NOT NULL,
    address BLOB NOT NULL,
    hash TEXT AS (json_extract(transfer, '$.hash')),
    retrieved_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    block_number TEXT AS (json_extract(transfer, '$.blockNum'))
);

CREATE INDEX IF NOT EXISTS idx_fetched_transfers_activity ON fetched_alchemy_transfers (chain_id, address, hash);
CREATE INDEX IF NOT EXISTS idx_fetched_transfers_lookup ON fetched_alchemy_transfers (chain_id, hash);
