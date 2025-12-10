-- Create new table with updated structure and primary key
CREATE TABLE token_balances_new (
    user_address VARCHAR NOT NULL,
    token_address VARCHAR NOT NULL,
    token_description VARCHAR NOT NULL DEFAULT "",
    token_url VARCHAR NOT NULL DEFAULT "",
    balance VARCHAR NOT NULL,
    raw_balance VARCHAR NOT NULL DEFAULT "0",
    chain_id INT NOT NULL,
    PRIMARY KEY (user_address, chain_id, token_address) ON CONFLICT REPLACE
);

-- Copy data from old table to new table (excluding dropped columns)
INSERT INTO token_balances_new (user_address, token_address, token_description, token_url, balance, raw_balance, chain_id)
SELECT user_address, token_address, token_description, token_url, balance, raw_balance, chain_id
FROM token_balances;

-- Drop old table
DROP TABLE token_balances;

-- Rename new table to original name
ALTER TABLE token_balances_new RENAME TO token_balances;