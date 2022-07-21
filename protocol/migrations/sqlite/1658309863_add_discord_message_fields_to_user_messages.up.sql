ALTER TABLE user_messages ADD COLUMN discord_message_id TEXT DEFAULT "";
ALTER TABLE user_messages ADD COLUMN discord_message_type VARCHAR DEFAULT "";
ALTER TABLE user_messages ADD COLUMN discord_message_timestamp INT;
ALTER TABLE user_messages ADD COLUMN discord_message_timestamp_edited INT;
ALTER TABLE user_messages ADD COLUMN discord_message_content TEXT DEFAULT "";
ALTER TABLE user_messages ADD COLUMN discord_message_author_id TEXT DEFAULT "";

