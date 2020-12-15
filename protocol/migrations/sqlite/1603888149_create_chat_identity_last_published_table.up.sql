CREATE TABLE IF NOT EXISTS chat_identity_last_published (
  chat_id VARCHAR NOT NULL PRIMARY KEY ON CONFLICT REPLACE,
  clock_value INT NOT NULL,
  hash BLOB NOT NULL
);

CREATE TABLE IF NOT EXISTS chat_identity_contacts (
  contact_id VARCHAR NOT NULL,
  image_type VARCHAR NOT NULL,
  clock_value INT NOT NULL,
  payload BLOB NOT NULL,
  UNIQUE(contact_id, image_type) ON CONFLICT REPLACE
);
