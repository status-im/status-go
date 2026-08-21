CREATE TABLE blocks (
    network_id UNSIGNED BIGINT NOT NULL,
    address VARCHAR NOT NULL,
    blk_number BIGINT NOT NULL,
    blk_hash BIGINT NOT NULL,
    loaded BOOL DEFAULT FALSE,
    CONSTRAINT unique_mapping_for_account_to_block_per_network UNIQUE (address,blk_hash,network_id)
);

CREATE TABLE blocks_ranges (
    network_id UNSIGNED BIGINT NOT NULL,
    address VARCHAR NOT NULL,
    blk_from BIGINT NOT NULL,
    blk_to BIGINT NOT NULL,
    balance BLOB,
    nonce INTEGER);

CREATE TABLE blocks_ranges_sequential (
    network_id UNSIGNED BIGINT NOT NULL,
    address VARCHAR NOT NULL,
    blk_start BIGINT,
    blk_first BIGINT NOT NULL,
    blk_last BIGINT NOT NULL,
    PRIMARY KEY (network_id, address)
) WITHOUT ROWID;

CREATE TABLE pending_transactions (
    network_id UNSIGNED BIGINT NOT NULL,
    hash VARCHAR NOT NULL,
    timestamp UNSIGNED BIGINT NOT NULL,
    from_address VARCHAR NOT NULL,
    to_address VARCHAR,
    symbol VARCHAR,
    gas_price BLOB,
    gas_limit BLOB,
    value BLOB,
    data TEXT,
    type VARCHAR,
    additional_data TEXT,
    multi_transaction_id INT,
    PRIMARY KEY (network_id, hash)
) WITHOUT ROWID;

CREATE TABLE "saved_addresses" (
    address VARCHAR NOT NULL,
    name TEXT NOT NULL,
    favourite BOOLEAN NOT NULL DEFAULT FALSE,
    removed BOOLEAN NOT NULL DEFAULT FALSE,
    update_clock INT NOT NULL DEFAULT 0,
    chain_short_names VARCHAR DEFAULT "",
    ens_name VARCHAR DEFAULT "",
    is_test BOOLEAN DEFAULT FALSE,
    created_at INT DEFAULT 0,
    PRIMARY KEY (address, ens_name, is_test)
) WITHOUT ROWID;

CREATE TABLE token_balances (
    user_address VARCHAR NOT NULL,
    token_name VARCHAR NOT NULL,
    token_symbol VARCHAR NOT NULL,
    token_address VARCHAR NOT NULL,
    token_color VARCHAR NOT NULL DEFAULT "",
    token_decimals INT NOT NULL,
    token_description VARCHAR NOT NULL DEFAULT "",
    token_url VARCHAR NOT NULL DEFAULT "",
    balance VARCHAR NOT NULL,
    chain_id INT NOT NULL,
    PRIMARY KEY (user_address, chain_id, token_symbol) ON CONFLICT REPLACE
);

CREATE TABLE tokens (
    address VARCHAR NOT NULL,
    network_id UNSIGNED BIGINT NOT NULL,
    name TEXT NOT NULL,
    symbol VARCHAR NOT NULL,
    decimals UNSIGNED INT,
    color VARCHAR,
    PRIMARY KEY (address, network_id)
) WITHOUT ROWID;

CREATE TABLE visible_tokens (
    chain_id UNSIGNED INT,
    address VARCHAR NOT NULL
);

CREATE TABLE currency_format_cache (
    symbol VARCHAR NOT NULL,
    display_decimals INT NOT NULL,
    strip_trailing_zeroes BOOLEAN NOT NULL
);

CREATE TABLE multi_transactions (
    from_address VARCHAR NOT NULL,
    from_asset VARCHAR NOT NULL,
    from_amount VARCHAR NOT NULL,
    to_address VARCHAR NOT NULL,
    to_asset VARCHAR NOT NULL,
    type VARCHAR NOT NULL,
    timestamp UNSIGNED BIGINT NOT NULL,
    to_amount VARCHAR,
    from_network_id UNSIGNED BIGINT,
    to_network_id UNSIGNED BIGINT,
    cross_tx_id VARCHAR DEFAULT "",
    from_tx_hash BLOB,
    to_tx_hash BLOB
);

CREATE TABLE balance_history (
    chain_id UNSIGNED BIGINT NOT NULL,
    address VARCHAR NOT NULL,
    currency VARCHAR NOT NULL,
    block BIGINT NOT NULL,
    timestamp INT NOT NULL,
    bitset INT NOT NULL,
    balance BLOB
);

CREATE TABLE price_cache (
    token VARCHAR NOT NULL,
    currency VARCHAR NOT NULL,
    price REAL NOT NULL
);

CREATE TABLE IF NOT EXISTS collectibles_ownership_cache (
    chain_id UNSIGNED BIGINT NOT NULL,
    contract_address VARCHAR NOT NULL,
    token_id BLOB NOT NULL,
    owner_address VARCHAR NOT NULL
);

CREATE TABLE transfers (
    network_id UNSIGNED BIGINT NOT NULL,
    hash VARCHAR NOT NULL,
    address VARCHAR NOT NULL,
    blk_hash VARCHAR NOT NULL,
    tx BLOB,
    sender VARCHAR,
    receipt BLOB,
    log BLOB,
    type VARCHAR NOT NULL,
    blk_number BIGINT NOT NULL,
    timestamp UNSIGNED BIGINT NOT NULL,
    loaded BOOL DEFAULT 1,
    multi_transaction_id INT,
    base_gas_fee TEXT NOT NULL DEFAULT "",
    status INT,
    receipt_type INT,
    tx_hash BLOB,
    log_index INT,
    block_hash BLOB,
    cumulative_gas_used INT,
    contract_address TEXT,
    gas_used INT,
    tx_index INT,
    tx_type INT,
    protected BOOLEAN,
    gas_limit UNSIGNED INT,
    gas_price_clamped64 INT,
    gas_tip_cap_clamped64 INT,
    gas_fee_cap_clamped64 INT,
    amount_padded128hex CHAR(32),
    account_nonce INT,
    size INT,
    token_address BLOB,
    token_id BLOB,
    tx_from_address BLOB,
    tx_to_address BLOB,
    FOREIGN KEY(network_id,address,blk_hash) REFERENCES blocks(network_id,address,blk_hash) ON DELETE CASCADE,
    CONSTRAINT unique_transfer_per_address_per_network UNIQUE (hash,address,network_id)
);

CREATE INDEX balance_history_filter_entries ON balance_history (chain_id, address, currency, block, timestamp, bitset);

CREATE INDEX idx_transfers_blk_loaded ON transfers(blk_number, loaded);

CREATE UNIQUE INDEX price_cache_identify_entry ON price_cache (token, currency);

CREATE UNIQUE INDEX balance_history_identify_entry ON balance_history (chain_id, address, currency, block);

CREATE UNIQUE INDEX currency_format_cache_identify_entry ON currency_format_cache (symbol);

CREATE INDEX idx_transfers_filter
ON transfers (multi_transaction_id, loaded, timestamp, status, network_id, tx_from_address, tx_to_address, token_address, token_id, type);

CREATE INDEX idx_pending_transactions
ON pending_transactions (multi_transaction_id, from_address, to_address, network_id, timestamp, symbol);

CREATE INDEX idx_multi_transactions
ON multi_transactions (from_address, to_address, type, from_asset, timestamp, to_asset, from_amount, to_amount);

CREATE INDEX IF NOT EXISTS collectibles_ownership_filter_entries ON collectibles_ownership_cache (chain_id, owner_address);
