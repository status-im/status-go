package gif

import (
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/status-im/status-go/internal/db/multiaccounts/accounts"
)

// Service represents out own implementation of personal sign operations.
type Service struct {
	accountsDB *accounts.Database
}

func (s *Service) ThirdpartyServicesEnabled() bool {
	enabled, err := s.accountsDB.ThirdpartyServicesEnabled()
	if err != nil {
		return true
	}
	return enabled
}

// New returns a new Service.
func NewService(db *accounts.Database) *Service {
	return &Service{accountsDB: db}
}

// APIs returns a list of new APIs.
func (s *Service) APIs() []rpc.API {
	if !s.ThirdpartyServicesEnabled() {
		return nil
	}
	return []rpc.API{
		{
			Namespace: "gif",
			Version:   "0.1.0",
			Service:   NewGifAPI(s.accountsDB),
			Public:    true,
		},
	}
}

// Start is run when a service is started.
func (s *Service) Start() error {
	return nil
}

// Stop is run when a service is stopped.
func (s *Service) Stop() error {
	return nil
}
