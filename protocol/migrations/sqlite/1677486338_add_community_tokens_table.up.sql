CREATE TABLE IF NOT EXISTS community_tokens (
  community_id TEXT NOT NULL,
  address TEXT NOT NULL,
  type INT NOT NULL,
  name TEXT NOT NULL,
  symbol TEXT NOT NULL,
  description TEXT NOT NULL,
  supply INT NOT NULL DEFAULT 0,
  infinite_supply BOOLEAN NOT NULL DEFAULT FALSE,
  transferable BOOLEAN NOT NULL DEFAULT FALSE,
  remote_self_destruct BOOLEAN NOT NULL DEFAULT FALSE,
  chain_id INT NOT NULL,
  deploy_state INT NOT NULL,
  image_base64 TEXT NOT NULL DEFAULT "",
  PRIMARY KEY(community_id, address, chain_id)
);
