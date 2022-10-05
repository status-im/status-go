package ens

import (
	"github.com/ethereum/go-ethereum/p2p"
	ethRpc "github.com/ethereum/go-ethereum/rpc"
	"github.com/status-im/status-go/account"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/rpc"
	"github.com/status-im/status-go/services/rpcfilters"
)

// NewService initializes service instance.
func NewService(rpcClient *rpc.Client, accountsManager *account.GethManager, rpcFiltersSrvc *rpcfilters.Service, config *params.NodeConfig) *Service {
	return &Service{
		rpcClient,
		accountsManager,
		rpcFiltersSrvc,
		config,
		NewAPI(rpcClient, accountsManager, rpcFiltersSrvc, config),
	}
}

// Service is a browsers service.
type Service struct {
	rpcClient       *rpc.Client
	accountsManager *account.GethManager
	rpcFiltersSrvc  *rpcfilters.Service
	config          *params.NodeConfig
	api             *API
}

// Start a service.
func (s *Service) Start() error {
	return nil
}

// Stop a service.
func (s *Service) Stop() error {
	s.api.Stop()
	return nil
}

func (s *Service) API() *API {
	return s.api
}

// APIs returns list of available RPC APIs.
func (s *Service) APIs() []ethRpc.API {
	return []ethRpc.API{
		{
			Namespace: "ens",
			Version:   "0.1.0",
			Service:   s.api,
		},
	}
}

// Protocols returns list of p2p protocols.
func (s *Service) Protocols() []p2p.Protocol {
	return nil
}
