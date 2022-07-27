CREATE UNIQUE INDEX idx_search_by_discord_message_id ON user_messages (community_id, discord_message_id);
