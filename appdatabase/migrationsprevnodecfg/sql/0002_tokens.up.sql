CREATE TABLE IF NOT EXISTS tokens (
  address VARCHAR NOT NULL,
  network_id UNSIGNED BIGINT NOT NULL,
  name TEXT NOT NULL,
  symbol VARCHAR NOT NULL,
  decimals UNSIGNED INT,
  color VARCHAR,
  PRIMARY KEY (address, network_id)
) WITHOUT ROWID;

