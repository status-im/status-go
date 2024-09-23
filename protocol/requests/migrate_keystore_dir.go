package requests

import (
	"errors"

	"github.com/status-im/status-go/multiaccounts"
)

// MigrateKeyStoreDir represents a request to migrate key files to a new directory.
type MigrateKeyStoreDir struct {
	// Account represents the account associated with the key files.
	Account multiaccounts.Account `json:"account"`

	// Password is the password used to decrypt the key files.
	Password string `json:"password"`

	// OldDir is the old directory path where the key files are currently located.
	OldDir string `json:"oldDir"`

	// NewDir is the new directory path where the key files will be migrated to.
	NewDir string `json:"newDir"`
}

// Validate checks the validity of the MigrateKeyStoreDir request.
var (
	ErrMigrateKeyStoreDirEmptyAccount  = errors.New("migrate-keystore-dir: Account cannot be empty")
	ErrMigrateKeyStoreDirEmptyPassword = errors.New("migrate-keystore-dir: Password cannot be empty")
	ErrMigrateKeyStoreDirEmptyOldDir   = errors.New("migrate-keystore-dir: OldDir cannot be empty")
	ErrMigrateKeyStoreDirEmptyNewDir   = errors.New("migrate-keystore-dir: NewDir cannot be empty")
)

func (r *MigrateKeyStoreDir) Validate() error {
	if r.Account.KeyUID == "" {
		return ErrMigrateKeyStoreDirEmptyAccount
	}
	if r.Password == "" {
		return ErrMigrateKeyStoreDirEmptyPassword
	}
	if r.OldDir == "" {
		return ErrMigrateKeyStoreDirEmptyOldDir
	}
	if r.NewDir == "" {
		return ErrMigrateKeyStoreDirEmptyNewDir
	}
	return nil
}
