ALTER TABLE settings ADD COLUMN bookmarks BLOB;
UPDATE settings SET bookmarks = "[]";