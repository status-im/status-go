ALTER TABLE activity_center_notifications
  ADD COLUMN deleted BOOLEAN NOT NULL DEFAULT FALSE;

CREATE INDEX activity_center_chat_id_deleted_dismissed_accepted_idx
  ON activity_center_notifications(chat_id, deleted, dismissed, accepted);

CREATE INDEX activity_center_deleted_dismissed_accepted_author_idx
  ON activity_center_notifications(author, deleted, dismissed, accepted);
