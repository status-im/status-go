package accounts

import (
	"context"
	"encoding/json"

	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/params"
)

func NewSettingsAPI(db *accounts.Database) *SettingsAPI {
	return &SettingsAPI{db}
}

// SettingsAPI is class with methods available over RPC.
type SettingsAPI struct {
	db *accounts.Database
}

func (api *SettingsAPI) SaveConfig(ctx context.Context, typ string, conf json.RawMessage) error {
	return api.db.SaveConfig(typ, conf)
}

func (api *SettingsAPI) GetConfig(ctx context.Context, typ string) (json.RawMessage, error) {
	return api.db.GetConfigBlob(typ)
}

func (api *SettingsAPI) GetConfigs(ctx context.Context, types []string) (map[string]json.RawMessage, error) {
	return api.db.GetConfigBlobs(types)
}

func (api *SettingsAPI) SaveNodeConfig(ctx context.Context, conf *params.NodeConfig) error {
	return api.db.SaveConfig(accounts.NodeConfigTag, conf)
}
