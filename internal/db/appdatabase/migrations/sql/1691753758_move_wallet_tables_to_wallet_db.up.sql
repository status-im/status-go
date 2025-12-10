DROP TABLE blocks;
DROP TABLE blocks_ranges;
DROP TABLE blocks_ranges_sequential;
DROP TABLE pending_transactions;
DROP TABLE saved_addresses;
-- token_balances is the only table that was placed by mistake in nodeconfig
-- migrations and in tests it is missing if nodeconfig migration is not used.
DROP TABLE IF EXISTS token_balances;
DROP TABLE tokens;
DROP TABLE visible_tokens;
DROP TABLE currency_format_cache;
DROP TABLE multi_transactions;
DROP TABLE balance_history;
DROP TABLE price_cache;
DROP TABLE collectibles_ownership_cache;
DROP TABLE transfers;

-- All indices are automatically removed
