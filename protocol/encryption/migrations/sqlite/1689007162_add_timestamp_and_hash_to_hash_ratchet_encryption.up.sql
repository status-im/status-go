-- hash_ratchet_encryption updates
-- TODO change primary key of hash_ratchet_encryption
--  from PRIMARY KEY(group_id, key_id) to PRIMARY KEY(group_id, hash_id)

ALTER TABLE hash_ratchet_encryption ADD created_at TIMESTAMP NOT NULL;
ALTER TABLE hash_ratchet_encryption ADD hash_id BLOB NOT NULL;
DROP INDEX idx_hash_ratchet_enc;
CREATE UNIQUE INDEX idx_hash_ratchet_enc ON hash_ratchet_encryption(group_id, hash_id);

-- hash_ratchet_encryption_cache updates
-- TODO add new foreign key binding `hash_id`, see https://www.sqlite.org/foreignkeys.html#fk_schemacommands

ALTER TABLE hash_ratchet_encryption_cache ADD hash_id BLOB NOT NULL;
DROP INDEX idx_hash_ratchet_enc_cache;
CREATE UNIQUE INDEX idx_hash_ratchet_enc_cache ON hash_ratchet_encryption_cache(group_id, hash_id);
