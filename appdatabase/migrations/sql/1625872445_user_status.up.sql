ALTER TABLE settings ADD COLUMN current_user_status BLOB;
ALTER TABLE settings ADD COLUMN send_status_updates BOOLEAN DEFAULT TRUE;
UPDATE settings SET send_status_updates = 1;
CREATE TABLE status_updates (
  public_key TEXT PRIMARY KEY ON CONFLICT REPLACE,
  status_type INT NOT NULL DEFAULT 0,
  clock INT NOT NULL,
  custom_text TEXT DEFAULT ""
);
