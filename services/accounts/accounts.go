package accounts

import (
	"context"

	"github.com/status-im/status-go/multiaccounts/accounts"
)

func NewAccountsAPI(db *accounts.Database) *API {
	return &API{db}
}

// API is class with methods available over RPC.
type API struct {
	db *accounts.Database
}

func (api *API) SaveAccounts(ctx context.Context, accounts []accounts.Account) error {
	return api.db.SaveAccounts(accounts)
}

func (api *API) GetAccounts(ctx context.Context) ([]accounts.Account, error) {
	return api.db.GetAccounts()
}
