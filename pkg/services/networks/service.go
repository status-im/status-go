package networks

import (
	"github.com/ethereum/go-ethereum/rpc"
)

// Service exposes the network manager over JSON-RPC and owns its lifecycle.
type Service struct {
	manager *Manager
}

func NewService(manager *Manager) *Service {
	return &Service{manager: manager}
}

func (s *Service) APIs() []rpc.API {
	return []rpc.API{
		{
			Namespace: "networks",
			Version:   "0.1.0",
			Service:   NewAPI(s.manager),
		},
	}
}

func (s *Service) Start() error {
	s.manager.Start()
	return nil
}

func (s *Service) Stop() error {
	s.manager.Stop()
	return nil
}
