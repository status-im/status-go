-- One entry per chain_id, address, currency, block
-- balance is a serialized big.Int
CREATE TABLE IF NOT EXISTS balance_history_v2 (
    chain_id UNSIGNED BIGINT NOT NULL,
    address VARCHAR NOT NULL,
    currency VARCHAR NOT NULL,
    block BIGINT NOT NULL,
    timestamp INT NOT NULL,
    balance BLOB
);

CREATE UNIQUE INDEX IF NOT EXISTS balance_history_identify_entry ON balance_history_v2 (chain_id, address, currency, block);
CREATE INDEX IF NOT EXISTS balance_history_filter_entries ON balance_history_v2 (chain_id, address, currency, block, timestamp);

DROP TABLE balance_history;
ALTER TABLE balance_history_v2 RENAME TO balance_history;

