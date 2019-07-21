package permissions

import (
	"context"
	"errors"
)

var (
	// ErrServiceNotInitialized returned when permissions is not initialized/started,.
	ErrServiceNotInitialized = errors.New("permissions service is not initialized")
)

func NewAPI(s *Service) *API {
	return &API{s}
}

// API is class with methods available over RPC.
type API struct {
	s *Service
}

func (api *API) AddDappPermissions(ctx context.Context, perms DappPermissions) error {
	if api.s.db == nil {
		return ErrServiceNotInitialized
	}
	return api.s.db.AddPermissions(perms)
}

func (api *API) GetDappPermissions(ctx context.Context) ([]DappPermissions, error) {
	if api.s.db == nil {
		return nil, ErrServiceNotInitialized
	}
	return api.s.db.GetPermissions()
}

func (api *API) DeleteDappPermissions(ctx context.Context, name string) error {
	if api.s.db == nil {
		return ErrServiceNotInitialized
	}
	return api.s.db.DeletePermission(name)
}
