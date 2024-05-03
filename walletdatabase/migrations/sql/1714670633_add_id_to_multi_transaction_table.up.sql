-- Create a copy of multi_transactions table with an additional column for the primary key
CREATE TABLE new_table (
    id INTEGER PRIMARY KEY, -- Add a new column for the primary key
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
) WITHOUT ROWID;

-- Copy data
INSERT INTO new_table (id, from_address, from_asset, from_amount, to_address, to_asset, type, timestamp, to_amount, from_network_id, to_network_id, cross_tx_id, from_tx_hash, to_tx_hash)
    SELECT rowid, from_address, from_asset, from_amount, to_address, to_asset, type, timestamp, to_amount, from_network_id, to_network_id, cross_tx_id, from_tx_hash, to_tx_hash
    FROM multi_transactions;

-- Drop the existing table and rename the new table
DROP TABLE multi_transactions;
ALTER TABLE new_table RENAME TO multi_transactions;