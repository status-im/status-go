package accountsmanagement

import (
	"crypto/ecdsa"

	"github.com/status-im/extkeys"

	cryptoypes "github.com/status-im/status-go/crypto/types"
	"github.com/status-im/status-go/internal/accounts-management/keystore"
)

type KeyStore interface {
	// ImportECDSA imports a private key into the keystore
	ImportECDSA(priv *ecdsa.PrivateKey, passphrase string) (keystore.KeystoreAccount, error)
	// ImportSingleExtendedKey imports an extended key setting it in both the PrivateKey and ExtendedKey fields
	// of the Key struct.
	// ImportExtendedKey is used in older version of Status where PrivateKey is set to be the BIP44 key at index 0,
	// and ExtendedKey is the extended key of the BIP44 key at index 1.
	ImportSingleExtendedKey(extKey *extkeys.ExtendedKey, passphrase string) (keystore.KeystoreAccount, error)
	// AccountDecryptedKey returns decrypted key for account (provided that password is correct).
	AccountDecryptedKey(address cryptoypes.Address, passphrase string) (keystore.KeystoreAccount, *ecdsa.PrivateKey, *extkeys.ExtendedKey, error)
	// Delete deletes the key matched by account.
	// If the account contains no filename, the address must match a unique key.
	Delete(address cryptoypes.Address, passphrase string) error
	// Find returns the account matched by address
	Find(address cryptoypes.Address) (keystore.KeystoreAccount, error)
	// Accounts returns all accounts in the keystore
	Accounts() []keystore.KeystoreAccount
	// ReEncryptKeyStoreDir re-encrypts all keys in the keystore directory.
	ReEncryptKeyStoreDir(oldPass, newPass string) error
	// MigrateKeyStoreDir migrates the keystore directory from one location to another.
	MigrateKeyStoreDir(newDir string) error
	// KeystorePath returns the path to the keystore directory
	KeystorePath() string
}
