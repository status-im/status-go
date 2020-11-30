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
	Name           string                 `json:"name"`
	Timestamp      int64                  `json:"timestamp"`
	KeycardPairing string                 `json:"keycard-pairing"`
	KeyUID         string                 `json:"key-uid"`
	Images         []images.IdentityImage `json:"images"`
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
	rows, err := db.db.Query("SELECT  a.name, a.loginTimestamp, a.keycardPairing, a.keyUid, ii.name, ii.image_payload, ii.width, ii.height, ii.file_size, ii.resize_target FROM accounts AS a LEFT JOIN identity_images AS ii ON ii.key_uid = a.keyUid ORDER BY loginTimestamp DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rst []Account
	accs := map[string]*Account{}
	accLoginTimestamp := sql.NullInt64{}
	for rows.Next() {
		acc := Account{}
		ii := &images.IdentityImage{}
		iiName := sql.NullString{}
		iiWidth := sql.NullInt64{}
		iiHeight := sql.NullInt64{}
		iiFileSize := sql.NullInt64{}
		iiResizeTarget := sql.NullInt64{}

		err = rows.Scan(
			&acc.Name,
			&accLoginTimestamp,
			&acc.KeycardPairing,
			&acc.KeyUID,
			&iiName,
			&ii.Payload,
			&iiWidth,
			&iiHeight,
			&iiFileSize,
			&iiResizeTarget,
		)
		if err != nil {
			return nil, err
		}

		acc.Timestamp = accLoginTimestamp.Int64

		ii.KeyUID = acc.KeyUID
		ii.Name = iiName.String
		ii.Width = int(iiWidth.Int64)
		ii.Height = int(iiHeight.Int64)
		ii.FileSize = int(iiFileSize.Int64)
		ii.ResizeTarget = int(iiResizeTarget.Int64)

		if ii.Name == "" && len(ii.Payload) == 0 && ii.Width == 0 && ii.Height == 0 && ii.FileSize == 0 && ii.ResizeTarget == 0 {
			ii = nil
		}

		if ii != nil {
			a, ok := accs[acc.Name]
			if ok {
				a.Images = append(a.Images, *ii)
			} else {
				acc.Images = append(acc.Images, *ii)
			}
		}

		accs[acc.Name] = &acc
	}

	// Yes, I know, I'm converting a map into a slice, this is to maintain the function signature and API behaviour and
	// not need to loop through the slice searching for an account with a given keyUID
	for _, a := range accs {
		rst = append(rst, *a)
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

func (db *Database) GetIdentityImages(keyUID string) ([]*images.IdentityImage, error) {
	rows, err := db.db.Query(`SELECT key_uid, name, image_payload, width, height, file_size, resize_target FROM identity_images WHERE key_uid = ?`, keyUID)
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

func (db *Database) GetIdentityImage(keyUID, it string) (*images.IdentityImage, error) {
	rows, err := db.db.Query("SELECT key_uid, name, image_payload, width, height, file_size, resize_target FROM identity_images WHERE key_uid = ? AND name = ?", keyUID, it)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ii images.IdentityImage
	for rows.Next() {
		err = rows.Scan(&ii.KeyUID, &ii.Name, &ii.Payload, &ii.Width, &ii.Height, &ii.FileSize, &ii.ResizeTarget)
		if err != nil {
			return nil, err
		}
	}

	return &ii, nil
}

func (db *Database) StoreIdentityImages(keyUID string, iis []*images.IdentityImage) error {
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

		ii.KeyUID = keyUID
		_, err := tx.Exec(
			"INSERT INTO identity_images (key_uid, name, image_payload, width, height, file_size, resize_target) VALUES (?, ?, ?, ?, ?, ?, ?)",
			ii.KeyUID,
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

func (db *Database) DeleteIdentityImage(keyUID string) error {
	_, err := db.db.Exec(`DELETE FROM identity_images WHERE key_uid = ?`, keyUID)
	return err
}
