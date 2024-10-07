-- store raw blocks
CREATE TABLE IF NOT EXISTS blockchain_data_blocks (
    chain_id UNSIGNED BIGINT NOT NULL,
    block_number BLOB NOT NULL,
    block_hash BLOB NOT NULL,
    with_transaction_details BOOLEAN NOT NULL,
    block_json JSON NOT NULL,
    CONSTRAINT unique_block_per_chain_per_block_number UNIQUE (chain_id,block_number,with_transaction_details) ON CONFLICT REPLACE,
    CONSTRAINT unique_block_per_chain_per_block_hash UNIQUE (chain_id,block_hash,with_transaction_details) ON CONFLICT REPLACE
);

CREATE INDEX IF NOT EXISTS idx_blockchain_data_blocks_chain_id_block_number ON blockchain_data_blocks (chain_id, block_number, with_transaction_details);
CREATE INDEX IF NOT EXISTS idx_blockchain_data_blocks_chain_id_block_hash ON blockchain_data_blocks (chain_id, block_hash, with_transaction_details);

-- store raw block uncles
CREATE TABLE IF NOT EXISTS blockchain_data_block_uncles (
    chain_id UNSIGNED BIGINT NOT NULL,
    block_hash BLOB NOT NULL,
    uncle_index UNSIGNED BIGINT NOT NULL,
    block_uncle_json JSON,
    PRIMARY KEY (chain_id, block_hash, uncle_index),
    CONSTRAINT unique_block_uncles_per_chain_per_block_hash_per_index UNIQUE (chain_id,block_hash,uncle_index) ON CONFLICT REPLACE
) WITHOUT ROWID;

CREATE INDEX IF NOT EXISTS idx_blockchain_data_block_uncles_chain_id_block_hash_uncle_index ON blockchain_data_block_uncles (chain_id, block_hash, uncle_index);

-- store raw transactions
CREATE TABLE IF NOT EXISTS blockchain_data_transactions (
    chain_id UNSIGNED BIGINT NOT NULL,
    transaction_hash BLOB NOT NULL,
    transaction_json JSON NOT NULL,
    PRIMARY KEY (chain_id, transaction_hash),
    CONSTRAINT unique_transaction_per_chain_per_transaction_hash UNIQUE (chain_id, transaction_hash) ON CONFLICT REPLACE
) WITHOUT ROWID;

CREATE INDEX IF NOT EXISTS idx_blockchain_data_transactions_chain_id_transaction_hash ON blockchain_data_transactions (chain_id, transaction_hash);

-- store raw transaction receipts
CREATE TABLE IF NOT EXISTS blockchain_data_receipts (
    chain_id UNSIGNED BIGINT NOT NULL,
    transaction_hash BLOB NOT NULL,
    receipt_json JSON NOT NULL,
    PRIMARY KEY (chain_id, transaction_hash),
    CONSTRAINT unique_receipt_per_chain_per_transaction_hash UNIQUE (chain_id, transaction_hash) ON CONFLICT REPLACE
) WITHOUT ROWID;

CREATE INDEX IF NOT EXISTS idx_blockchain_data_receipts_chain_id_transaction_hash ON blockchain_data_receipts (chain_id, transaction_hash);

-- store balances
CREATE TABLE IF NOT EXISTS blockchain_data_balances (
    chain_id UNSIGNED BIGINT NOT NULL,
    account BLOB NOT NULL,
    block_number BLOB NOT NULL,
    balance BLOB NOT NULL,
    PRIMARY KEY (chain_id, account, block_number),
    CONSTRAINT unique_balance_per_chain_per_account_per_block_number UNIQUE (chain_id, account, block_number) ON CONFLICT REPLACE
) WITHOUT ROWID;

CREATE INDEX IF NOT EXISTS idx_blockchain_data_balances_chain_id_account_block_number ON blockchain_data_balances (chain_id, account, block_number);

-- store transaction counts
CREATE TABLE IF NOT EXISTS blockchain_data_transaction_counts (
    chain_id UNSIGNED BIGINT NOT NULL,
    account BLOB NOT NULL,
    block_number BLOB NOT NULL,
    transaction_count BIGINT NOT NULL,
    PRIMARY KEY (chain_id, account, block_number),
    CONSTRAINT unique_transaction_count_per_chain_per_account_per_block_number UNIQUE (chain_id, account, block_number) ON CONFLICT REPLACE
) WITHOUT ROWID;

CREATE INDEX IF NOT EXISTS idx_blockchain_data_transaction_count_chain_id_account_block_number ON blockchain_data_transaction_counts (chain_id, account, block_number);