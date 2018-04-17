package mailservice

import (
	"crypto/ecdsa"

	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rpc"
	whisper "github.com/ethereum/go-ethereum/whisper/whisperv6"
)

// MailService is a service that provides some additional Whisper API.
type MailService struct {
	whisper *whisper.Whisper
	nodeID  *ecdsa.PrivateKey
}

// Make sure that MailService implements node.Service interface.
var _ node.Service = (*MailService)(nil)

// New returns a new MailService.
func New(w *whisper.Whisper) *MailService {
	return &MailService{whisper: w}
}

// Protocols returns a new protocols list. In this case, there are none.
func (s *MailService) Protocols() []p2p.Protocol {
	return []p2p.Protocol{}
}

// APIs returns a list of new APIs.
func (s *MailService) APIs() []rpc.API {
	return []rpc.API{
		{
			Namespace: "shh",
			Version:   "1.0",
			Service:   NewPublicAPI(s),
			Public:    true,
		},
	}
}

// Start is run when a service is started.
// It does nothing in this case but is required by `node.Service` interface.
func (s *MailService) Start(server *p2p.Server) error {
	s.nodeID = server.PrivateKey
	return nil
}

// Stop is run when a service is stopped.
// It does nothing in this case but is required by `node.Service` interface.
func (s *MailService) Stop() error {
	return nil
}
