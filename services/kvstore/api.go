package kvstore

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

type StoreEntry struct {
	RlnRateLimitEnabled bool `json:"rlnRateLimitEnabled"`
}

func (api *API) GetStoreEntry(ctx context.Context) (*StoreEntry, error) {
	rlnRateLimitEnabled, err := api.db.GetBool(ConfigRlnRateLimitEnabled)
	if err != nil {
		return nil, err
	}

	configs := StoreEntry{
		RlnRateLimitEnabled: rlnRateLimitEnabled,
	}

	return &configs, nil
}

func (api *API) SetRlnRateLimitEnabled(ctx context.Context, enabled bool) error {
	return api.db.SetBool(ConfigRlnRateLimitEnabled, enabled)
}
