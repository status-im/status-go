-- store route input parameters
CREATE TABLE IF NOT EXISTS route_input_parameters (
    uuid TEXT NOT NULL AS (json_extract(route_input_params_json, '$.uuid')),
    route_input_params_json JSON NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_route_input_parameters_per_uuid ON route_input_parameters (uuid);

-- store route build tx parameters
CREATE TABLE IF NOT EXISTS route_build_tx_parameters (
    uuid TEXT NOT NULL AS (json_extract(route_build_tx_params_json, '$.uuid')),
    route_build_tx_params_json JSON NOT NULL,
    FOREIGN KEY(uuid) REFERENCES route_input_parameters(uuid) ON DELETE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_route_build_tx_parameters_per_uuid ON route_build_tx_parameters (uuid);

-- store route paths
CREATE TABLE IF NOT EXISTS route_paths (
    uuid TEXT NOT NULL,
    path_idx INTEGER NOT NULL,
    path_json JSON NOT NULL,
    FOREIGN KEY(uuid) REFERENCES route_input_parameters(uuid) ON DELETE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_route_path_per_uuid_index ON route_paths (uuid, path_idx);

-- store route path transactions
CREATE TABLE IF NOT EXISTS route_path_transactions (
    uuid TEXT NOT NULL,
    path_idx INTEGER NOT NULL,
    tx_idx INTEGER NOT NULL,
    is_approval BOOLEAN NOT NULL,
    chain_id UNSIGNED BIGINT NOT NULL,
    tx_hash BLOB NOT NULL,
    tx_args_json JSON NOT NULL,
    FOREIGN KEY(uuid, path_idx) REFERENCES route_paths(uuid, path_idx) ON DELETE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_route_path_transaction_per_uuid_path_idx_tx_idx ON route_path_transactions (uuid, path_idx, tx_idx);
CREATE UNIQUE INDEX IF NOT EXISTS idx_route_path_transaction_per_chain_id_tx_hash ON route_path_transactions (chain_id, tx_hash);

-- store sent transactions
CREATE TABLE IF NOT EXISTS sent_transactions (
    chain_id UNSIGNED BIGINT NOT NULL,
    tx_hash BLOB NOT NULL,
    tx_json JSON NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_sent_transactions_per_chain_id_tx_hash ON sent_transactions (chain_id, tx_hash);
