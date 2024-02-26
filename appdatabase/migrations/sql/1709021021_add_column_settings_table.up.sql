ALTER TABLE settings ADD COLUMN last_used_wallet_account_name_index_suggestion INTEGER DEFAULT 1;
UPDATE
  settings
SET
  last_used_wallet_account_name_index_suggestion =
  (
    SELECT
      COUNT(*) AS accs_count
    FROM
      keypairs_accounts
    WHERE
      chat = 0
  );
