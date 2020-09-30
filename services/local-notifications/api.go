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

func (api *API) SwitchWalletNotifications(ctx context.Context, _ bool) error {
	log.Debug("Switch Transaction Notification")
	return nil
}
