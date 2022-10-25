ALTER TABLE activity_center_notifications ADD COLUMN community_id VARCHAR;
ALTER TABLE activity_center_notifications ADD COLUMN membership_status INT NOT NULL DEFAULT 0;