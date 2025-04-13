-- delete all from balance_history table, only once, since the approach has changed (storing token addresses instead of symbols (currencies)).
DELETE FROM balance_history;

DROP INDEX IF EXISTS balance_history_identify_entry;
DROP INDEX IF EXISTS balance_history_filter_entries;

ALTER TABLE balance_history RENAME COLUMN currency TO token_address;

CREATE UNIQUE INDEX IF NOT EXISTS balance_history_identify_entry ON balance_history (chain_id, address, token_address, block);
CREATE INDEX IF NOT EXISTS balance_history_filter_entries ON balance_history (chain_id, address, token_address, block, timestamp);