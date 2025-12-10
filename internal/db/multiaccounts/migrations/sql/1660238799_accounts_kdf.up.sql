ALTER TABLE accounts ADD COLUMN kdfIterations INT NOT NULL DEFAULT 3200;
UPDATE accounts SET kdfIterations = 3200;
