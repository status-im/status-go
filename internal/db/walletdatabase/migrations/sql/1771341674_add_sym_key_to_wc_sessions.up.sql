-- Add sym_key column to connector_wc_sessions for session restoration after reconnect
ALTER TABLE connector_wc_sessions ADD COLUMN sym_key TEXT NOT NULL DEFAULT '';
