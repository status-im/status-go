ALTER TABLE chats ADD COLUMN unviewed_mentions_count INT DEFAULT 0;
UPDATE chats SET unviewed_mentions_count = 0;
