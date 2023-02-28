package pairing

import "github.com/status-im/status-go/api"

func updateLoggedInKeyUID(accountPayloadManagerConfig *AccountPayloadManagerConfig, backend *api.GethStatusBackend) {
	activeAccount, _ := backend.GetActiveAccount()
	if activeAccount != nil {
		accountPayloadManagerConfig.LoggedInKeyUID = activeAccount.KeyUID
	}
}
