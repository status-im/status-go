ALTER TABLE accounts ADD COLUMN emoji TEXT NOT NULL DEFAULT "";
UPDATE accounts SET emoji = "";
