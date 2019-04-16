CREATE TABLE whisper_received_messages (
hash VARCHAR(32) PRIMARY KEY NOT NULL,
enckey TEXT,
body BLOB
);
