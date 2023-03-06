CREATE TABLE IF NOT EXISTS saved_addresses_new (
  address VARCHAR NOT NULL,
  name TEXT NOT NULL,
  favourite BOOLEAN NOT NULL DEFAULT FALSE,
  removed BOOLEAN NOT NULL DEFAULT FALSE,
  update_clock INT NOT NULL DEFAULT 0,
  chain_short_names VARCHAR DEFAULT "",
  ens_name VARCHAR DEFAULT "",
  is_test BOOLEAN DEFAULT FALSE,
  PRIMARY KEY (address, ens_name, is_test)
) WITHOUT ROWID;

INSERT INTO saved_addresses_new (address, name, favourite, removed, update_clock)
   SELECT address, name, favourite, removed, update_clock FROM saved_addresses;
DROP TABLE saved_addresses;
ALTER TABLE saved_addresses_new RENAME TO saved_addresses;
