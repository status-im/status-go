package pairing

import (
	"os"
	"path/filepath"

	"github.com/status-im/status-go/account/generator"
	"github.com/status-im/status-go/eth-node/keystore"
)

func validateKeys(keys map[string][]byte, password string) error {
	for _, key := range keys {
		k, err := keystore.DecryptKey(key, password)
		if err != nil {
			return err
		}

		err = generator.ValidateKeystoreExtendedKey(k)
		if err != nil {
			return err
		}
	}

	return nil
}

func emptyDir(dir string) error {
	// Open the directory
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()

	// Get all the directory entries
	entries, err := d.Readdir(-1)
	if err != nil {
		return err
	}

	// Remove all the files and directories
	for _, entry := range entries {
		name := entry.Name()
		if name == "." || name == ".." {
			continue
		}
		path := filepath.Join(dir, name)
		if entry.IsDir() {
			err = os.RemoveAll(path)
			if err != nil {
				return err
			}
		} else {
			err = os.Remove(path)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
