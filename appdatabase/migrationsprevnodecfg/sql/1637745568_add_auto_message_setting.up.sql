ALTER TABLE settings ADD COLUMN auto_message_enabled BOOLEAN DEFAULT FALSE;
UPDATE settings SET auto_message_enabled = 0;
