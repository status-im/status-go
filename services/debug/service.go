package debug

import (
	"context"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rpc"
	whisper "github.com/ethereum/go-ethereum/whisper/whisperv6"
)

// Make sure that Service implements node.Service interface.
var _ node.Service = (*Service)(nil)

// Subscriber whisper interface to add key pairs
type Subscriber interface {
	SubscribeEnvelopeEvents(events chan<- whisper.EnvelopeEvent) event.Subscription
}

// Poster interface for a posting shh messages.
type Poster interface {
	Post(context.Context, whisper.NewMessage) (hexutil.Bytes, error)
}

// Service represents provides some debugging specific endpoints.
type Service struct {
	w Subscriber
	p Poster // *shhext.PublicAPI
}

// New returns a new Service.
func New(w Subscriber) *Service {
	return &Service{w: w}
}

// Protocols returns a new protocols list. In this case, there are none.
func (s *Service) Protocols() []p2p.Protocol {
	return []p2p.Protocol{}
}

// APIs returns a list of new APIs.
func (s *Service) APIs() []rpc.API {

	return []rpc.API{
		{
			Namespace: "debug",
			Version:   "1.0",
			Service:   NewAPI(s),
			Public:    true,
		},
	}
}

// SetPoster sets the poster to be used generally shhext.PublicAPI to manage
// API calls.
func (s *Service) SetPoster(p Poster) {
	s.p = p
}

// Start is run when a service is started.
// It does nothing in this case but is required by `node.Service` interface.
func (s *Service) Start(server *p2p.Server) error {
	return nil
}

// Stop is run when a service is stopped.
// It does nothing in this case but is required by `node.Service` interface.
func (s *Service) Stop() error {
	return nil
}
