-- All wallet account positions should be positive, startig from 0.
-- Chat account position will be always -1.
UPDATE
  keypairs_accounts
SET
  position = (SELECT MIN(position) - 1 AS min_pos FROM keypairs_accounts)
WHERE
  chat = TRUE;

CREATE TABLE keypairs_accounts_tmp (
  id         INTEGER PRIMARY KEY AUTOINCREMENT,
  address    VARCHAR,
  key_uid    VARCHAR,
  pubkey     VARCHAR,
  path       VARCHAR  NOT NULL DEFAULT "",
  name       VARCHAR  NOT NULL DEFAULT "",
  color      VARCHAR  NOT NULL DEFAULT "",
  emoji      VARCHAR  NOT NULL DEFAULT "",
  wallet     BOOL     NOT NULL DEFAULT FALSE,
  chat       BOOL     NOT NULL DEFAULT FALSE,
  hidden     BOOL     NOT NULL DEFAULT FALSE,
  operable   VARCHAR  NOT NULL DEFAULT "no",
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL,
  clock      INT      NOT NULL DEFAULT 0,
  position   INT      NOT NULL DEFAULT 0,
  FOREIGN KEY (key_uid) REFERENCES keypairs (key_uid) ON DELETE CASCADE
);

INSERT INTO
  keypairs_accounts_tmp (address, key_uid, pubkey, path, name, color, emoji, wallet, chat,
  hidden, operable, created_at, updated_at, clock, position)
SELECT *
FROM
  keypairs_accounts
ORDER BY
  position;

DELETE FROM keypairs_accounts;

INSERT INTO
  keypairs_accounts
SELECT
  address, key_uid, pubkey, path, name, color, emoji, wallet, chat, hidden, operable,
  created_at, updated_at, clock, id-2 AS pos
FROM
  keypairs_accounts_tmp;

DROP TABLE keypairs_accounts_tmp;

-- we need to keep a clock when accounts reordering was executed
ALTER TABLE settings ADD COLUMN wallet_accounts_position_change_clock INTEGER NOT NULL DEFAULT 0;