ALTER TABLE raw_messages ADD COLUMN sender BLOB;
ALTER TABLE raw_messages ADD COLUMN community_id BLOB;
ALTER TABLE raw_messages ADD COLUMN resend_type INT DEFAULT 0;
ALTER TABLE raw_messages ADD COLUMN resend_method INT DEFAULT 0;
ALTER TABLE raw_messages ADD COLUMN pubsub_topic VARCHAR DEFAULT '';
ALTER TABLE raw_messages ADD COLUMN hash_ratchet_group_id BLOB;
ALTER TABLE raw_messages ADD COLUMN community_key_ex_msg_type INT DEFAULT 0;

DROP INDEX IF EXISTS idx_resend_automatically;
CREATE INDEX idx_resend_type ON raw_messages(resend_type);
ALTER TABLE raw_messages DROP COLUMN resend_automatically;
