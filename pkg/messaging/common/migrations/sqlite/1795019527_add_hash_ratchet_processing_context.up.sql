ALTER TABLE hash_ratchet_encrypted_messages ADD COLUMN from_archive BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE hash_ratchet_encrypted_messages ADD COLUMN store_waku_messages BOOLEAN NOT NULL DEFAULT FALSE;
