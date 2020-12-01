/* Copy the accounts table into a temp table, EXCLUDE the `photoPath` and INCLUDE `identicon` column */
CREATE TEMPORARY TABLE accounts_backup(
    keyUid VARCHAR PRIMARY KEY,
    name TEXT NOT NULL,
    loginTimestamp BIG INT,
    identicon TEXT,
    keycardPairing TEXT
) WITHOUT ROWID;
INSERT INTO accounts_backup SELECT keyUid, name, loginTimestamp, photoPath, keycardPairing FROM accounts;

/* Drop the old accounts table and recreate with all columns EXCLUDING `photoPath` and INCLUDING `identicon`*/
DROP TABLE accounts;
CREATE TABLE accounts(
    keyUid VARCHAR PRIMARY KEY,
    name TEXT NOT NULL,
    loginTimestamp BIG INT,
    identicon TEXT,
    keycardPairing TEXT
 ) WITHOUT ROWID;
INSERT INTO accounts SELECT keyUid, name, loginTimestamp, identicon, keycardPairing FROM accounts_backup;

/* Tidy up, drop the temp table */
DROP TABLE accounts_backup;