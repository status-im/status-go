package account

import (
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/status-im/status-go/extkeys"
)

// extendedKeyImporter import ECDSA key (obtained from extended key) and returns decrypted key for account.
type extendedKeyImporter interface {
	Import(keyStore accountKeyStorer, extKey *extkeys.ExtendedKey, password string) (address, pubKey string, err error)
}

type extendedKeyImporterBase struct{}

// Import processes incoming extended key, extracts required info and creates corresponding account key.
// Once account key is formed, that key is put (if not already) into keystore i.e. key is *encoded* into key file.
func (i *extendedKeyImporterBase) Import(keyStore accountKeyStorer, extKey *extkeys.ExtendedKey, password string) (address, pubKey string, err error) {
	// imports extended key, create key file (if necessary)
	account, err := keyStore.ImportExtendedKey(extKey, password)
	if err != nil {
		return "", "", err
	}
	address = account.Address.Hex()

	// obtain public key to return
	account, key, err := keyStore.AccountDecryptedKey(account, password)
	if err != nil {
		return address, "", err
	}
	pubKey = gethcommon.ToHex(crypto.FromECDSAPub(&key.PrivateKey.PublicKey))

	return
}
