package kvstore2

import (
	"context"
)

func NewAPI(db *Database) *API {
	return &API{db}
}

// API is class with methods available over RPC.
type API struct {
	db *Database
}

type KvstoreConfigs struct {
	RlnRateLimitEnabled bool `json:"rlnRateLimitEnabled"`
}

func (api *API) GetKvstoreConfigs(ctx context.Context) (*KvstoreConfigs, error) {
	rlnRateLimitEnabled, err := api.db.GetBool(ConfigRlnRateLimitEnabled)
	if err != nil {
		return nil, err
	}

	configs := KvstoreConfigs{
		RlnRateLimitEnabled: rlnRateLimitEnabled,
	}

	return &configs, nil
}

func (api *API) SaveRlnRateLimitEnabled(ctx context.Context, enabled bool) error {
	return api.db.SetBool(ConfigRlnRateLimitEnabled, enabled)
}
