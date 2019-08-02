CREATE TABLE IF NOT EXISTS settings (
type VARCHAR PRIMARY KEY,
value BLOB
) WITHOUT ROWID;

CREATE TABLE IF NOT EXISTS accounts (
address VARCHAR PRIMARY KEY,
wallet BOOLEAN,
chat BOOLEAN,
type TEXT,
storage TEXT,
pubkey BLOB,
path TEXT,
name TEXT,
color TEXT
) WITHOUT ROWID;

CREATE UNIQUE INDEX unique_wallet_address ON accounts (wallet) WHERE (wallet);
CREATE UNIQUE INDEX unique_chat_address ON accounts (chat) WHERE (chat);
