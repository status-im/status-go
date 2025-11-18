-- All community tokens have 18 decimals by default and no way to have any other value (detected by the community creation contract)
UPDATE community_tokens SET decimals = 18 WHERE type = 1;