package accountsstore

import (
	"database/sql"

	"github.com/ethereum/go-ethereum/common"
	"github.com/status-im/status-go/accountsstore/migrations"
	"github.com/status-im/status-go/sqlite"
)

// Account stores public information about account.
type Account struct {
	Name    string         `json:"name"`
	Address common.Address `json:"address"`
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
	rows, err := db.db.Query("SELECT address, name from accounts ORDER BY loginTimestamp DESC")
	if err != nil {
		return nil, err
	}
	rst := []Account{}
	for rows.Next() {
		acc := Account{}
		err = rows.Scan(&acc.Address, &acc.Name)
		if err != nil {
			return nil, err
		}
		rst = append(rst, acc)
	}
	return rst, nil
}

func (db *Database) SaveAccount(account Account) error {
	_, err := db.db.Exec("INSERT INTO accounts (address, name) VALUES (?, ?)", account.Address, account.Name)
	return err
}

func (db *Database) UpdateAccountTimestamp(address common.Address, loginTimestamp int64) error {
	_, err := db.db.Exec("UPDATE accounts SET loginTimestamp = ? WHERE address = ?", loginTimestamp, address)
	return err
}

func (db *Database) DeleteAccount(address common.Address) error {
	_, err := db.db.Exec("DELETE FROM accounts WHERE address = ?", address)
	return err
}
