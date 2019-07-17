package browsers

import (
	"context"
	"errors"
)

var (
	// ErrServiceNotInitialized returned when wallet is not initialized/started,.
	ErrServiceNotInitialized = errors.New("browsers service is not initialized")
)

func NewAPI(s *Service) *API {
	return &API{s}
}

// API is class with methods available over RPC.
type API struct {
	s *Service
}

func (api *API) AddBrowser(ctx context.Context, browser Browser) error {
	if api.s.db == nil {
		return ErrServiceNotInitialized
	}
	return api.s.db.InsertBrowser(browser)
}

func (api *API) GetBrowsers(ctx context.Context) ([]*Browser, error) {
	if api.s.db == nil {
		return nil, ErrServiceNotInitialized
	}
	return api.s.db.GetBrowsers()
}

func (api *API) DeleteBrowser(ctx context.Context, id string) error {
	if api.s.db == nil {
		return ErrServiceNotInitialized
	}
	return api.s.db.DeleteBrowser(id)
}
