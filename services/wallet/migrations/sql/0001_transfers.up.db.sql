CREATE TABLE IF NOT EXISTS transfers (
hash VARCHAR UNIQUE,
address VARCHAR NOT NULL,
blk_hash VARCHAR NOT NULL,
tx BLOB,
receipt BLOB,
type VARCHAR NOT NULL,
FOREIGN KEY(blk_hash) REFERENCES blocks(hash) ON DELETE CASCADE,
CONSTRAINT unique_transfer_on_hash_address UNIQUE (hash,address)
);

CREATE TABLE IF NOT EXISTS blocks (
hash VARCHAR PRIMARY KEY,
number BIGINT UNIQUE NOT NULL,
head BOOL DEFAULT FALSE
) WITHOUT ROWID;

CREATE TABLE IF NOT EXISTS accounts_to_blocks (
address VARCHAR NOT NULL,
blk_number BIGINT NOT NULL,
sync INT,
FOREIGN KEY(blk_number) REFERENCES blocks(number) ON DELETE CASCADE,
CONSTRAINT unique_mapping_on_address_block_number UNIQUE (address,blk_number)
);
