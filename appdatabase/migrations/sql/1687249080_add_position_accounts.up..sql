ALTER TABLE keypairs_accounts ADD COLUMN position INT NOT NULL DEFAULT 0;

UPDATE keypairs_accounts AS ka SET position = ka2.rowNumber FROM (
  SELECT created_at, ROW_NUMBER() OVER (ORDER BY created_at) AS rowNumber FROM keypairs_accounts) AS ka2
WHERE ka.created_at = ka2.created_at;