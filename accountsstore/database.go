package accountsstore

import (
	"database/sql"

	"github.com/ethereum/go-ethereum/common"
	"github.com/status-im/status-go/accountsstore/migrations"
	"github.com/status-im/status-go/sqlite"
)

type Account struct {
	Name    string
	Address common.Address
}

// InitializeDB creates db file at a given path and applies migrations.
func InitializeDB(path, password string) (*Database, error) {
	db, err := sqlite.OpenDB(path, password)
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

func (db *Database) GetAccounts() ([]Account, error) {
	rows, err := db.db.Query("SELECT address, name from accounts")
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

func (db *Database) SaveAccount(acc Account) error {
	_, err := db.db.Exec("INSERT OR REPLACE INTO accounts (address, name) VALUES (?, ?)", acc.Address, acc.Name)
	return err
}

func (db *Database) DeleteAccount(address common.Address) error {
	_, err := db.db.Exec("DELETE FROM accounts WHERE address = ?", address)
	return err
}

func (db *Database) SaveConfig(address common.Address, typ string, value interface{}) error {
	_, err := db.db.Exec("INSERT OR REPLACE INTO configurations (address, type, value) VALUES (?, ?, ?)", address, typ, &sqlite.JSONBlob{value})
	return err
}

func (db *Database) GetConfig(address common.Address, typ string, value interface{}) error {
	return db.db.QueryRow("SELECT value FROM configurations WHERE address = ? AND type = ?", address, typ).Scan(&sqlite.JSONBlob{value})
}
