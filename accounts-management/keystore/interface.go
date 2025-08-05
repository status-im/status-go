package keystore

import (
	"crypto/ecdsa"
	"errors"

	"github.com/status-im/extkeys"

	"github.com/status-im/status-go/accounts-management/types"
	ethtypes "github.com/status-im/status-go/eth-node/types"
)

var (
	ErrNoMatch = errors.New("no key for given address or file")
	ErrDecrypt = errors.New("could not decrypt key with given password")
)

type KeyStore interface {
	// ImportECDSA imports a private key into the keystore
	ImportECDSA(priv *ecdsa.PrivateKey, passphrase string) (types.KeystoreAccount, error)
	// ImportSingleExtendedKey imports an extended key setting it in both the PrivateKey and ExtendedKey fields
	// of the Key struct.
	// ImportExtendedKey is used in older version of Status where PrivateKey is set to be the BIP44 key at index 0,
	// and ExtendedKey is the extended key of the BIP44 key at index 1.
	ImportSingleExtendedKey(extKey *extkeys.ExtendedKey, passphrase string) (types.KeystoreAccount, error)
	// AccountDecryptedKey returns decrypted key for account (provided that password is correct).
	AccountDecryptedKey(address ethtypes.Address, passphrase string) (types.KeystoreAccount, *ecdsa.PrivateKey, *extkeys.ExtendedKey, error)
	// Delete deletes the key matched by account.
	// If the account contains no filename, the address must match a unique key.
	Delete(address ethtypes.Address) error
	// Find returns the account matched by address
	Find(address ethtypes.Address) (types.KeystoreAccount, error)
	// Accounts returns all accounts in the keystore
	Accounts() []types.KeystoreAccount
	// ReEncryptKeyStoreDir re-encrypts all keys in the keystore directory.
	ReEncryptKeyStoreDir(oldPass, newPass string) error
	// MigrateKeyStoreDir migrates the keystore directory from one location to another.
	MigrateKeyStoreDir(newDir string) error
	// KeystorePath returns the path to the keystore directory
	KeystorePath() string
}
