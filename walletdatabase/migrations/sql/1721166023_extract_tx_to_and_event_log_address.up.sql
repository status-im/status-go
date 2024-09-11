-- This migration is done in GO code as a custom step.
-- This file serves as an anchor for the migration system
-- Check migrateWalletTransactionToAndEventLogAddress from walletdatabase/database.go

-- The following steps are done in GO code:

-- ALTER TABLE transfers ADD COLUMN transaction_to BLOB;
-- ALTER TABLE transfers ADD COLUMN event_log_address BLOB;

-- Extract the following:
-- 1) Transaction To field (Receiver address for ETH transfers or 
-- address of the contract interacted with)
-- 2) Event Log Address (Address of the contract that emitted the event)
