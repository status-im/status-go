package chatapi

import (
	gethnode "github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/status-im/status-go/node"
)

// Make sure that Service implements node.Service interface.
var _ gethnode.Service = (*Service)(nil)

// Service represents our own implementation of personal sign operations.
type Service struct {
	api *API
}

// New returns a new Service.
func New(node *node.StatusNode) *Service {
	return &Service{
		api: NewMockAPI(node),
	}
}

// Protocols returns a new protocols list. In this case, there are none.
func (s *Service) Protocols() []p2p.Protocol {
	return []p2p.Protocol{}
}

// APIs returns a list of new APIs.
func (s *Service) APIs() []rpc.API {
	return []rpc.API{
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
	return nil
}
