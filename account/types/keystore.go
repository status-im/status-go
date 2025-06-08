package types

import (
	"crypto/ecdsa"

	"github.com/status-im/extkeys"

	ethtypes "github.com/status-im/status-go/eth-node/types"
)

type KeyStore interface {
	// ImportECDSA imports a private key into the keystore
	ImportECDSA(priv *ecdsa.PrivateKey, passphrase string) (Account, error)
	// ImportSingleExtendedKey imports an extended key setting it in both the PrivateKey and ExtendedKey fields
	// of the Key struct.
	// ImportExtendedKey is used in older version of Status where PrivateKey is set to be the BIP44 key at index 0,
	// and ExtendedKey is the extended key of the BIP44 key at index 1.
	ImportSingleExtendedKey(extKey *extkeys.ExtendedKey, passphrase string) (Account, error)
	// ImportExtendedKeyForWallet stores ECDSA key (obtained from extended key) along with CKD#2 (root for sub-accounts)
	// If key file is not found, it is created. Key is encrypted with the given passphrase.
	// Deprecated: status-go is now using ImportSingleExtendedKey
	ImportExtendedKeyForWallet(extKey *extkeys.ExtendedKey, passphrase string) (Account, error)
	// AccountDecryptedKey returns decrypted key for account (provided that password is correct).
	AccountDecryptedKey(address ethtypes.Address, passphrase string) (Account, *ethtypes.Key, error)
	// Delete deletes the key matched by account if the passphrase is correct.
	// If the account contains no filename, the address must match a unique key.
	Delete(address ethtypes.Address) error
	// Find returns the account matched by address
	Find(address ethtypes.Address) (Account, error)
	// Accounts returns all accounts in the keystore
	Accounts() []Account
	// VerifyPassword tries to decrypt a given account key file, with a provided password.
	// If no error is returned, then account is considered verified.
	VerifyPassword(address ethtypes.Address, passphrase string) (*ethtypes.Key, error)
	// ReEncryptKeyStoreDir re-encrypts all keys in the keystore directory.
	ReEncryptKeyStoreDir(keyDirPath, oldPass, newPass string) error
}
