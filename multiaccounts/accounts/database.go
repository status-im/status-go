package accounts

import (
	"database/sql"
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/status-im/status-go/sqlite"
)

const (
	uniqueChatConstraint   = "UNIQUE constraint failed: accounts.chat"
	uniqueWalletConstraint = "UNIQUE constraint failed: accounts.wallet"
)

var (
	// ErrWalletNotUnique returned if another account has `wallet` field set to true.
	ErrWalletNotUnique = errors.New("another account is set to be default wallet. disable it before using new")
	// ErrChatNotUnique returned if another account has `chat` field set to true.
	ErrChatNotUnique = errors.New("another account is set to be default chat. disable it before using new")
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

func NewDB(db *sql.DB) *Database {
	return &Database{db: db}
}

// Database sql wrapper for operations with browser objects.
type Database struct {
	db *sql.DB
}

// Close closes database.
func (db Database) Close() error {
	return db.db.Close()
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
	pubkey := []byte{}
	for rows.Next() {
		acc := Account{}
		err := rows.Scan(
			&acc.Address, &acc.Wallet, &acc.Chat, &acc.Type, &acc.Storage,
			&pubkey, &acc.Path, &acc.Name, &acc.Color)
		if err != nil {
			return nil, err
		}
		if lth := len(pubkey); lth > 0 {
			acc.PublicKey = make(hexutil.Bytes, lth)
			copy(acc.PublicKey, pubkey)
		}
		accounts = append(accounts, acc)
	}
	return accounts, nil
}

func (db *Database) SaveAccounts(accounts []Account) (err error) {
	var (
		tx     *sql.Tx
		insert *sql.Stmt
		update *sql.Stmt
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
	// NOTE(dshulyak) replace all record values using address (primary key)
	// can't use `insert or replace` because of the additional constraints (wallet and chat)
	insert, err = tx.Prepare("INSERT OR IGNORE INTO accounts (address) VALUES (?)")
	if err != nil {
		return err
	}
	update, err = tx.Prepare("UPDATE accounts SET wallet = ?, chat = ?, type = ?, storage = ?, pubkey = ?, path = ?, name = ?, color = ? WHERE address = ?")
	if err != nil {
		return err
	}
	for i := range accounts {
		acc := &accounts[i]
		_, err = insert.Exec(acc.Address)
		if err != nil {
			return
		}
		_, err = update.Exec(acc.Wallet, acc.Chat, acc.Type, acc.Storage, acc.PublicKey, acc.Path, acc.Name, acc.Color, acc.Address)
		if err != nil {
			switch err.Error() {
			case uniqueChatConstraint:
				err = ErrChatNotUnique
			case uniqueWalletConstraint:
				err = ErrWalletNotUnique
			}
			return
		}
	}
	return
}

func (db *Database) GetWalletAddress() (rst common.Address, err error) {
	err = db.db.QueryRow("SELECT address FROM accounts WHERE wallet = 1").Scan(&rst)
	return
}

func (db *Database) GetChatAddress() (rst common.Address, err error) {
	err = db.db.QueryRow("SELECT address FROM accounts WHERE chat = 1").Scan(&rst)
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
