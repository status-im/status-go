CREATE TABLE IF NOT EXISTS emoji_reactions (
  id VARCHAR PRIMARY KEY ON CONFLICT REPLACE,
  clock_value INT NOT NULL,
  source TEXT NOT NULL,
  emoji_id INT NOT NULL,
  message_id VARCHAR NOT NULL,
  chat_id VARCHAR NOT NULL,
  retracted INT DEFAULT 0
);