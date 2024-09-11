-- store raw block headers
CREATE TABLE IF NOT EXISTS blockchain_data_blocks (
    chain_id UNSIGNED BIGINT NOT NULL,
    block_number BLOB NOT NULL,
    block_hash BLOB NOT NULL,
    block_header_json JSON NOT NULL,
    block_json JSON,
    CONSTRAINT unique_block_header_per_chain_per_block_number UNIQUE (chain_id,block_number) ON CONFLICT REPLACE,
    CONSTRAINT unique_block_header_per_chain_per_block_hash UNIQUE (chain_id,block_hash) ON CONFLICT REPLACE
) WITHOUT ROWID;

CREATE INDEX IF NOT EXISTS idx_blockchain_data_block_headers_chain_id_block_number ON blockchain_data_block_headers (chain_id, block_number);
CREATE INDEX IF NOT EXISTS idx_blockchain_data_block_headers_chain_id_block_hash ON blockchain_data_block_headers (chain_id, block_hash);

-- store raw transactions
CREATE TABLE IF NOT EXISTS blockchain_data_transactions (
    chain_id UNSIGNED BIGINT NOT NULL,
    block_hash BLOB NOT NULL,
    transaction_hash BLOB NOT NULL,
    transaction_json JSON NOT NULL,
    receipt_json JSON,
    CONSTRAINT unique_transaction_per_chain_per_transaction_hash UNIQUE (chain_id, transaction_hash) ON CONFLICT REPLACE,
    FOREIGN KEY(chain_id, block_hash) REFERENCES blockchain_data_block_headers(chain_id, block_hash) 
      ON DELETE CASCADE
) WITHOUT ROWID;

CREATE INDEX IF NOT EXISTS idx_blockchain_data_transactions_chain_id_transaction_hash ON blockchain_data_transactions (chain_id, transaction_hash);
CREATE INDEX IF NOT EXISTS idx_blockchain_data_transactions_chain_id_block_hash ON blockchain_data_transactions (chain_id, block_hash);
