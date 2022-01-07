ALTER TABLE settings ADD COLUMN messages_from_contacts_only BOOLEAN DEFAULT FALSE;
UPDATE settings SET messages_from_contacts_only = 0;
