ALTER TABLE settings ADD COLUMN current_user_status BLOB;
ALTER TABLE settings ADD COLUMN send_status_updates BOOLEAN DEFAULT TRUE;

