package settings

import (
	"context"

	"github.com/status-im/status-go/accountsstore/settings"
)

func NewAccountsAPI(db *settings.Database) *AccountsAPI {
	return &AccountsAPI{db}
}

// AccountsAPI is class with methods available over RPC.
type AccountsAPI struct {
	db *settings.Database
}

func (api *AccountsAPI) SaveAccounts(ctx context.Context, accounts []settings.Account) error {
	return api.db.SaveAccounts(accounts)
}

func (api *AccountsAPI) GetAccounts(ctx context.Context) ([]settings.Account, error) {
	return api.db.GetAccounts()
}
