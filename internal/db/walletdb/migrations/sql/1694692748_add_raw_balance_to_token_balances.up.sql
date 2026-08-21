-- Add column for storing raw balance expressed in base unints as big integer decimal
ALTER TABLE token_balances ADD COLUMN raw_balance VARCHAR NOT NULL DEFAULT "0"
