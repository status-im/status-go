CREATE TABLE IF NOT EXISTS token_balances (
  user_address VARCHAR NOT NULL,
  token_name VARCHAR NOT NULL,
  token_symbol VARCHAR NOT NULL,
  token_address VARCHAR NOT NULL,
  token_color VARCHAR NOT NULL DEFAULT "",
  token_decimals INT NOT NULL,
  token_description VARCHAR NOT NULL DEFAULT "",
  token_url VARCHAR NOT NULL DEFAULT "",
  balance VARCHAR NOT NULL,
  chain_id INT NOT NULL,
  PRIMARY KEY (user_address, chain_id, token_symbol) ON CONFLICT REPLACE
)
