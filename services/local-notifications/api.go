package localnotifications

import (
	"context"

	"github.com/ethereum/go-ethereum/log"
)

func NewAPI(s *Service) *API {
	return &API{s}
}

type API struct {
	s *Service
}

func (api *API) WatchTransaction(ctx context.Context) error {
	log.Debug("Add watch tx")
	// TODO: Insert into DB and check when processing transactions
	return nil
}
