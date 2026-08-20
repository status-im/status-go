ALTER TABLE user_messages ADD COLUMN contact_verification_status INT;
ALTER TABLE activity_center_notifications ADD COLUMN contact_verification_status INT DEFAULT 0;

DROP TABLE verification_requests;

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
