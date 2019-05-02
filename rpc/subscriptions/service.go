package subscriptions

import (
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	gethrpc "github.com/ethereum/go-ethereum/rpc"

	"github.com/status-im/status-go/rpc"
)

// Make sure that Service implements node.Service interface.
var _ node.Service = (*Service)(nil)

// Service represents out own implementation of personal sign operations.
type Service struct {
	api *API
}

// New returns a new Service.
func New(rpc *rpc.Client) *Service {
	return &Service{
		api: NewPublicAPI(rpc),
	}
}

// Protocols returns a new protocols list. In this case, there are none.
func (s *Service) Protocols() []p2p.Protocol {
	return []p2p.Protocol{}
}

// APIs returns a list of new APIs.
func (s *Service) APIs() []gethrpc.API {
	return []gethrpc.API{
		{
			Namespace: "status",
			Version:   "1.0",
			Service:   s.api,
			Public:    true,
		},
	}
}

// Start is run when a service is started.
func (s *Service) Start(server *p2p.Server) error {
	return nil
}

// Stop is run when a service is stopped.
func (s *Service) Stop() error {
	return s.api.ClearSignalSubscriptions()
}
