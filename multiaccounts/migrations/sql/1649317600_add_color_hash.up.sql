ALTER TABLE accounts ADD COLUMN colorHash TEXT NOT NULL DEFAULT "";
ALTER TABLE accounts ADD COLUMN colorId INT NOT NULL DEFAULT 0;
UPDATE accounts SET colorHash = "";
UPDATE accounts SET colorId = 0;
