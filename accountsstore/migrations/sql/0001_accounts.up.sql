CREATE TABLE IF NOT EXISTS accounts (
address VARCHAR PRIMARY KEY,
name TEXT NOT NULL,
loginTimestamp BIG INT
) WITHOUT ROWID;

CREATE TABLE IF NOT EXISTS configurations (
address VARCHAR NOT NULL,
type VARCHAR NOT NULL,
value BLOB,
FOREIGN KEY(address) REFERENCES accounts(address) ON DELETE CASCADE,
CONSTRAINT unique_config_type_per_address UNIQUE (address,type)
)
