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
  last_message BLOB,
  members BLOB,
  membership_updates BLOB
);

CREATE TABLE contacts (
  id TEXT PRIMARY KEY ON CONFLICT REPLACE,
  address TEXT NOT NULL,
  name TEXT NOT NULL,
  ens_verified BOOLEAN DEFAULT FALSE,
  ens_verified_at INT NOT NULL DEFAULT 0,
  alias TEXT NOT NULL,
  identicon TEXT NOT NULL,
  photo TEXT NOT NULL,
  last_updated INT NOT NULL DEFAULT 0,
  system_tags BLOB,
  device_info BLOB,
  tribute_to_talk TEXT NOT NULL
);

-- It's important that this table has rowid as we rely on it
-- when implementing infinite-scroll.
CREATE TABLE IF NOT EXISTS user_messages (
    id VARCHAR PRIMARY KEY ON CONFLICT REPLACE,
    whisper_timestamp INTEGER NOT NULL,
    source TEXT NOT NULL,
    destination BLOB,
    text VARCHAR NOT NULL,
    content_type INT NOT NULL,
    username VARCHAR,
    timestamp INT NOT NULL,
    chat_id VARCHAR NOT NULL,
    local_chat_id VARCHAR NOT NULL,
    hide BOOLEAN DEFAULT FALSE,
    response_to VARCHAR,
    message_type INT,
    clock_value INT NOT NULL,
    seen BOOLEAN NOT NULL DEFAULT FALSE,
    outgoing_status VARCHAR,
    parsed_text BLOB,
    raw_payload BLOB,
    sticker_pack INT,
    sticker_hash VARCHAR,
    command_id VARCHAR,
    command_value VARCHAR,
    command_address VARCHAR,
    command_from VARCHAR,
    command_contract VARCHAR,
    command_transaction_hash VARCHAR,
    command_signature BLOB,
    command_state INT
);

CREATE INDEX idx_album_id on user_messages(local_chat_id, album_id);
CREATE INDEX idx_source ON user_messages(source);
CREATE INDEX idx_search_by_chat_id ON  user_messages(
    substr('0000000000000000000000000000000000000000000000000000000000000000' || clock_value, -64, 64) || id, chat_id, hide
);

CREATE TABLE IF NOT EXISTS raw_messages (
  id VARCHAR PRIMARY KEY ON CONFLICT REPLACE,
  local_chat_id VARCHAR NOT NULL,
  last_sent INT NOT NULL,
  send_count INT NOT NULL,
  sent BOOLEAN DEFAULT FALSE,
  resend_automatically BOOLEAN DEFAULT FALSE,
  message_type INT,
  recipients BLOB,
  payload BLOB);

CREATE TABLE IF NOT EXISTS messenger_transactions_to_validate (
  message_id VARCHAR,
  command_id VARCHAR NOT NULL,
  transaction_hash VARCHAR PRIMARY KEY,
  retry_count INT,
  first_seen INT,
  signature BLOB NOT NULL,
  to_validate BOOLEAN DEFAULT TRUE,
  public_key BLOB);

CREATE INDEX idx_messenger_transaction_to_validate ON  messenger_transactions_to_validate(to_validate);
