package account

import (
	"crypto/ecdsa"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/status-im/status-go/extkeys"
)

type accountNode interface {
	AccountKeyStore() (accountKeyStorer, error)
	AccountManager() (gethAccountManager, error)
	WhisperService() (whisperService, error)
}

type gethAccountManager interface {
	Wallets() []accounts.Wallet
}

type accountKeyStorer interface {
	AccountDecryptedKey(account accounts.Account, password string) (accounts.Account, *keystore.Key, error)
	IncSubAccountIndex(account accounts.Account, password string) error
	ImportExtendedKey(extKey *extkeys.ExtendedKey, password string) (accounts.Account, error)
	Accounts() []accounts.Account
}

type whisperService interface {
	DeleteKeyPairs() error
	SelectKeyPair(key *ecdsa.PrivateKey) error
}

type subAccountFinder interface {
	Find(keyStore accountKeyStorer, extKey *extkeys.ExtendedKey, subAccountIndex uint32) ([]accounts.Account, error)
}

type keyFileFinder interface {
	Find(keyStoreDir string, addressObj gethcommon.Address) ([]byte, error)
}

// importExtendedKey processes incoming extended key, extracts required info and creates corresponding account key.
// Once account key is formed, that key is put (if not already) into keystore i.e. key is *encoded* into key file.
type extendedKeyImporter interface {
	Import(keyStore accountKeyStorer, extKey *extkeys.ExtendedKey, password string) (address, pubKey string, err error)
}
