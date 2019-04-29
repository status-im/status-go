package incentivisation

import (
	"context"
)

// PublicAPI represents a set of APIs from the `web3.peer` namespace.
type PublicAPI struct {
	s *Service
}

// NewAPI creates an instance of the peer API.
func NewAPI(s *Service) *PublicAPI {
	return &PublicAPI{s: s}
}

func (api *PublicAPI) Registered(context context.Context) error {
	return nil
}
