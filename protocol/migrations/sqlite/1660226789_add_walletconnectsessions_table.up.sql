CREATE TABLE IF NOT EXISTS wallet_connect_v1_sessions (
  peer_id PRIMARY KEY NOT NULL,
  dapp_name TEXT NOT NULL,
  dapp_url TEXT NOT NULL,
  info TEXT NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
