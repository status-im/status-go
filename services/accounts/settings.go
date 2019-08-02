package accounts

import (
	"context"
	"encoding/json"

	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/params"
)

func NewAPI(db *accounts.Database) *API {
	return &API{db}
}

// API is class with methods available over RPC.
type API struct {
	db *accounts.Database
}

func (api *API) SaveConfig(ctx context.Context, typ string, conf json.RawMessage) error {
	return api.db.SaveConfig(typ, conf)
}

func (api *API) GetConfig(ctx context.Context, typ string) (json.RawMessage, error) {
	rst, err := api.db.GetBlob(typ)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(rst), nil
}

func (api *API) SaveNodeConfig(ctx context.Context, conf *params.NodeConfig) error {
	return api.db.SaveConfig(accounts.NodeConfigTag, conf)
}
