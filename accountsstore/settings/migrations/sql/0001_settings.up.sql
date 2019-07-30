CREATE TABLE IF NOT EXISTS settings (
type VARCHAR PRIMARY KEY,
value BLOB
) WITHOUT ROWID;

CREATE TABLE IF NOT EXISTS accounts (
address VARCHAR PRIMARY KEY,
main BOOLEAN,
wallet BOOLEAN,
chat BOOLEAN,
watch BOOLEAN
) WITHOUT ROWID;

CREATE UNIQUE INDEX unique_main_address ON accounts (main) WHERE (main);
CREATE UNIQUE INDEX unique_wallet_address ON accounts (wallet) WHERE (wallet);
CREATE UNIQUE INDEX unique_chat_address ON accounts (chat) WHERE (chat);
