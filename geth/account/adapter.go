package account

import (
	"crypto/ecdsa"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/status-im/status-go/extkeys"
	"github.com/status-im/status-go/geth/common"
)

// accountNode narrows responsibility geth/common.NodeManager interface.
type accountNode interface {
	// AccountKeyStore returns narrowed responsibility of go-ethereum/accounts/keystore.KeyStore
	AccountKeyStore() (accountKeyStorer, error)
	// AccountManager returns narrowed responsibility of go-ethereum/accounts.Manager
	AccountManager() (gethAccountManager, error)
	// WhisperService returns narrowed responsibility  go-ethereum/whisper/whisperv5.Whisper
	WhisperService() (whisperService, error)
}

// gethAccountManager narrows responsibility of go-ethereum/accounts.Manager.
type gethAccountManager interface {
	Wallets() []accounts.Wallet
}

// accountKeyStorer narrows responsibility go-ethereum/accounts/keystore.KeyStore.
type accountKeyStorer interface {
	AccountDecryptedKey(account accounts.Account, password string) (accounts.Account, *keystore.Key, error)
	IncSubAccountIndex(account accounts.Account, password string) error
	ImportExtendedKey(extKey *extkeys.ExtendedKey, password string) (accounts.Account, error)
	Accounts() []accounts.Account
}

// whisperService narrows responsibility of go-ethereum/whisper/whisperv5.Whisper.
type whisperService interface {
	DeleteKeyPairs() error
	SelectKeyPair(key *ecdsa.PrivateKey) error
}

func newAccountNodeManager(node common.NodeManager) *accountManagerNodeManagerAdapter {
	return &accountManagerNodeManagerAdapter{node: node}
}

// accountManagerNodeManagerAdapter convert geth/common.NodeManager to accountNode interface.
type accountManagerNodeManagerAdapter struct {
	node common.NodeManager
}

func (a *accountManagerNodeManagerAdapter) AccountKeyStore() (accountKeyStorer, error) {
	return a.node.AccountKeyStore()
}
func (a *accountManagerNodeManagerAdapter) AccountManager() (gethAccountManager, error) {
	return a.node.AccountManager()
}
func (a *accountManagerNodeManagerAdapter) WhisperService() (whisperService, error) {
	return a.node.WhisperService()
}
