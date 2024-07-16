-- Step 1: Create a temporary table to store unique records
CREATE TABLE IF NOT EXISTS balance_history_temp AS
SELECT DISTINCT chain_id, address, currency, block, timestamp, balance
FROM balance_history;

-- Step 2: Truncate the original table
DELETE FROM balance_history;

-- Step 3: Insert unique records back into the original table
INSERT INTO balance_history (chain_id, address, currency, block, timestamp, balance)
SELECT chain_id, address, currency, block, timestamp, balance
FROM balance_history_temp;

-- Step 4: Drop the temporary table
DROP TABLE balance_history_temp;

-- Step 5: Recreate the indices
CREATE UNIQUE INDEX IF NOT EXISTS balance_history_identify_entry ON balance_history (chain_id, address, currency, block);
CREATE INDEX IF NOT EXISTS balance_history_filter_entries ON balance_history (chain_id, address, currency, block, timestamp);
