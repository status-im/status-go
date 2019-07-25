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

CREATE TABLE IF NOT EXISTS browsers (
id TEXT PRIMARY KEY,
name TEXT NOT NULL,
timestamp USGIGNED BIGINT,
dapp BOOLEAN DEFAULT false,
historyIndex UNSIGNED INT
) WITHOUT ROWID;

CREATE TABLE IF NOT EXISTS browsers_history (
browser_id TEXT NOT NULL,
history TEXT,
FOREIGN KEY(browser_id) REFERENCES browsers(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS dapps (
name TEXT PRIMARY KEY
) WITHOUT ROWID;

CREATE TABLE IF NOT EXISTS permissions (
dapp_name TEXT NOT NULL,
permission TEXT NOT NULL,
FOREIGN KEY(dapp_name) REFERENCES dapps(name) ON DELETE CASCADE
);


CREATE TABLE IF NOT EXISTS transfers (
hash VARCHAR UNIQUE,
address VARCHAR NOT NULL,
blk_hash VARCHAR NOT NULL,
tx BLOB,
sender VARCHAR NOT NULL,
receipt BLOB,
log BLOB,
type VARCHAR NOT NULL,
FOREIGN KEY(blk_hash) REFERENCES blocks(hash) ON DELETE CASCADE,
CONSTRAINT unique_transfer_on_hash_address UNIQUE (hash,address)
);

CREATE TABLE IF NOT EXISTS blocks (
hash VARCHAR PRIMARY KEY,
number BIGINT UNIQUE NOT NULL,
timestamp UNSIGNED BIGINT NOT NULL,
head BOOL DEFAULT FALSE
) WITHOUT ROWID;

CREATE TABLE IF NOT EXISTS accounts_to_blocks (
address VARCHAR NOT NULL,
blk_number BIGINT NOT NULL,
sync INT,
FOREIGN KEY(blk_number) REFERENCES blocks(number) ON DELETE CASCADE,
CONSTRAINT unique_mapping_on_address_block_number UNIQUE (address,blk_number)
);
