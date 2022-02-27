package accounts

import (
	"context"

	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/multiaccounts/settings"
	"github.com/status-im/status-go/nodecfg"
	"github.com/status-im/status-go/params"
)

func NewSettingsAPI(db *accounts.Database, config *params.NodeConfig) *SettingsAPI {
	return &SettingsAPI{db, config}
}

// SettingsAPI is class with methods available over RPC.
type SettingsAPI struct {
	db     *accounts.Database
	config *params.NodeConfig
}

func (api *SettingsAPI) SaveSetting(ctx context.Context, typ string, val interface{}) error {
	// NOTE(Ferossgp): v0.62.0 Backward compatibility, skip this for older clients instead of returning error
	if typ == "waku-enabled" {
		return nil
	}

	return api.db.SaveSetting(typ, val)
}

func (api *SettingsAPI) GetSettings(ctx context.Context) (settings.Settings, error) {
	return api.db.GetSettings()
}

// NodeConfig returns the currently used node configuration
func (api *SettingsAPI) NodeConfig(ctx context.Context) (*params.NodeConfig, error) {
	return api.config, nil
}

// Saves the nodeconfig in the database. The node must be restarted for the changes to be applied
func (api *SettingsAPI) SaveNodeConfig(ctx context.Context, n *params.NodeConfig) error {
	return nodecfg.SaveNodeConfig(api.db.DB(), n)
}
