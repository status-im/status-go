-- This migration is done in GO code as a custom step.
-- This file serves as an anchor for the migration system
-- Check migrateWalletTransferFromToAddresses from appdatabase/database.go

-- The following steps are done in GO code:

-- ALTER TABLE transfers ADD COLUMN tx_from_address BLOB;
-- ALTER TABLE transfers ADD COLUMN tx_to_address BLOB;

-- Extract transfer from/to addresses and add the information into the new columns
-- Re-extract token address and insert it as blob instead of string