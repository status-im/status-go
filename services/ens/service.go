package ens

import (
	"github.com/ethereum/go-ethereum/p2p"
	ethRpc "github.com/ethereum/go-ethereum/rpc"
	"github.com/status-im/status-go/rpc"
)

// NewService initializes service instance.
func NewService(rpcClient *rpc.Client) *Service {
	return &Service{rpcClient}
}

// Service is a browsers service.
type Service struct {
	rpcClient *rpc.Client
}

// Start a service.
func (s *Service) Start() error {
	return nil
}

// Stop a service.
func (s *Service) Stop() error {
	return nil
}

// APIs returns list of available RPC APIs.
func (s *Service) APIs() []ethRpc.API {
	return []ethRpc.API{
		{
			Namespace: "ens",
			Version:   "0.1.0",
			Service:   NewAPI(s.rpcClient),
		},
	}
}

// Protocols returns list of p2p protocols.
func (s *Service) Protocols() []p2p.Protocol {
	return nil
}
