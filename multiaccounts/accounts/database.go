package accounts

import (
	"database/sql"

	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/multiaccounts/errors"
	"github.com/status-im/status-go/multiaccounts/settings"
	"github.com/status-im/status-go/nodecfg"
	"github.com/status-im/status-go/params"
)

const (
	uniqueChatConstraint   = "UNIQUE constraint failed: accounts.chat"
	uniqueWalletConstraint = "UNIQUE constraint failed: accounts.wallet"
)

type Account struct {
	Address   types.Address  `json:"address"`
	Wallet    bool           `json:"wallet"`
	Chat      bool           `json:"chat"`
	Type      string         `json:"type,omitempty"`
	Storage   string         `json:"storage,omitempty"`
	Path      string         `json:"path,omitempty"`
	PublicKey types.HexBytes `json:"public-key,omitempty"`
	Name      string         `json:"name"`
	Emoji     string         `json:"emoji"`
	Color     string         `json:"color"`
	Hidden    bool           `json:"hidden"`
}

const (
	accountTypeGenerated = "generated"
	accountTypeKey       = "key"
	accountTypeSeed      = "seed"
	accountTypeWatch     = "watch"
)

// IsOwnAccount returns true if this is an account we have the private key for
// NOTE: Wallet flag can't be used as it actually indicates that it's the default
// Wallet
func (a *Account) IsOwnAccount() bool {
	return a.Wallet || a.Type == accountTypeSeed || a.Type == accountTypeGenerated || a.Type == accountTypeKey
}

// Database sql wrapper for operations with browser objects.
type Database struct {
	*settings.Database
	db *sql.DB
}

// NewDB returns a new instance of *Database
func NewDB(db *sql.DB) (*Database, error) {
	sDB, err := settings.MakeNewDB(db)
	if err != nil {
		return nil, err
	}

	return &Database{sDB, db}, nil
}

// DB Gets db sql.DB
func (db Database) DB() *sql.DB {
	return db.db
}

// Close closes database.
func (db Database) Close() error {
	return db.db.Close()
}

func (db *Database) GetAccounts() ([]Account, error) {
	rows, err := db.db.Query("SELECT address, wallet, chat, type, storage, pubkey, path, name, emoji, color, hidden FROM accounts ORDER BY created_at")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	accounts := []Account{}
	pubkey := []byte{}
	for rows.Next() {
		acc := Account{}
		err := rows.Scan(
			&acc.Address, &acc.Wallet, &acc.Chat, &acc.Type, &acc.Storage,
			&pubkey, &acc.Path, &acc.Name, &acc.Emoji, &acc.Color, &acc.Hidden)
		if err != nil {
			return nil, err
		}
		if lth := len(pubkey); lth > 0 {
			acc.PublicKey = make(types.HexBytes, lth)
			copy(acc.PublicKey, pubkey)
		}
		accounts = append(accounts, acc)
	}
	return accounts, nil
}

func (db *Database) GetAccountByAddress(address types.Address) (rst *Account, err error) {
	row := db.db.QueryRow("SELECT address, wallet, chat, type, storage, pubkey, path, name, emoji, color, hidden FROM accounts  WHERE address = ? COLLATE NOCASE", address)

	acc := &Account{}
	pubkey := []byte{}
	err = row.Scan(
		&acc.Address, &acc.Wallet, &acc.Chat, &acc.Type, &acc.Storage,
		&pubkey, &acc.Path, &acc.Name, &acc.Emoji, &acc.Color, &acc.Hidden)

	if err != nil {
		return nil, err
	}
	acc.PublicKey = pubkey
	return acc, nil
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
	insert, err = tx.Prepare("INSERT OR IGNORE INTO accounts (address, created_at, updated_at) VALUES (?, datetime('now'), datetime('now'))")
	if err != nil {
		return err
	}
	update, err = tx.Prepare("UPDATE accounts SET wallet = ?, chat = ?, type = ?, storage = ?, pubkey = ?, path = ?, name = ?,  emoji = ?, color = ?, hidden = ?, updated_at = datetime('now') WHERE address = ?")
	if err != nil {
		return err
	}
	for i := range accounts {
		acc := &accounts[i]
		_, err = insert.Exec(acc.Address)
		if err != nil {
			return
		}
		_, err = update.Exec(acc.Wallet, acc.Chat, acc.Type, acc.Storage, acc.PublicKey, acc.Path, acc.Name, acc.Emoji, acc.Color, acc.Hidden, acc.Address)
		if err != nil {
			switch err.Error() {
			case uniqueChatConstraint:
				err = errors.ErrChatNotUnique
			case uniqueWalletConstraint:
				err = errors.ErrWalletNotUnique
			}
			return
		}
	}
	return
}

func (db *Database) DeleteAccount(address types.Address) error {
	_, err := db.db.Exec("DELETE FROM accounts WHERE address = ?", address)
	return err
}

func (db *Database) DeleteSeedAndKeyAccounts() error {
	_, err := db.db.Exec("DELETE FROM accounts WHERE type = ? OR type = ?", accountTypeSeed, accountTypeKey)
	return err
}

func (db *Database) GetWalletAddress() (rst types.Address, err error) {
	err = db.db.QueryRow("SELECT address FROM accounts WHERE wallet = 1").Scan(&rst)
	return
}

func (db *Database) GetWalletAddresses() (rst []types.Address, err error) {
	rows, err := db.db.Query("SELECT address FROM accounts WHERE chat = 0 ORDER BY created_at")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		addr := types.Address{}
		err = rows.Scan(&addr)
		if err != nil {
			return nil, err
		}
		rst = append(rst, addr)
	}
	return rst, nil
}

func (db *Database) GetChatAddress() (rst types.Address, err error) {
	err = db.db.QueryRow("SELECT address FROM accounts WHERE chat = 1").Scan(&rst)
	return
}

func (db *Database) GetAddresses() (rst []types.Address, err error) {
	rows, err := db.db.Query("SELECT address FROM accounts ORDER BY created_at")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		addr := types.Address{}
		err = rows.Scan(&addr)
		if err != nil {
			return nil, err
		}
		rst = append(rst, addr)
	}
	return rst, nil
}

// AddressExists returns true if given address is stored in database.
func (db *Database) AddressExists(address types.Address) (exists bool, err error) {
	err = db.db.QueryRow("SELECT EXISTS (SELECT 1 FROM accounts WHERE address = ?)", address).Scan(&exists)
	return exists, err
}

func (db *Database) GetNodeConfig() (*params.NodeConfig, error) {
	return nodecfg.GetNodeConfig(db.db)
}
