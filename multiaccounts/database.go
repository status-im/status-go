package multiaccounts

import (
	"context"
	"database/sql"

	"github.com/ethereum/go-ethereum/log"
	"github.com/status-im/status-go/images"
	"github.com/status-im/status-go/multiaccounts/migrations"
	"github.com/status-im/status-go/sqlite"
)

// Account stores public information about account.
type Account struct {
	Name           string                 `json:"name"`
	Timestamp      int64                  `json:"timestamp"`
	Identicon      string                 `json:"identicon"`
	KeycardPairing string                 `json:"keycard-pairing"`
	KeyUID         string                 `json:"key-uid"`
	Images         []images.IdentityImage `json:"images"`
}

type MultiAccountMarshaller interface {
	ToMultiAccount() *Account
}

type Database struct {
	db                         *sql.DB
	identityImageSubscriptions []chan struct{}
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

func (db *Database) GetAccounts() (rst []Account, err error) {
	rows, err := db.db.Query("SELECT  a.name, a.loginTimestamp, a.identicon, a.keycardPairing, a.keyUid, ii.name, ii.image_payload, ii.width, ii.height, ii.file_size, ii.resize_target FROM accounts AS a LEFT JOIN identity_images AS ii ON ii.key_uid = a.keyUid ORDER BY loginTimestamp DESC")
	if err != nil {
		return nil, err
	}
	defer func() {
		err = rows.Close()
	}()

	for rows.Next() {
		acc := Account{}
		accLoginTimestamp := sql.NullInt64{}
		accIdenticon := sql.NullString{}
		ii := &images.IdentityImage{}
		iiName := sql.NullString{}
		iiWidth := sql.NullInt64{}
		iiHeight := sql.NullInt64{}
		iiFileSize := sql.NullInt64{}
		iiResizeTarget := sql.NullInt64{}

		err = rows.Scan(
			&acc.Name,
			&accLoginTimestamp,
			&accIdenticon,
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
		acc.Identicon = accIdenticon.String

		ii.KeyUID = acc.KeyUID
		ii.Name = iiName.String
		ii.Width = int(iiWidth.Int64)
		ii.Height = int(iiHeight.Int64)
		ii.FileSize = int(iiFileSize.Int64)
		ii.ResizeTarget = int(iiResizeTarget.Int64)

		if ii.Name == "" && len(ii.Payload) == 0 && ii.Width == 0 && ii.Height == 0 && ii.FileSize == 0 && ii.ResizeTarget == 0 {
			ii = nil
		}

		// Last index
		li := len(rst) - 1

		// Don't process nil identity images
		if ii != nil {
			// attach the identity image to a previously created account if present, check keyUID matches
			if len(rst) > 0 && rst[li].KeyUID == acc.KeyUID {
				rst[li].Images = append(rst[li].Images, *ii)
				// else attach the identity image to the newly created account
			} else {
				acc.Images = append(acc.Images, *ii)
			}
		}

		// Append newly created account only if this is the first loop or the keyUID doesn't match
		if len(rst) == 0 || rst[li].KeyUID != acc.KeyUID {
			rst = append(rst, acc)
		}
	}

	return rst, nil
}

func (db *Database) SaveAccount(account Account) error {
	_, err := db.db.Exec("INSERT OR REPLACE INTO accounts (name, identicon, keycardPairing, keyUid) VALUES (?, ?, ?, ?)", account.Name, account.Identicon, account.KeycardPairing, account.KeyUID)
	return err
}

func (db *Database) UpdateAccount(account Account) error {
	_, err := db.db.Exec("UPDATE accounts SET name = ?, identicon = ?, keycardPairing = ? WHERE keyUid = ?", account.Name, account.Identicon, account.KeycardPairing, account.KeyUID)
	return err
}

func (db *Database) UpdateAccountKeycardPairing(account Account) error {
	_, err := db.db.Exec("UPDATE accounts SET keycardPairing = ? WHERE keyUid = ?", account.KeycardPairing, account.KeyUID)
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

func (db *Database) GetIdentityImages(keyUID string) (iis []*images.IdentityImage, err error) {
	rows, err := db.db.Query(`SELECT key_uid, name, image_payload, width, height, file_size, resize_target FROM identity_images WHERE key_uid = ?`, keyUID)
	if err != nil {
		return nil, err
	}
	defer func() {
		err = rows.Close()
	}()

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
	var ii images.IdentityImage
	err := db.db.QueryRow("SELECT key_uid, name, image_payload, width, height, file_size, resize_target FROM identity_images WHERE key_uid = ? AND name = ?", keyUID, it).Scan(&ii.KeyUID, &ii.Name, &ii.Payload, &ii.Width, &ii.Height, &ii.FileSize, &ii.ResizeTarget)
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	return &ii, nil
}

func (db *Database) StoreIdentityImages(keyUID string, iis []*images.IdentityImage) (err error) {
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

	db.publishOnIdentityImageSubscriptions()

	return nil
}

func (db *Database) SubscribeToIdentityImageChanges() chan struct{} {
	s := make(chan struct{}, 100)
	db.identityImageSubscriptions = append(db.identityImageSubscriptions, s)
	return s
}

func (db *Database) publishOnIdentityImageSubscriptions() {
	// Publish on channels, drop if buffer is full
	for _, s := range db.identityImageSubscriptions {
		select {
		case s <- struct{}{}:
		default:
			log.Warn("subscription channel full, dropping message")
		}
	}
}

func (db *Database) DeleteIdentityImage(keyUID string) error {
	_, err := db.db.Exec(`DELETE FROM identity_images WHERE key_uid = ?`, keyUID)
	return err
}
