-- Drop any previously created user_messages table.
-- We don't need to stay backward compatible with it
-- because it's not used anywhere except for the console client.
DROP TABLE user_messages;

CREATE TABLE IF NOT EXISTS user_messages (
    id BLOB UNIQUE NOT NULL,
    chat_id VARCHAR NOT NULL REFERENCES chats(id) ON DELETE CASCADE,
    content_type VARCHAR,
    message_type VARCHAR,
    text TEXT,
    clock BIGINT,
    timestamp BIGINT,
    content_chat_id TEXT,
    content_text TEXT,
    public_key BLOB,
    flags INT NOT NULL DEFAULT 0
);
