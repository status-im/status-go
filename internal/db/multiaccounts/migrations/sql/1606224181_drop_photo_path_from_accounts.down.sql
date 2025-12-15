/* Copy the accounts table into a temp table, EXCLUDE the `identicon` column and INCLUDE the `photoPath` column */
CREATE TEMPORARY TABLE accounts_backup(
    keyUid VARCHAR PRIMARY KEY,
    name TEXT NOT NULL,
    loginTimestamp BIG INT,
    photoPath TEXT,
    keycardPairing TEXT
) WITHOUT ROWID;
INSERT INTO accounts_backup SELECT keyUid, name, loginTimestamp, identicon, keycardPairing FROM accounts;

/* Drop the old accounts table and recreate with all columns EXCLUDING `identicon` and INCLUDING `photoPath` */
DROP TABLE accounts;
CREATE TABLE IF NOT EXISTS accounts (
    keyUid VARCHAR PRIMARY KEY,
    name TEXT NOT NULL,
    loginTimestamp BIG INT,
    photoPath TEXT,
    keycardPairing TEXT
) WITHOUT ROWID;
INSERT INTO accounts SELECT keyUid, name, loginTimestamp, photoPath, keycardPairing FROM accounts_backup;

/* Tidy up, drop the temp table */
DROP TABLE accounts_backup;