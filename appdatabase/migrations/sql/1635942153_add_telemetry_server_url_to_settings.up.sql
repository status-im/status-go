ALTER TABLE settings ADD COLUMN telemetry_server_url VARCHAR NOT NULL DEFAULT "";
UPDATE settings SET telemetry_server_url = "";