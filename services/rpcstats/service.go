package rpcstats

import (
	"github.com/ethereum/go-ethereum/rpc"
)

// Service represents our own implementation of status status operations.
type Service struct{}

// New returns a new Service.
func New() *Service {
	return &Service{}
}

// APIs returns a list of new APIs.
func (s *Service) APIs() []rpc.API {
	return []rpc.API{
		{
			Namespace: "rpcstats",
			Version:   "1.0",
			Service:   NewAPI(s),
			Public:    true,
		},
	}
}

// Start is run when a service is started.
// It does nothing in this case but is required by `node.Service` interface.
func (s *Service) Start() error {
	resetStats()
	return nil
}

// Stop is run when a service is stopped.
// It does nothing in this case but is required by `node.Service` interface.
func (s *Service) Stop() error {
	return nil
}
