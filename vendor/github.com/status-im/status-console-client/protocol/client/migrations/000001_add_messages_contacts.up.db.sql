CREATE TABLE IF NOT EXISTS user_messages (
id BLOB UNIQUE NOT NULL,
contact_id VARCHAR NOT NULL,
content_type VARCHAR,
message_type VARCHAR,
text TEXT,
clock BIGINT,
timestamp BIGINT,
content_chat_id TEXT,
content_text TEXT,
public_key BLOB
);

CREATE INDEX contact_ids ON user_messages(contact_id);

CREATE TABLE IF NOT EXISTS user_contacts (
id VARCHAR PRIMARY KEY NOT NULL,
name VARCHAR NOT NULL,
topic TEXT NOT NULL,
type INT NOT NULL,
state INT,
public_key BLOB
) WITHOUT ROWID;

CREATE TABLE IF NOT EXISTS history_user_contact_topic (
synced BIGINT DEFAULT 0 NOT NULL,
contact_id VARCHAR UNIQUE NOT NULL,
FOREIGN KEY(contact_id) REFERENCES user_contacts(id) ON DELETE CASCADE
);
