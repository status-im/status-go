-- This was renamed by mistake from 1662447680_add_keypairs_table.up.sql and is kept for backwards compatibility being a no-op
CREATE TABLE IF NOT EXISTS keypairs (
    keycard_uid VARCHAR NOT NULL,
    keycard_name VARCHAR NOT NULL,
    keycard_locked BOOLEAN DEFAULT FALSE,
    account_address VARCHAR NOT NULL,
    key_uid VARCHAR NOT NULL
);