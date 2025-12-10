ALTER TABLE multi_transactions ADD COLUMN from_network_id UNSIGNED BIGINT;
ALTER TABLE multi_transactions ADD COLUMN to_network_id UNSIGNED BIGINT;
ALTER TABLE multi_transactions ADD COLUMN cross_tx_id VARCHAR DEFAULT "";
ALTER TABLE multi_transactions ADD COLUMN from_tx_hash BLOB;
ALTER TABLE multi_transactions ADD COLUMN to_tx_hash BLOB;
