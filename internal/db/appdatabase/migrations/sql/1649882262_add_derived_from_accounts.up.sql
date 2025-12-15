ALTER TABLE accounts ADD COLUMN derived_from TEXT NOT NULL DEFAULT "";
UPDATE accounts SET derived_from = "";
