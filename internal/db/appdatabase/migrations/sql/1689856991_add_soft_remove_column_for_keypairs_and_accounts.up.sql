ALTER TABLE keypairs ADD COLUMN removed BOOLEAN DEFAULT FALSE;
ALTER TABLE keypairs_accounts ADD COLUMN removed BOOLEAN DEFAULT FALSE;

UPDATE
  settings
SET
  wallet_accounts_position_change_clock = (SELECT MAX(clock) AS max_clock FROM keypairs_accounts)
WHERE
  synthetic_id = 'id';