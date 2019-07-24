package settings

import (
	"context"
	"encoding/json"

	"github.com/status-im/status-go/accountsstore/settings"
	"github.com/status-im/status-go/params"
)

func NewAPI(db *settings.Database) *API {
	return &API{db}
}

// API is class with methods available over RPC.
type API struct {
	db *settings.Database
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
	return api.db.SaveConfig(settings.NodeConfigTag, conf)
}
