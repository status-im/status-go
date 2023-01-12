package collectibles

import (
	"github.com/ethereum/go-ethereum/p2p"
	ethRpc "github.com/ethereum/go-ethereum/rpc"
	"github.com/status-im/status-go/account"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/rpc"
)

// Collectibles service
type Service struct {
	api *API
}

// Returns a new Collectibles Service.
func NewService(rpcClient *rpc.Client, accountsManager *account.GethManager, config *params.NodeConfig) *Service {
	return &Service{
		NewAPI(rpcClient, accountsManager, config),
	}
}

// Protocols returns a new protocols list. In this case, there are none.
func (s *Service) Protocols() []p2p.Protocol {
	return []p2p.Protocol{}
}

// APIs returns a list of new APIs.
func (s *Service) APIs() []ethRpc.API {
	return []ethRpc.API{
		{
			Namespace: "collectibles",
			Version:   "0.1.0",
			Service:   s.api,
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
