package mailservice

import (
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rpc"
	whisper "github.com/ethereum/go-ethereum/whisper/whisperv6"
)

// ServiceProvider provides node and required services.
type ServiceProvider interface {
	Node() (*node.Node, error)
	WhisperService() (*whisper.Whisper, error)
}

// MailService is a service that provides some additional Whisper API.
type MailService struct {
	provider ServiceProvider
}

// Make sure that MailService implements node.Service interface.
var _ node.Service = (*MailService)(nil)

// New returns a new MailService.
func New(provider ServiceProvider) *MailService {
	return &MailService{provider}
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
			Service:   NewPublicAPI(s.provider),
			Public:    true,
		},
	}
}

// Start is run when a service is started.
// It does nothing in this case but is required by `node.Service` interface.
func (s *MailService) Start(server *p2p.Server) error {
	return nil
}

// Stop is run when a service is stopped.
// It does nothing in this case but is required by `node.Service` interface.
func (s *MailService) Stop() error {
	return nil
}
