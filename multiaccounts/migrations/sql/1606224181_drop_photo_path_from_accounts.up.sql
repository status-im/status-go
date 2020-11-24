/* Copy the accounts table into a temp table, EXCLUDE the `photoPath` column */
CREATE TEMPORARY TABLE accounts_backup(
    keyUid VARCHAR PRIMARY KEY,
    name TEXT NOT NULL,
    loginTimestamp BIG INT,
    keycardPairing TEXT
) WITHOUT ROWID;
INSERT INTO accounts_backup SELECT keyUid, name, loginTimestamp, keycardPairing FROM accounts;

/* Drop the old accounts table and recreate with all columns EXCLUDING `photoPath` */
DROP TABLE accounts;
CREATE TABLE accounts(keyUid, name, loginTimestamp, keycardPairing);
INSERT INTO accounts SELECT keyUid, name, loginTimestamp, keycardPairing FROM accounts_backup;

/* Tidy up, drop the temp table */
DROP TABLE accounts_backup;