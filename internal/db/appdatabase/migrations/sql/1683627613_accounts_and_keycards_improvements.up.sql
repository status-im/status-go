-- Possible `operable` column values:
-- "no"        // an account is non operable it is not a keycard account and there is no keystore file for it and no keystore file for the address it is derived from
-- "partially" // an account is partially operable if it is not a keycard account and there is created keystore file for the address it is derived from
-- "fully"     // an account is fully operable if it is not a keycard account and there is a keystore file for it

-- Adding new tables `keypairs` and `keypairs_accounts`
CREATE TABLE IF NOT EXISTS keypairs (
  key_uid VARCHAR PRIMARY KEY NOT NULL CHECK (length(trim(key_uid)) > 0),
  name VARCHAR NOT NULL DEFAULT "",
  type VARCHAR NOT NULL DEFAULT "",
  derived_from VARCHAR NOT NULL DEFAULT "",
  last_used_derivation_index INT NOT NULL DEFAULT 0,
  synced_from VARCHAR NOT NULL DEFAULT "", -- keeps an info which device this keypair is added from
  clock INT NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS keypairs_accounts (
  address VARCHAR PRIMARY KEY,
  key_uid VARCHAR,
  pubkey VARCHAR,
  path VARCHAR NOT NULL DEFAULT "",
  name VARCHAR NOT NULL DEFAULT "",
  color VARCHAR NOT NULL DEFAULT "",
  emoji VARCHAR NOT NULL DEFAULT "",
  wallet BOOL NOT NULL DEFAULT FALSE,
  chat BOOL NOT NULL DEFAULT FALSE,
  hidden BOOL NOT NULL DEFAULT FALSE,
  operable VARCHAR NOT NULL DEFAULT "no", -- describes an account's operability (read an explanation at the top of this file)
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL,
  clock INT NOT NULL DEFAULT 0,
  FOREIGN KEY(key_uid) REFERENCES keypairs(key_uid) 
    ON DELETE CASCADE
);

-- Fulfilling the tables
INSERT INTO keypairs 
  SELECT key_uid, keypair_name, "profile", derived_from, last_used_derivation_index, "", clock 
  FROM accounts 
  WHERE type != "watch" AND type != "seed" AND type != "key"
  GROUP BY key_uid;

INSERT INTO keypairs 
  SELECT key_uid, keypair_name, type, derived_from, last_used_derivation_index, "", clock 
  FROM accounts 
  WHERE type != "watch" AND type != "" AND type != "generated"
  GROUP BY key_uid;

INSERT INTO keypairs_accounts 
  SELECT a.address, kp.key_uid, a.pubkey, a.path, a.name, a.color, a.emoji, a.wallet, a.chat, a.hidden, "fully", a.created_at, a.updated_at, a.clock 
  FROM accounts a 
  LEFT JOIN keypairs kp 
  ON a.key_uid = kp.key_uid;

-- Removing old `accounts` table
DROP TABLE accounts;

-- Add foreign key to `keycards` table
-- There is no other way to add foreign key to `keycards` table except to re-create tables.
ALTER TABLE keycards RENAME TO keycards_old;
ALTER TABLE keycards_accounts RENAME TO keycards_accounts_old;

CREATE TABLE IF NOT EXISTS keycards (
    keycard_uid VARCHAR NOT NULL PRIMARY KEY,
    keycard_name VARCHAR NOT NULL,
    keycard_locked BOOLEAN DEFAULT FALSE,
    key_uid VARCHAR NOT NULL,
    last_update_clock INT NOT NULL DEFAULT 0,
    FOREIGN KEY(key_uid) REFERENCES keypairs(key_uid) 
      ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS keycards_accounts (
    keycard_uid VARCHAR NOT NULL,
    account_address VARCHAR NOT NULL,
    FOREIGN KEY(keycard_uid) REFERENCES keycards(keycard_uid) 
      ON UPDATE CASCADE
      ON DELETE CASCADE
);

INSERT INTO keycards 
  SELECT kc_old.keycard_uid, kc_old.keycard_name, kc_old.keycard_locked, kp.key_uid, kc_old.last_update_clock 
  FROM keycards_old kc_old
  JOIN keypairs kp 
  ON kc_old.key_uid = kp.key_uid;

INSERT INTO keycards_accounts 
  SELECT kc.keycard_uid, kc_acc_old.account_address 
  FROM keycards_accounts_old kc_acc_old
  JOIN keycards kc
  ON kc_acc_old.keycard_uid = kc.keycard_uid;

DROP TABLE keycards_accounts_old;
DROP TABLE keycards_old;