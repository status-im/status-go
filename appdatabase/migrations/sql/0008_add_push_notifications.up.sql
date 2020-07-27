ALTER TABLE settings ADD COLUMN remote_push_notifications_enabled BOOLEAN DEFAULT FALSE;
ALTER TABLE settings ADD COLUMN send_push_notifications BOOLEAN DEFAULT TRUE;
ALTER TABLE settings ADD COLUMN push_notifications_server_enabled BOOLEAN DEFAULT FALSE;
ALTER TABLE settings ADD COLUMN push_notifications_from_contacts_only BOOLEAN DEFAULT FALSE;
