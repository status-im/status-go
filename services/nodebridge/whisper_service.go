package nodebridge

import (
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/status-im/status-go/eth-node/types"
)

// Make sure that WhisperService implements node.Service interface.
var _ node.Service = (*WhisperService)(nil)

type WhisperService struct {
	Whisper types.Whisper
}

// Protocols returns a new protocols list. In this case, there are none.
func (w *WhisperService) Protocols() []p2p.Protocol {
	return []p2p.Protocol{}
}

// APIs returns a list of new APIs.
func (w *WhisperService) APIs() []rpc.API {
	return []rpc.API{
		{
			Namespace: "status",
			Version:   "1.0",
			Service:   w.Whisper,
			Public:    false,
		},
	}
}

// Start is run when a service is started.
// It does nothing in this case but is required by `node.Service` interface.
func (w *WhisperService) Start(server *p2p.Server) error {
	return nil
}

// Stop is run when a service is stopped.
// It does nothing in this case but is required by `node.Service` interface.
func (w *WhisperService) Stop() error {
	return nil
}
