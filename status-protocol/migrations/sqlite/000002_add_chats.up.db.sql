CREATE TABLE IF NOT EXISTS chats (
id VARCHAR PRIMARY KEY ON CONFLICT REPLACE,
name VARCHAR NOT NULL,
color VARCHAR NOT NULL DEFAULT '#a187d5',
type INT NOT NULL,
active BOOLEAN NOT NULL DEFAULT TRUE,
timestamp INT NOT NULL,
deleted_at_clock_value INT NOT NULL DEFAULT 0,
public_key BLOB,
unviewed_message_count INT NOT NULL DEFAULT 0,
last_clock_value INT NOT NULL DEFAULT 0,
last_message_content_type VARCHAR,
last_message_content VARCHAR,
last_message_timestamp INT,
last_message_clock_value INT,
members BLOB,
membership_updates BLOB
);


DROP TABLE membership_updates;
DROP TABLE chat_members;
