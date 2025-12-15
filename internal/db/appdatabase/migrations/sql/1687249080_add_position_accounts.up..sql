CREATE TABLE keypairs_accounts_t1 (
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
    clock      INT      NOT NULL DEFAULT 0
);

INSERT INTO keypairs_accounts_t1(address,key_uid,pubkey,path,name,color,emoji,wallet,chat,hidden,operable,created_at,updated_at,clock) SELECT
   address,
   key_uid,
   pubkey,
   path,
   name,
   color,
   emoji,
   wallet,
   chat,
   hidden,
   operable,
   created_at,
   updated_at,
   clock
FROM keypairs_accounts ORDER BY created_at;

CREATE TABLE keypairs_accounts_t2 (
   address    VARCHAR PRIMARY KEY,
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

INSERT INTO keypairs_accounts_t2 SELECT
   address,
   key_uid,
   pubkey,
   path,
   name,
   color,
   emoji,
   wallet,
   chat,
   hidden,
   operable,
   created_at,
   updated_at,
   clock,
   id
FROM keypairs_accounts_t1;

DROP TABLE keypairs_accounts;
DROP TABLE keypairs_accounts_t1;

ALTER TABLE keypairs_accounts_t2 RENAME TO keypairs_accounts;
