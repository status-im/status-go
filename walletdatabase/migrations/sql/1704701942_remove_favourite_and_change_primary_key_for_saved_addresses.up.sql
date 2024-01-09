CREATE TABLE IF NOT EXISTS saved_addresses_new (
  address VARCHAR NOT NULL,
  name VARCHAR NOT NULL DEFAULT "",
  removed BOOLEAN NOT NULL DEFAULT FALSE,
  update_clock INT NOT NULL DEFAULT 0,
  chain_short_names VARCHAR NOT NULL DEFAULT "",
  ens_name VARCHAR NOT NULL DEFAULT "",
  is_test BOOLEAN NOT NULL DEFAULT FALSE,
  created_at INT NOT NULL DEFAULT 0,
  color VARCHAR NOT NULL DEFAULT "primary",
  PRIMARY KEY (address, is_test),
  UNIQUE (name, is_test)
) WITHOUT ROWID;

INSERT OR IGNORE INTO saved_addresses_new
  (
    address,
    name,
    removed,
    update_clock,
    chain_short_names,
    ens_name,
    is_test,
    created_at,
    color
  )
SELECT
  address,
  name,
  removed,
  update_clock,
  chain_short_names,
  ens_name,
  is_test,
  created_at,
  color
FROM
  saved_addresses;

DROP TABLE saved_addresses;

ALTER TABLE saved_addresses_new RENAME TO saved_addresses;