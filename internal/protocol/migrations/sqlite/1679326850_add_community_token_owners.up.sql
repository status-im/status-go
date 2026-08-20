CREATE TABLE IF NOT EXISTS community_token_owners (
  chain_id INT NOT NULL,
  address TEXT NOT NULL,
  owner TEXT NOT NULL COLLATE NOCASE,
  amount INT NOT NULL,
  PRIMARY KEY(chain_id, address, owner)
);
