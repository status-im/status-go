package account

import (
	"io/ioutil"
	"os"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
)

// makeAccountManager creates ethereum accounts.Manager with single disk backend and lightweight kdf.
// If keydir is empty new temporary directory with go-ethereum-keystore will be intialized.
func makeAccountManager(keydir string) (manager *accounts.Manager, err error) {
	if keydir == "" {
		// There is no datadir.
		keydir, err = ioutil.TempDir("", "go-ethereum-keystore")
	}
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(keydir, 0700); err != nil {
		return nil, err
	}
	return accounts.NewManager(keystore.NewKeyStore(keydir, keystore.LightScryptN, keystore.LightScryptP)), nil
}
