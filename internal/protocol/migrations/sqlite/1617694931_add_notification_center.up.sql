ALTER TABLE chats ADD COLUMN accepted BOOLEAN DEFAULT false;
UPDATE chats SET accepted = 1;

CREATE TABLE activity_center_notifications (
  id VARCHAR NOT NULL PRIMARY KEY,
  timestamp INT NOT NULL,
  notification_type INT NOT NULL,
  chat_id VARCHAR,
  read BOOLEAN NOT NULL DEFAULT FALSE,
  dismissed BOOLEAN NOT NULL DEFAULT FALSE,
  accepted BOOLEAN NOT NULL DEFAULT FALSE
) WITHOUT ROWID;

CREATE INDEX activity_center_dimissed_accepted ON activity_center_notifications(dismissed, accepted);

CREATE INDEX activity_center_read ON activity_center_notifications(read);
