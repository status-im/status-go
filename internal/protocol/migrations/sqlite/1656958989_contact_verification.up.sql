ALTER TABLE contacts ADD COLUMN verification_status INT DEFAULT 0;

UPDATE contacts SET verification_status = 0;  

CREATE TABLE IF NOT EXISTS trusted_users (
  id TEXT PRIMARY KEY ON CONFLICT REPLACE,
  trust_status INT NOT NULL DEFAULT 0,
  updated_at INT NOT NULL DEFAULT 0
) WITHOUT ROWID;

CREATE TABLE IF NOT EXISTS verification_requests (
  from_user TEXT,
  to_user TEXT,
  challenge TEXT NOT NULL,
  requested_at INT NOT NULL DEFAULT 0,
  response TEXT,
  replied_at INT NOT NULL DEFAULT 0,
  verification_status INT NOT NULL DEFAULT 0,
  CONSTRAINT fromto_unique UNIQUE (from_user, to_user) ON CONFLICT REPLACE
);
