CREATE TABLE IF NOT EXISTS user_messages (
    id BLOB UNIQUE NOT NULL,
    chat_id VARCHAR NOT NULL,
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

CREATE INDEX chat_ids ON user_messages(chat_id);

CREATE TABLE IF NOT EXISTS membership_updates (
    id VARCHAR PRIMARY KEY NOT NULL,
    data BLOB NOT NULL,
    chat_id VARCHAR NOT NULL,
    FOREIGN KEY (chat_id) REFERENCES chats(id)
) WITHOUT ROWID;

CREATE TABLE IF NOT EXISTS chat_members (
    public_key BLOB NOT NULL,
    chat_id VARCHAR NOT NULL,
    admin BOOLEAN NOT NULL DEFAULT FALSE,
    joined BOOLEAN NOT NULL DEFAULT FALSE,
    FOREIGN KEY (chat_id) REFERENCES chats(id),
    UNIQUE(chat_id, public_key)
);
