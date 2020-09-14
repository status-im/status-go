CREATE TABLE IF NOT EXISTS pending_transactions (
  network_id UNSIGNED BIGINT NOT NULL,
  transaction_hash VARCHAR NOT NULL,
  blk_number BIGINT NOT NULL,
  from_address VARCHAR NOT NULL,
  to_address VARCHAR NOT NULL,
  type VARCHAR NOT NULL,
  data TEXT,
  PRIMARY KEY (network_id, transaction_hash)
) WITHOUT ROWID;

