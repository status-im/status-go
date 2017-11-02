package account

import (
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"io/ioutil"
	"os"
	"path/filepath"
)

var keyFinder keyFileFinder = keyFileFinderBase{}

type keyFileFinderBase struct{}

func (kf keyFileFinderBase) Find(keyStoreDir string, addressObj common.Address) ([]byte, error) {
	var err error
	var foundKeyFile []byte

	checkAccountKey := func(path string, fileInfo os.FileInfo) error {
		if len(foundKeyFile) > 0 || fileInfo.IsDir() {
			return nil
		}

		rawKeyFile, e := ioutil.ReadFile(path)
		if e != nil {
			return fmt.Errorf("invalid account key file: %v", e)
		}

		var accountKey struct {
			Address string `json:"address"`
		}
		if e := json.Unmarshal(rawKeyFile, &accountKey); e != nil {
			return fmt.Errorf("failed to read key file: %s", e)
		}

		if common.HexToAddress("0x"+accountKey.Address).Hex() == addressObj.Hex() {
			foundKeyFile = rawKeyFile
		}

		return nil
	}
	// locate key within key store directory (address should be within the file)
	err = filepath.Walk(keyStoreDir, func(path string, fileInfo os.FileInfo, er error) error {
		if er != nil {
			return er
		}
		return checkAccountKey(path, fileInfo)
	})
	return foundKeyFile, err
}
