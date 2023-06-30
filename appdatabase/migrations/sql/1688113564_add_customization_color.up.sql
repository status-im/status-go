ALTER TABLE settings ADD COLUMN customization_color TEXT NOT NULL DEFAULT "";
UPDATE settings SET customization_color = "";
