CREATE TABLE IF NOT EXISTS multi_transactions (
    from_address VARCHAR NOT NULL,
    from_asset VARCHAR NOT NULL,
    from_amount VARCHAR NOT NULL,
    to_address VARCHAR NOT NULL,
    to_asset VARCHAR NOT NULL,
    type VARCHAR NOT NULL,
    timestamp UNSIGNED BIGINT NOT NULL
);

ALTER TABLE pending_transactions ADD COLUMN multi_transaction_id INT;
ALTER TABLE transfers ADD COLUMN multi_transaction_id INT;