package settings

import (
	"database/sql"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/status-im/status-go/accountsstore/settings/migrations"
	"github.com/status-im/status-go/sqlite"
)

type Account struct {
	Address   common.Address `json:"address"`
	Wallet    bool           `json:"wallet"`
	Chat      bool           `json:"chat"`
	Type      string         `json:"type"`
	Storage   string         `json:"storage"`
	Path      string         `json:"path"`
	PublicKey hexutil.Bytes  `json:"publicKey"`
	Name      string         `json:"name"`
	Color     string         `json:"color"`
}

// Database sql wrapper for operations with browser objects.
type Database struct {
	db *sql.DB
}

// Close closes database.
func (db Database) Close() error {
	return db.db.Close()
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

func (db *Database) SaveConfig(typ string, value interface{}) error {
	_, err := db.db.Exec("INSERT OR REPLACE INTO settings (type, value) VALUES (?, ?)", typ, &sqlite.JSONBlob{value})
	return err
}

func (db *Database) GetConfig(typ string, value interface{}) error {
	return db.db.QueryRow("SELECT value FROM settings WHERE type = ?", typ).Scan(&sqlite.JSONBlob{value})
}

func (db *Database) GetBlob(typ string) (rst []byte, err error) {
	return rst, db.db.QueryRow("SELECT value FROM settings WHERE type = ?", typ).Scan(&rst)
}

func (db *Database) GetAccounts() ([]Account, error) {
	rows, err := db.db.Query("SELECT address, wallet, chat, type, storage, pubkey, path, name, color FROM accounts")
	if err != nil {
		return nil, err
	}
	accounts := []Account{}
	for rows.Next() {
		acc := Account{}
		err := rows.Scan(
			&acc.Address, &acc.Wallet, &acc.Chat, &acc.Type, &acc.Storage,
			&acc.PublicKey, &acc.Path, &acc.Name, &acc.Color)
		if err != nil {
			return nil, err
		}
		accounts = append(accounts, acc)
	}
	return accounts, nil
}

func (db *Database) SaveAccounts(accounts []Account) (err error) {
	var (
		tx     *sql.Tx
		insert *sql.Stmt
	)
	tx, err = db.db.Begin()
	if err != nil {
		return
	}
	defer func() {
		if err == nil {
			err = tx.Commit()
			return
		}
		_ = tx.Rollback()
	}()
	insert, err = tx.Prepare("INSERT INTO accounts (address, wallet, chat, type, storage, pubkey, path, name, color) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}
	for i := range accounts {
		acc := &accounts[i]
		_, err = insert.Exec(acc.Address, acc.Wallet, acc.Chat, acc.Type, acc.Storage, acc.PublicKey, acc.Path, acc.Name, acc.Color)
		if err != nil {
			return
		}
	}
	return
}

func (db *Database) GetWalletAddress() (rst common.Address, err error) {
	err = db.db.QueryRow("SELECT address FROM accounts WHERE wallet = true").Scan(&rst)
	return
}

func (db *Database) GetChatAddress() (rst common.Address, err error) {
	err = db.db.QueryRow("SELECT address FROM accounts WHERE chat = true").Scan(&rst)
	return
}

func (db *Database) GetAddresses() (rst []common.Address, err error) {
	rows, err := db.db.Query("SELECT address FROM accounts")
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		addr := common.Address{}
		err = rows.Scan(&addr)
		if err != nil {
			return nil, err
		}
		rst = append(rst, addr)
	}
	return rst, nil
}
