package accounts

import (
	"context"

	"github.com/status-im/status-go/multiaccounts/accounts"
)

func NewAccountsAPI(db *accounts.Database) *AccountsAPI {
	return &AccountsAPI{db}
}

// AccountsAPI is class with methods available over RPC.
type AccountsAPI struct {
	db *accounts.Database
}

func (api *AccountsAPI) SaveAccounts(ctx context.Context, accounts []accounts.Account) error {
	return api.db.SaveAccounts(accounts)
}

func (api *AccountsAPI) GetAccounts(ctx context.Context) ([]accounts.Account, error) {
	return api.db.GetAccounts()
}
