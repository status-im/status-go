CREATE TABLE IF NOT EXISTS fetched_activity_fetch_parameters (
    id TEXT PRIMARY KEY NOT NULL,
    chain_id UNSIGNED BIGINT NOT NULL,
    address BLOB NOT NULL AS (unhex(substr(json_extract(parameters, '$.address'),3))),
    parameters JSON NOT NULL,
    provider TEXT NOT NULL,
    previous_cursor TEXT,
    next_cursor TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_fetched_activity_fetch_parameters_per_chain_id_address ON fetched_activity_fetch_parameters (chain_id, address);

CREATE TABLE IF NOT EXISTS fetched_activity_transactions (
    fetch_parameters_id TEXT NOT NULL,
    tx_hash BLOB NOT NULL,
    processed BOOLEAN NOT NULL DEFAULT FALSE,
    FOREIGN KEY (fetch_parameters_id) REFERENCES fetched_activity_fetch_parameters(id)
);

CREATE TABLE IF NOT EXISTS fetched_activity_entries (
    fetch_parameters_id TEXT NOT NULL,
    entry JSON NOT NULL,
    timestamp INTEGER NOT NULL AS (json_extract(entry, '$.timestamp')),
    chain_id_out UNSIGNED INTEGER AS (json_extract(entry, '$.chainIdOut')),
    chain_id_in UNSIGNED INTEGER AS (json_extract(entry, '$.chainIdIn')),
    sender BLOB AS (unhex(substr(json_extract(entry, '$.sender'),3))),
    recipient BLOB AS (unhex(substr(json_extract(entry, '$.recipient'),3))),
    tx_hash BLOB AS (unhex(substr(json_extract(entry, '$.txHash'),3))),
    FOREIGN KEY (fetch_parameters_id) REFERENCES fetched_activity_fetch_parameters(id)
    FOREIGN KEY (tx_hash) REFERENCES fetched_activity_transactions(tx_hash)
);

CREATE INDEX IF NOT EXISTS idx_fetched_activity_entries_per_sender ON fetched_activity_entries (sender);
CREATE INDEX IF NOT EXISTS idx_fetched_activity_entries_per_recipient ON fetched_activity_entries (recipient);
CREATE INDEX IF NOT EXISTS idx_fetched_activity_entries_per_chain_id_out ON fetched_activity_entries (chain_id_out);
CREATE INDEX IF NOT EXISTS idx_fetched_activity_entries_per_chain_id_in ON fetched_activity_entries (chain_id_in);
CREATE INDEX IF NOT EXISTS idx_fetched_activity_entries_per_tx_hash ON fetched_activity_entries (tx_hash);