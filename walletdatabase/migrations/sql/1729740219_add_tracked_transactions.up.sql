-- store state of tracked transactions
CREATE TABLE IF NOT EXISTS tracked_transactions(
    chain_id UNSIGNED BIGINT NOT NULL,
    tx_hash BLOB NOT NULL,
    tx_status STRING NOT NULL,
    timestamp INTEGER NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_tracked_transactions_per_chain_id_tx_hash ON tracked_transactions (chain_id, tx_hash);