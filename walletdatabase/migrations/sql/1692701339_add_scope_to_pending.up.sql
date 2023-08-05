-- See TxStatus in transactions/pendingtxtracker.go
ALTER TABLE pending_transactions ADD status TEXT NOT NULL DEFAULT "Pending";

-- The watcher will auto delete the pending txs that are confirmed or failed
-- Else the producer module will have to delete them manually on processing
ALTER TABLE pending_transactions ADD COLUMN auto_delete BOOLEAN NOT NULL DEFAULT 1;

DROP INDEX idx_pending_transactions;

CREATE INDEX idx_pending_transactions
ON pending_transactions (multi_transaction_id, from_address, to_address, network_id, timestamp, symbol, type, status, auto_delete);
