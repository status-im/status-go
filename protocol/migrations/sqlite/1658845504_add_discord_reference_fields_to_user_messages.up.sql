ALTER TABLE user_messages ADD COLUMN discord_message_reference_message_id TEXT DEFAULT "";
ALTER TABLE user_messages ADD COLUMN discord_message_reference_channel_id TEXT DEFAULT "";
ALTER TABLE user_messages ADD COLUMN discord_message_reference_guild_id TEXT DEFAULT "";

