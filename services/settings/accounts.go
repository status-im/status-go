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

func (api *AccountsAPI) SaveSubAccounts(ctx context.Context, accounts []settings.SubAccount) error {
	return api.db.SaveSubAccounts(accounts)
}

func (api *AccountsAPI) GetSubAccounts(ctx context.Context) ([]settings.SubAccount, error) {
	return api.db.GetSubAccounts()
}
