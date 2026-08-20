ALTER TABLE chats ADD COLUMN joined INT DEFAULT 0;
UPDATE chats SET joined = 0 WHERE joined = NULL;

