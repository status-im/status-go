package accounts

import (
	"database/sql"
	"encoding/json"

	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/multiaccounts/errors"
	"github.com/status-im/status-go/multiaccounts/settings"
	notificationssettings "github.com/status-im/status-go/multiaccounts/settings_notifications"
	sociallinkssettings "github.com/status-im/status-go/multiaccounts/settings_social_links"
	"github.com/status-im/status-go/nodecfg"
	"github.com/status-im/status-go/params"
)

const (
	uniqueChatConstraint   = "UNIQUE constraint failed: accounts.chat"
	uniqueWalletConstraint = "UNIQUE constraint failed: accounts.wallet"
)

type Account struct {
	Address     types.Address  `json:"address"`
	Wallet      bool           `json:"wallet"`
	Chat        bool           `json:"chat"`
	Type        string         `json:"type,omitempty"`
	Storage     string         `json:"storage,omitempty"`
	Path        string         `json:"path,omitempty"`
	PublicKey   types.HexBytes `json:"public-key,omitempty"`
	Name        string         `json:"name"`
	Emoji       string         `json:"emoji"`
	Color       string         `json:"color"`
	Hidden      bool           `json:"hidden"`
	DerivedFrom string         `json:"derived-from,omitempty"`
	Clock       uint64         `json:"clock,omitempty"`
	Removed     bool           `json:"removed,omitempty"`
}

const (
	AccountTypeGenerated = "generated"
	AccountTypeKey       = "key"
	AccountTypeSeed      = "seed"
	AccountTypeWatch     = "watch"
)

// IsOwnAccount returns true if this is an account we have the private key for
// NOTE: Wallet flag can't be used as it actually indicates that it's the default
// Wallet
func (a *Account) IsOwnAccount() bool {
	return a.Wallet || a.Type == AccountTypeSeed || a.Type == AccountTypeGenerated || a.Type == AccountTypeKey
}

func (a *Account) MarshalJSON() ([]byte, error) {
	item := struct {
		Address          types.Address  `json:"address"`
		MixedcaseAddress string         `json:"mixedcase-address"`
		Wallet           bool           `json:"wallet"`
		Chat             bool           `json:"chat"`
		Type             string         `json:"type,omitempty"`
		Storage          string         `json:"storage,omitempty"`
		Path             string         `json:"path,omitempty"`
		PublicKey        types.HexBytes `json:"public-key,omitempty"`
		Name             string         `json:"name"`
		Emoji            string         `json:"emoji"`
		Color            string         `json:"color"`
		Hidden           bool           `json:"hidden"`
		DerivedFrom      string         `json:"derived-from,omitempty"`
		Clock            uint64         `json:"clock"`
		Removed          bool           `json:"removed"`
	}{
		Address:          a.Address,
		MixedcaseAddress: a.Address.Hex(),
		Wallet:           a.Wallet,
		Chat:             a.Chat,
		Type:             a.Type,
		Storage:          a.Storage,
		Path:             a.Path,
		PublicKey:        a.PublicKey,
		Name:             a.Name,
		Emoji:            a.Emoji,
		Color:            a.Color,
		Hidden:           a.Hidden,
		DerivedFrom:      a.DerivedFrom,
		Clock:            a.Clock,
		Removed:          a.Removed,
	}

	return json.Marshal(item)
}

// Database sql wrapper for operations with browser objects.
type Database struct {
	*settings.Database
	*notificationssettings.NotificationsSettings
	*sociallinkssettings.SocialLinksSettings
	db *sql.DB
}

// NewDB returns a new instance of *Database
func NewDB(db *sql.DB) (*Database, error) {
	sDB, err := settings.MakeNewDB(db)
	if err != nil {
		return nil, err
	}
	sn := notificationssettings.NewNotificationsSettings(db)
	ssl := sociallinkssettings.NewSocialLinksSettings(db)

	return &Database{sDB, sn, ssl, db}, nil
}

// DB Gets db sql.DB
func (db Database) DB() *sql.DB {
	return db.db
}

// Close closes database.
func (db Database) Close() error {
	return db.db.Close()
}

func (db *Database) GetAccounts() ([]*Account, error) {
	rows, err := db.db.Query("SELECT address, wallet, chat, type, storage, pubkey, path, name, emoji, color, hidden, derived_from, clock FROM accounts ORDER BY created_at")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	accounts := []*Account{}
	pubkey := []byte{}
	for rows.Next() {
		acc := &Account{}
		err := rows.Scan(
			&acc.Address, &acc.Wallet, &acc.Chat, &acc.Type, &acc.Storage,
			&pubkey, &acc.Path, &acc.Name, &acc.Emoji, &acc.Color, &acc.Hidden, &acc.DerivedFrom, &acc.Clock)
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
	row := db.db.QueryRow("SELECT address, wallet, chat, type, storage, pubkey, path, name, emoji, color, hidden, derived_from, clock FROM accounts  WHERE address = ? COLLATE NOCASE", address)

	acc := &Account{}
	pubkey := []byte{}
	err = row.Scan(
		&acc.Address, &acc.Wallet, &acc.Chat, &acc.Type, &acc.Storage,
		&pubkey, &acc.Path, &acc.Name, &acc.Emoji, &acc.Color, &acc.Hidden, &acc.DerivedFrom, &acc.Clock)

	if err != nil {
		return nil, err
	}
	acc.PublicKey = pubkey
	return acc, nil
}

func (db *Database) SaveAccounts(accounts []*Account) (err error) {
	var (
		tx     *sql.Tx
		insert *sql.Stmt
		delete *sql.Stmt
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
	delete, err = tx.Prepare("DELETE FROM accounts WHERE address = ?")
	update, err = tx.Prepare("UPDATE accounts SET wallet = ?, chat = ?, type = ?, storage = ?, pubkey = ?, path = ?, name = ?,  emoji = ?, color = ?, hidden = ?, derived_from = ?, updated_at = datetime('now'), clock = ? WHERE address = ?")
	if err != nil {
		return err
	}
	for i := range accounts {
		acc := accounts[i]
		if acc.Removed {
			_, err = delete.Exec(acc.Address)
			if err != nil {
				return
			}
			continue
		}
		_, err = insert.Exec(acc.Address)
		if err != nil {
			return
		}
		_, err = update.Exec(acc.Wallet, acc.Chat, acc.Type, acc.Storage, acc.PublicKey, acc.Path, acc.Name, acc.Emoji, acc.Color, acc.Hidden, acc.DerivedFrom, acc.Clock, acc.Address)
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
	_, err := db.db.Exec("DELETE FROM accounts WHERE type = ? OR type = ?", AccountTypeSeed, AccountTypeKey)
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

// GetPath returns true if account with given address was recently key and doesn't have a key yet
func (db *Database) GetPath(address types.Address) (path string, err error) {
	err = db.db.QueryRow("SELECT path FROM accounts WHERE address = ?", address).Scan(&path)
	return path, err
}

func (db *Database) GetNodeConfig() (*params.NodeConfig, error) {
	return nodecfg.GetNodeConfigFromDB(db.db)
}
