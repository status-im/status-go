DELETE FROM collectibles_ownership_cache;
DELETE FROM collectibles_ownership_update_timestamps;

CREATE UNIQUE INDEX idx_unique_id_per_address ON collectibles_ownership_cache (chain_id, owner_address, contract_address, token_id);
