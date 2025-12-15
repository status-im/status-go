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
color TEXT,
created_at DATETIME NOT NULL,
updated_at DATETIME NOT NULL
) WITHOUT ROWID;

CREATE UNIQUE INDEX unique_wallet_address ON accounts (wallet) WHERE (wallet);
CREATE UNIQUE INDEX unique_chat_address ON accounts (chat) WHERE (chat);
CREATE INDEX created_at_account ON accounts (created_at) WHERE (created_at);

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
network_id UNSIGNED BIGINT NOT NULL,
hash VARCHAR NOT NULL,
address VARCHAR NOT NULL,
blk_hash VARCHAR NOT NULL,
tx BLOB,
sender VARCHAR,
receipt BLOB,
log BLOB,
type VARCHAR NOT NULL,
blk_number BIGINT NOT NULL,
timestamp UNSIGNED BIGINT NOT NULL,
loaded BOOL DEFAULT 1,
FOREIGN KEY(network_id,address,blk_hash) REFERENCES blocks(network_id,address,blk_hash) ON DELETE CASCADE,
CONSTRAINT unique_transfer_per_address_per_network UNIQUE (hash,address,network_id)
);

CREATE TABLE IF NOT EXISTS blocks (
network_id UNSIGNED BIGINT NOT NULL,
address VARCHAR NOT NULL,
blk_number BIGINT NOT NULL,
blk_hash BIGINT NOT NULL,
loaded BOOL DEFAULT FALSE,
CONSTRAINT unique_mapping_for_account_to_block_per_network UNIQUE (address,blk_hash,network_id)
);

CREATE TABLE IF NOT EXISTS blocks_ranges (
network_id UNSIGNED BIGINT NOT NULL,
address VARCHAR NOT NULL,
blk_from BIGINT NOT NULL,
blk_to BIGINT NOT NULL
);

CREATE TABLE IF NOT EXISTS mailservers (
    id VARCHAR PRIMARY KEY,
    name VARCHAR NOT NULL,
    address VARCHAR NOT NULL,
    password VARCHAR,
    fleet VARCHAR NOT NULL
) WITHOUT ROWID;

CREATE TABLE IF NOT EXISTS mailserver_request_gaps (
    gap_from UNSIGNED INTEGER NOT NULL,
    gap_to UNSIGNED INTEGER NOT NULL,
    id TEXT PRIMARY KEY,
    chat_id TEXT NOT NULL
) WITHOUT ROWID;

CREATE INDEX mailserver_request_gaps_chat_id_idx ON mailserver_request_gaps (chat_id);

CREATE TABLE IF NOT EXISTS mailserver_topics (
    topic VARCHAR PRIMARY KEY,
    chat_ids VARCHAR,
    last_request INTEGER DEFAULT 1,
    discovery BOOLEAN DEFAULT FALSE,
    negotiated BOOLEAN DEFAULT FALSE
) WITHOUT ROWID;

CREATE TABLE IF NOT EXISTS mailserver_chat_request_ranges (
    chat_id VARCHAR PRIMARY KEY,
    lowest_request_from INTEGER,
    highest_request_to INTEGER
) WITHOUT ROWID;
