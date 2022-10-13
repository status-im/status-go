ALTER TABLE user_messages ADD COLUMN identity_verification_state INT;

CREATE TABLE IF NOT EXISTS verification_requests_individual (
  from_user TEXT,
  to_user TEXT,
  challenge TEXT NOT NULL,
  requested_at INT NOT NULL DEFAULT 0,
  response TEXT,
  replied_at INT NOT NULL DEFAULT 0,
  verification_status INT NOT NULL DEFAULT 0,
  id TEXT PRIMARY KEY ON CONFLICT REPLACE
);
