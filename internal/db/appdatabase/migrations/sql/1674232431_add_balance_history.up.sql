-- One entry per chain_id, address, currency, block
-- bitset is used to select time interval granularity. The least significant bit has the coarsest granularity
-- balance is a serialized big.Int
CREATE TABLE IF NOT EXISTS balance_history (
    chain_id UNSIGNED BIGINT NOT NULL,
    address VARCHAR NOT NULL,
    currency VARCHAR NOT NULL,
    block BIGINT NOT NULL,
    timestamp INT NOT NULL,
    bitset INT NOT NULL,
    balance BLOB
);

CREATE UNIQUE INDEX IF NOT EXISTS balance_history_identify_entry ON balance_history (chain_id, address, currency, block);
CREATE INDEX IF NOT EXISTS balance_history_filter_entries ON balance_history (chain_id, address, currency, block, timestamp, bitset);