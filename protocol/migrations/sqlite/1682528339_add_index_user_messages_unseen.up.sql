CREATE INDEX
    idx_user_messages_unseen ON user_messages (local_chat_id, clock_value)
WHERE NOT(seen)
