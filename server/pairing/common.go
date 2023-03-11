package pairing

import (
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
