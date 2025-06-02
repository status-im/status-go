ALTER TABLE settings ADD COLUMN messenger_notifications_enabled BOOLEAN DEFAULT FALSE;

UPDATE settings SET messenger_notifications_enabled = notifications_enabled;
