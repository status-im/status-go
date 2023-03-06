package pairing

import (
	"github.com/status-im/status-go/account/generator"
	"github.com/status-im/status-go/api"
	"github.com/status-im/status-go/eth-node/keystore"
)

func updateLoggedInKeyUID(accountPayloadManagerConfig *AccountPayloadManagerConfig, backend *api.GethStatusBackend) {
	activeAccount, _ := backend.GetActiveAccount()
	if activeAccount != nil {
		accountPayloadManagerConfig.LoggedInKeyUID = activeAccount.KeyUID
	}
}

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
