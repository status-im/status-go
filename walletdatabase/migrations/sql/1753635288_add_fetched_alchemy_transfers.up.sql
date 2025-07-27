CREATE TABLE IF NOT EXISTS fetched_alchemy_transfers (
    transfer JSON NOT NULL,
    chain_id UNSIGNED INTEGER NOT NULL,
    address TEXT NOT NULL,
    hash TEXT AS (json_extract(transfer, '$.hash')),
    block_number INTEGER AS (
    CAST('0x' || substr(json_extract(transfer, '$.blockNum'), 3) AS INTEGER))
);

CREATE INDEX IF NOT EXISTS idx_fetched_transfers_chain_address ON fetched_alchemy_transfers (chain_id, address);
CREATE INDEX IF NOT EXISTS idx_fetched_transfers_hash ON fetched_alchemy_transfers (hash);
