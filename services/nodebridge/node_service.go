package nodebridge

import (
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/status-im/status-go/eth-node/types"
)

// Make sure that NodeService implements node.Service interface.
var _ node.Service = (*NodeService)(nil)

type NodeService struct {
	Node types.Node
}

// Protocols returns a new protocols list. In this case, there are none.
func (w *NodeService) Protocols() []p2p.Protocol {
	return []p2p.Protocol{}
}

// APIs returns a list of new APIs.
func (w *NodeService) APIs() []rpc.API {
	return []rpc.API{
		{
			Namespace: "status",
			Version:   "1.0",
			Service:   w.Node,
			Public:    false,
		},
	}
}

// Start is run when a service is started.
// It does nothing in this case but is required by `node.Service` interface.
func (w *NodeService) Start(server *p2p.Server) error {
	return nil
}

// Stop is run when a service is stopped.
// It does nothing in this case but is required by `node.Service` interface.
func (w *NodeService) Stop() error {
	return nil
}
