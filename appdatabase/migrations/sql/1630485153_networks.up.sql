CREATE TABLE IF NOT EXISTS networks (
  chain_id UNSIGNED BIGINT NOT NULL,
  chain_name VARCHAR NOT NULL,
  rpc_url VARCHAR NOT NULL,
  block_explorer_url VARCHAR,
  icon_url VARCHAR,
  native_currency_name VARCHAR,
  native_currency_symbol VARCHAR,
  native_currency_decimals UNSIGNED INT,
  is_test BOOLEAN,
  layer UNSIGNED INT,
  enabled Boolean,
  PRIMARY KEY (chain_id)
) WITHOUT ROWID;

