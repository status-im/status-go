CREATE TABLE torrent_config (
  enabled BOOLEAN DEFAULT false,
  port UNSIGNED INT,
  data_dir VARCHAR NOT NULL,
  torrent_dir VARCHAR NOT NULL,
  synthetic_id VARCHAR DEFAULT 'id' PRIMARY KEY
) WITHOUT ROWID;

