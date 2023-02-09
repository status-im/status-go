CREATE TABLE IF NOT EXISTS community_permissions (
    community_id TEXT NOT NULL,
    permission_id VARCHAR NOT NULL PRIMARY KEY,
    permission_private BOOLEAN NOT NULL DEFAULT FALSE,
    is_allowed_to INT NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS community_permissions_chats {
    permission_id VARCHAR NOT NULL,
    chat_id VARCHAR NOT NULL
};