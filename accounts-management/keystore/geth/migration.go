// Imported from github.com/ethereum/go-ethereum/accounts/keystore/passphrase.go
// and github.com/ethereum/go-ethereum/accounts/keystore/presale.go

package geth

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/status-im/status-go/crypto/types"
)

func migrateKeyStoreDir(oldDir, newDir string, addresses []string) error {
	paths := []string{}

	addressesMap := map[string]struct{}{}
	for _, address := range addresses {
		addressesMap[address] = struct{}{}
	}

	checkFile := func(path string, fileInfo os.FileInfo) error {
		if fileInfo.IsDir() || filepath.Dir(path) != oldDir {
			return nil
		}

		rawKeyFile, err := ioutil.ReadFile(path)
		if err != nil {
			return fmt.Errorf("invalid account key file: %v", err)
		}

		var accountKey struct {
			Address string `json:"address"`
		}
		if err := json.Unmarshal(rawKeyFile, &accountKey); err != nil {
			return fmt.Errorf("failed to read key file: %s", err)
		}

		address := types.HexToAddress("0x" + accountKey.Address).Hex()
		if _, ok := addressesMap[address]; ok {
			paths = append(paths, path)
		}

		return nil
	}

	err := filepath.Walk(oldDir, func(path string, fileInfo os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		return checkFile(path, fileInfo)
	})
	if err != nil {
		return fmt.Errorf("cannot traverse key store folder: %v", err)
	}

	for _, path := range paths {
		_, fileName := filepath.Split(path)
		newPath := filepath.Join(newDir, fileName)
		err := os.Rename(path, newPath)
		if err != nil {
			return err
		}
	}

	return nil
}
