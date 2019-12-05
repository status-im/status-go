package multiaccounts

import (
	"database/sql"

	"github.com/status-im/status-go/multiaccounts/migrations"
	"github.com/status-im/status-go/sqlite"
)

// Account stores public information about account.
type Account struct {
	Name           string `json:"name"`
	Timestamp      int64  `json:"timestamp"`
	PhotoPath      string `json:"photo-path"`
	KeycardPairing string `json:"keycard-pairing"`
	KeyUID         string `json:"key-uid"`
}

// InitializeDB creates db file at a given path and applies migrations.
func InitializeDB(path string) (*Database, error) {
	db, err := sqlite.OpenUnecryptedDB(path)
	if err != nil {
		return nil, err
	}
	err = migrations.Migrate(db)
	if err != nil {
		return nil, err
	}
	return &Database{db: db}, nil
}

type Database struct {
	db *sql.DB
}

func (db *Database) Close() error {
	return db.db.Close()
}

func (db *Database) GetAccounts() ([]Account, error) {
	rows, err := db.db.Query("SELECT name, loginTimestamp, photoPath, keycardPairing, keyUid from accounts ORDER BY loginTimestamp DESC")
	if err != nil {
		return nil, err
	}
	rst := []Account{}
	inthelper := sql.NullInt64{}
	for rows.Next() {
		acc := Account{}
		err = rows.Scan(&acc.Name, &inthelper, &acc.PhotoPath, &acc.KeycardPairing, &acc.KeyUID)
		if err != nil {
			return nil, err
		}
		acc.Timestamp = inthelper.Int64
		rst = append(rst, acc)
	}
	return rst, nil
}

func (db *Database) SaveAccount(account Account) error {
	_, err := db.db.Exec("INSERT OR REPLACE INTO accounts (name, photoPath, keycardPairing, keyUid) VALUES (?, ?, ?, ?)", account.Name, account.PhotoPath, account.KeycardPairing, account.KeyUID)
	return err
}

func (db *Database) UpdateAccount(account Account) error {
	_, err := db.db.Exec("UPDATE accounts SET name = ?, photoPath = ?, keycardPairing = ? WHERE keyUid = ?", account.Name, account.PhotoPath, account.KeycardPairing, account.KeyUID)
	return err
}

func (db *Database) UpdateAccountTimestamp(keyUID string, loginTimestamp int64) error {
	_, err := db.db.Exec("UPDATE accounts SET loginTimestamp = ? WHERE keyUid = ?", loginTimestamp, keyUID)
	return err
}

func (db *Database) DeleteAccount(keyUID string) error {
	_, err := db.db.Exec("DELETE FROM accounts WHERE keyUid = ?", keyUID)
	return err
}
