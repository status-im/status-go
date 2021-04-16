ALTER TABLE pending_transactions RENAME TO pending_transactions_old;

CREATE TABLE IF NOT EXISTS pending_transactions (
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
  PRIMARY KEY (network_id, hash)
) WITHOUT ROWID;

INSERT INTO pending_transactions(network_id, hash, from_address, to_address, type, additional_data, timestamp)
SELECT network_id, transaction_hash, from_address, to_address, type, data, 0 FROM pending_transactions_old;

DROP TABLE pending_transactions_old;
