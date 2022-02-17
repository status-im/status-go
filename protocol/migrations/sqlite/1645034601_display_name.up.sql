ALTER TABLE contacts ADD COLUMN display_name TEXT NOT NULL DEFAULT "";
UPDATE contacts SET display_name = "";
