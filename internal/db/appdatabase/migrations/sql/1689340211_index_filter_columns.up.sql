-- Enhance idx_transfers_filter by including more fields
DROP INDEX IF EXISTS idx_transfers_filter;
CREATE INDEX idx_transfers_filter
ON transfers (multi_transaction_id, loaded, timestamp, status, network_id, tx_from_address, tx_to_address, token_address, token_id, type);

-- Index for pending_transactions
CREATE INDEX idx_pending_transactions
ON pending_transactions (multi_transaction_id, from_address, to_address, network_id, timestamp, symbol);

-- Index for multi_transactions
CREATE INDEX idx_multi_transactions
ON multi_transactions (from_address, to_address, type, from_asset, timestamp, to_asset, from_amount, to_amount);
