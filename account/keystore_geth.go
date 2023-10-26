package account

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"

	gethbridge "github.com/status-im/status-go/eth-node/bridge/geth"
	"github.com/status-im/status-go/eth-node/types"
)

func hasWritePermission(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}

	// Check if the current user has write permissions
	if info.Mode().Perm()&0200 == 0 {
		return false
	}

	return true
}

// makeAccountManager creates ethereum accounts.Manager with single disk backend and lightweight kdf.
// If keydir is empty new temporary directory with go-ethereum-keystore will be intialized.
func makeAccountManager(keydir string) (manager *accounts.Manager, err error) {

	parentDir := "/" // Assuming root as a default parent directory

	if keydir != "" {
		parentDir = filepath.Dir(keydir)
	} else {
		parentDir = os.TempDir()
	}

	if !hasWritePermission(parentDir) {
		return nil, fmt.Errorf("no write permission on the directory -> %s", parentDir)
	}

	if keydir == "" {
		// There is no datadir.
		keydir, err = os.MkdirTemp("", "go-ethereum-keystore")
	}
	if err != nil {
		return nil, err
	}
	// we don't need to do this for android apps now
	// because at this stage we're already creating the keystoredir
	// on the client side
	// TODO: pass a parameter to not do this only for the android client
	//if err := os.MkdirAll(keydir, 0700); err != nil {
	//	return nil, err
	//}
	config := accounts.Config{InsecureUnlockAllowed: false}
	return accounts.NewManager(&config, keystore.NewKeyStore(keydir, keystore.LightScryptN, keystore.LightScryptP)), nil
}

func makeKeyStore(manager *accounts.Manager) (types.KeyStore, error) {
	backends := manager.Backends(keystore.KeyStoreType)
	if len(backends) == 0 {
		return nil, ErrAccountKeyStoreMissing
	}
	keyStore, ok := backends[0].(*keystore.KeyStore)
	if !ok {
		return nil, ErrAccountKeyStoreMissing
	}

	return gethbridge.WrapKeyStore(keyStore), nil
}
