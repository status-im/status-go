package browsers

import (
	"context"
)

func NewAPI(db *Database) *API {
	return &API{db: db}
}

// API is class with methods available over RPC.
type API struct {
	db *Database
}

func (api *API) AddBrowser(ctx context.Context, browser Browser) error {
	return api.db.InsertBrowser(browser)
}

func (api *API) GetBrowsers(ctx context.Context) ([]*Browser, error) {
	return api.db.GetBrowsers()
}

func (api *API) DeleteBrowser(ctx context.Context, id string) error {
	return api.db.DeleteBrowser(id)
}
