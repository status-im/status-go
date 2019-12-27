package accounts

import (
	"context"

	"github.com/status-im/status-go/multiaccounts/accounts"
)

func NewSettingsAPI(db *accounts.Database) *SettingsAPI {
	return &SettingsAPI{db}
}

// SettingsAPI is class with methods available over RPC.
type SettingsAPI struct {
	db *accounts.Database
}

func (api *SettingsAPI) SaveSetting(ctx context.Context, typ string, val interface{}) error {
	return api.db.SaveSetting(typ, val)
}

func (api *SettingsAPI) GetSettings(ctx context.Context) (accounts.Settings, error) {
	return api.db.GetSettings()
}
