ALTER TABLE settings ADD COLUMN link_preview_request_enabled BOOLEAN DEFAULT TRUE;
ALTER TABLE settings ADD COLUMN link_previews_enabled_sites BLOB;
UPDATE settings SET link_preview_request_enabled = 1;
