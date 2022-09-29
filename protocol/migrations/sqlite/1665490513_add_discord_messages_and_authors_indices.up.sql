CREATE INDEX idx_dm_author_id_dm_id ON discord_messages (id, author_id);
CREATE INDEX idx_response_to_source_discord_message_id ON user_messages (response_to, source, discord_message_id);
CREATE INDEX idx_pinned_response_to_source_discord_message_id ON pin_messages (message_id, pinned);

