ALTER TABLE chats ADD COLUMN read_messages_at_clock_value INT DEFAULT 0;
UPDATE chats SET read_messages_at_clock_value = 0;
CREATE INDEX local_chat_id_seen_mentioned_clock_value_idx ON user_messages(local_chat_id, seen, mentioned, clock_value);
