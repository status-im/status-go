ALTER TABLE chats ADD COLUMN first_message_timestamp INT DEFAULT 0;
UPDATE chats SET first_message_timestamp = 0;