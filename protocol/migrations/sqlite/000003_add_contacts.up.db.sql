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
