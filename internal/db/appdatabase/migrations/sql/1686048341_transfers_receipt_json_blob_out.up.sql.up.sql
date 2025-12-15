-- This migration is done in GO code as a custom step.
-- This file serves as an anchor for the migration system
-- Check migrateWalletJsonBlobs from appdatabase/database.go

-- The following steps are done in GO code:

-- ALTER TABLE transfers ADD COLUMN status INT;
-- ALTER TABLE transfers ADD COLUMN receipt_type INT;
-- ALTER TABLE transfers ADD COLUMN tx_hash BLOB;
-- ALTER TABLE transfers ADD COLUMN log_index INT;
-- ALTER TABLE transfers ADD COLUMN block_hash BLOB;
-- ALTER TABLE transfers ADD COLUMN cumulative_gas_used INT;
-- ALTER TABLE transfers ADD COLUMN contract_address TEXT;
-- ALTER TABLE transfers ADD COLUMN gas_used INT;
-- ALTER TABLE transfers ADD COLUMN tx_index INT;

-- ALTER TABLE transfers ADD COLUMN tx_type INT;
-- ALTER TABLE transfers ADD COLUMN protected BOOLEAN;
-- ALTER TABLE transfers ADD COLUMN gas_limit UNSIGNED INT;
-- ALTER TABLE transfers ADD COLUMN gas_price_clamped64 INT;
-- ALTER TABLE transfers ADD COLUMN gas_tip_cap_clamped64 INT;
-- ALTER TABLE transfers ADD COLUMN gas_fee_cap_clamped64 INT;
-- ALTER TABLE transfers ADD COLUMN amount_padded128hex CHAR(32);
-- ALTER TABLE transfers ADD COLUMN account_nonce INT;
-- ALTER TABLE transfers ADD COLUMN size INT;
-- ALTER TABLE transfers ADD COLUMN token_address BLOB;
-- ALTER TABLE transfers ADD COLUMN token_id BLOB;

-- CREATE INDEX transfers_filter ON transfers (status, token_address, token_id);`

-- Extract tx and receipt data from the json blob and add the information into the new columns