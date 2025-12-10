ALTER TABLE settings ADD COLUMN use_mailservers BOOLEAN DEFAULT TRUE;
UPDATE settings SET use_mailservers = 1;
