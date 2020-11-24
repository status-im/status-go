package multiaccounts

import (
	"context"
	"database/sql"

	"github.com/status-im/status-go/images"
	"github.com/status-im/status-go/multiaccounts/migrations"
	"github.com/status-im/status-go/sqlite"
)

// Account stores public information about account.
type Account struct {
	Name           string   `json:"name"`
	Timestamp      int64    `json:"timestamp"`
	KeycardPairing string   `json:"keycard-pairing"`
	KeyUID         string   `json:"key-uid"`
	ImageURIs      []string `json:"image-uris"`
}

type Database struct {
	db *sql.DB
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

func (db *Database) Close() error {
	return db.db.Close()
}

func (db *Database) GetAccounts() ([]Account, error) {
	rows, err := db.db.Query("SELECT name, loginTimestamp, keycardPairing, keyUid from accounts ORDER BY loginTimestamp DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var rst []Account
	inthelper := sql.NullInt64{}
	for rows.Next() {
		acc := Account{}
		err = rows.Scan(&acc.Name, &inthelper, &acc.KeycardPairing, &acc.KeyUID)
		if err != nil {
			return nil, err
		}
		acc.Timestamp = inthelper.Int64
		rst = append(rst, acc)
	}
	return rst, nil
}

func (db *Database) SaveAccount(account Account) error {
	_, err := db.db.Exec("INSERT OR REPLACE INTO accounts (name, keycardPairing, keyUid) VALUES (?, ?, ?)", account.Name, account.KeycardPairing, account.KeyUID)
	return err
}

func (db *Database) UpdateAccount(account Account) error {
	_, err := db.db.Exec("UPDATE accounts SET name = ?, keycardPairing = ? WHERE keyUid = ?", account.Name, account.KeycardPairing, account.KeyUID)
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

// Account images

func (db *Database) GetIdentityImages(keyUid string) ([]*images.IdentityImage, error) {
	rows, err := db.db.Query(`SELECT key_uid, name, image_payload, width, height, file_size, resize_target FROM identity_images WHERE key_uid = ?`, keyUid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var iis []*images.IdentityImage
	for rows.Next() {
		ii := &images.IdentityImage{}
		err = rows.Scan(&ii.KeyUID, &ii.Name, &ii.Payload, &ii.Width, &ii.Height, &ii.FileSize, &ii.ResizeTarget)
		if err != nil {
			return nil, err
		}

		iis = append(iis, ii)
	}

	return iis, nil
}

func (db *Database) GetIdentityImage(keyUid, it string) (*images.IdentityImage, error) {
	rows, err := db.db.Query("SELECT key_uid, name, image_payload, width, height, file_size, resize_target FROM identity_images WHERE key_uid = ? AND name = ?", keyUid, it)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ii images.IdentityImage
	for rows.Next() {
		err = rows.Scan(&ii.KeyUID ,&ii.Name, &ii.Payload, &ii.Width, &ii.Height, &ii.FileSize, &ii.ResizeTarget)
		if err != nil {
			return nil, err
		}
	}

	return &ii, nil
}

func (db *Database) StoreIdentityImages(keyUid string, iis []*images.IdentityImage) error {
	// Because SQL INSERTs are triggered in a loop use a tx to ensure a single call to the DB.
	tx, err := db.db.BeginTx(context.Background(), &sql.TxOptions{})
	if err != nil {
		return err
	}
	defer func() {
		if err == nil {
			err = tx.Commit()
			return
		}
		// don't shadow original error
		_ = tx.Rollback()
	}()

	for _, ii := range iis {
		if ii == nil {
			continue
		}

		_, err := tx.Exec(
			"INSERT INTO identity_images (key_uid, name, image_payload, width, height, file_size, resize_target) VALUES (?, ?, ?, ?, ?, ?, ?)",
			keyUid,
			ii.Name,
			ii.Payload,
			ii.Width,
			ii.Height,
			ii.FileSize,
			ii.ResizeTarget,
		)
		if err != nil {
			return err
		}
	}

	return nil
}

func (db *Database) DeleteIdentityImage(keyUid string) error {
	_, err := db.db.Exec(`DELETE FROM identity_images WHERE key_uid = ?`, keyUid)
	return err
}
