CREATE TABLE IF NOT EXISTS keycards (
    keycard_uid VARCHAR NOT NULL PRIMARY KEY,
    keycard_name VARCHAR NOT NULL,
    keycard_locked BOOLEAN DEFAULT FALSE,
    key_uid VARCHAR NOT NULL
);

CREATE TABLE IF NOT EXISTS keycards_accounts (
    keycard_uid VARCHAR NOT NULL,
    account_address VARCHAR NOT NULL,
    FOREIGN KEY(keycard_uid) REFERENCES keycards(keycard_uid) 
      ON UPDATE CASCADE 
      ON DELETE CASCADE    
);

INSERT INTO keycards select distinct keycard_uid, keycard_name, keycard_locked, key_uid from keypairs;

INSERT INTO keycards_accounts select keycard_uid, account_address from keypairs;

DROP TABLE keypairs;