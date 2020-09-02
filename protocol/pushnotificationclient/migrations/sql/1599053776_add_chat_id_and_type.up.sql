ALTER TABLE push_notification_client_sent_notifications ADD COLUMN chat_id TEXT;
ALTER TABLE push_notification_client_sent_notifications ADD COLUMN notification_type INT;

UPDATE push_notification_client_sent_notifications SET chat_id = "", notification_type = 1;
