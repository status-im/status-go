CREATE TABLE IF NOT EXISTS saved_addresses (
  address VARCHAR NOT NULL,
  network_id UNSIGNED BIGINT NOT NULL,
  name TEXT NOT NULL,
  PRIMARY KEY (network_id, address)
) WITHOUT ROWID;
