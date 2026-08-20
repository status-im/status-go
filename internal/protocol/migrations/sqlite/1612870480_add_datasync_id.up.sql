ALTER TABLE raw_messages ADD COLUMN datasync_id BLOB;
CREATE INDEX idx_datsync_id ON raw_messages(datasync_id);