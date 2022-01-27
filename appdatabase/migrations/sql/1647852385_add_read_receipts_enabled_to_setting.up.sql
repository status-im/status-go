ALTER TABLE settings ADD COLUMN read_receipts_enabled BOOLEAN DEFAULT false;
UPDATE settings SET read_receipts_enabled = 0;