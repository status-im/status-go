ALTER TABLE keycards RENAME TO keycards_old;
ALTER TABLE keycards_accounts RENAME TO keycards_accounts_old;

CREATE TABLE IF NOT EXISTS keycards (
    keycard_uid VARCHAR NOT NULL PRIMARY KEY,
    keycard_name VARCHAR NOT NULL,
    keycard_locked BOOLEAN DEFAULT FALSE,
    key_uid VARCHAR NOT NULL,
    position INT NOT NULL DEFAULT 0,
    FOREIGN KEY(key_uid) REFERENCES keypairs(key_uid)
      ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS keycards_accounts (
    keycard_uid VARCHAR NOT NULL,
    account_address VARCHAR NOT NULL,
    PRIMARY KEY (keycard_uid, account_address),
    FOREIGN KEY(keycard_uid) REFERENCES keycards(keycard_uid)
      ON UPDATE CASCADE
      ON DELETE CASCADE
);

INSERT INTO keycards
  SELECT keycard_uid, keycard_name, keycard_locked, key_uid, last_update_clock
  FROM keycards_old
  ORDER BY last_update_clock;

INSERT INTO keycards_accounts
  SELECT keycard_uid, account_address
  FROM keycards_accounts_old;

UPDATE keycards SET position = rowid - 1;

DROP TABLE keycards_accounts_old;
DROP TABLE keycards_old;