package visualidentity

import (
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rpc"
)

// Service represents out own implementation of Identity Visual Representation.
type Service struct {
	api *API
}

// New returns a new Service.
func NewService() *Service {
	return &Service{
		api: NewAPI(),
	}
}

func (s *Service) Init() error {
	alphabet, err := LoadAlphabet()
	if err == nil {
		s.api.emojisAlphabet = alphabet
	}
	return err
}

// Protocols returns a new protocols list. In this case, there are none.
func (s *Service) Protocols() []p2p.Protocol {
	return []p2p.Protocol{}
}

// APIs returns a list of new APIs.
func (s *Service) APIs() []rpc.API {
	return []rpc.API{
		{
			Namespace: "visualIdentity",
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
