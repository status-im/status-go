ALTER TABLE user_messages ADD COLUMN image_payload BLOB;
ALTER TABLE user_messages ADD COLUMN image_type INT;
ALTER TABLE user_messages ADD COLUMN image_base64 TEXT NOT NULL DEFAULT "";
