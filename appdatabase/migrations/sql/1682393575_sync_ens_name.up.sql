ALTER TABLE settings_sync_clock ADD COLUMN usernames INTEGER NOT NULL DEFAULT 0;

-- we need remove duplicate records since ens.AddEnsUsername(INSERT OR REPLACE INTO ens_usernames) may inserted duplicate records
CREATE TABLE ens_usernames_temp AS SELECT DISTINCT username, chain_id FROM ens_usernames;
DROP TABLE ens_usernames;
ALTER TABLE ens_usernames_temp RENAME TO ens_usernames;

-- we need add unique index to avoid duplicate records, or we can say it will make `INSERT OR REPLACE INTO` work
CREATE UNIQUE INDEX idx_unique_username_chain_id ON ens_usernames (username, chain_id);

ALTER TABLE ens_usernames ADD COLUMN clock INT DEFAULT 0;
ALTER TABLE ens_usernames ADD COLUMN removed BOOLEAN DEFAULT FALSE;
