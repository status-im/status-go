package appgeneral

import (
	"context"

	"github.com/status-im/status-go/pkg/version"
)

type API struct {
	s *Service
}

func NewAPI(s *Service) *API {
	return &API{s: s}
}

// Returns a list of currencies for user's selection
func (api *API) GetCurrencies(context context.Context) []*Currency {
	return GetCurrencies()
}

func (api *API) Version(context context.Context) string {
	return version.Version()
}
