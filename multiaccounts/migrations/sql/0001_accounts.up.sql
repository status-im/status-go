CREATE TABLE IF NOT EXISTS accounts (
address VARCHAR PRIMARY KEY,
name TEXT NOT NULL,
loginTimestamp BIG INT,
photoPath TEXT,
keycardPairing TEXT,
keycardKeyUid TEXT
) WITHOUT ROWID;
