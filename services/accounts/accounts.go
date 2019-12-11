package accounts

import (
	"context"

	"github.com/ethereum/go-ethereum/event"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/multiaccounts/accounts"
)

func NewAccountsAPI(db *accounts.Database, feed *event.Feed) *API {
	return &API{db, feed}
}

// API is class with methods available over RPC.
type API struct {
	db   *accounts.Database
	feed *event.Feed
}

func (api *API) SaveAccounts(ctx context.Context, accounts []accounts.Account) error {
	err := api.db.SaveAccounts(accounts)
	if err != nil {
		return err
	}
	api.feed.Send(accounts)
	return nil
}

func (api *API) GetAccounts(ctx context.Context) ([]accounts.Account, error) {
	return api.db.GetAccounts()
}

func (api *API) DeleteAccount(ctx context.Context, address types.Address) error {
	return api.db.DeleteAccount(address)
}
