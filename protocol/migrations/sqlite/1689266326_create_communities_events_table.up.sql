CREATE TABLE IF NOT EXISTS communities_events (
  id BLOB NOT NULL PRIMARY KEY ON CONFLICT REPLACE,
  raw_events BLOB NOT NULL,
  raw_description BLOB NOT NULL
  );