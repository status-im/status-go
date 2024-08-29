package appgeneral

import (
	"context"
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
