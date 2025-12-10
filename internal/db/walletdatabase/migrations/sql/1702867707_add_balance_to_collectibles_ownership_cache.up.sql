ALTER TABLE collectibles_ownership_cache ADD COLUMN balance BLOB NOT NULL DEFAULT x'01';

UPDATE collectibles_ownership_cache SET balance = x'01';

CREATE INDEX IF NOT EXISTS collectibles_ownership_filter_collectible ON collectibles_ownership_cache (chain_id, contract_address, token_id);
