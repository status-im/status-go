package settings

import (
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/status-im/status-go/accountsstore/settings"
)

// NewService initializes service instance.
func NewService(db *settings.Database) *Service {
	return &Service{db}
}

// Service is a browsers service.
type Service struct {
	db *settings.Database
}

// Start a service.
func (s *Service) Start(*p2p.Server) error {
	return nil
}

// Stop a service.
func (s *Service) Stop() error {
	return nil
}

// APIs returns list of available RPC APIs.
func (s *Service) APIs() []rpc.API {
	return []rpc.API{
		{
			Namespace: "settings",
			Version:   "0.1.0",
			Service:   NewAPI(s.db),
		},
		{
			Namespace: "accounts",
			Version:   "0.1.0",
			Service:   NewAccountsAPI(s.db),
		},
	}
}

// Protocols returns list of p2p protocols.
func (s *Service) Protocols() []p2p.Protocol {
	return nil
}
