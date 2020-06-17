ALTER TABLE user_messages ADD COLUMN audio_payload BLOB;
ALTER TABLE user_messages ADD COLUMN audio_type INT;
ALTER TABLE user_messages ADD COLUMN audio_base64 TEXT NOT NULL DEFAULT "";
