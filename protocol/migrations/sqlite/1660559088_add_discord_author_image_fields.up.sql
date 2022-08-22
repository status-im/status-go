ALTER TABLE discord_message_authors ADD COLUMN avatar_image_payload BLOB;
ALTER TABLE discord_message_authors ADD COLUMN avatar_image_type INT;
ALTER TABLE discord_message_authors ADD COLUMN avatar_image_base64 TEXT NOT NULL DEFAULT "";
