package statusaccounts

import (
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rpc"
	staccount "github.com/status-im/status-go/account"
)

// Service is a struct that provides the StatusAccounts APi.
type Service struct {
	g *generator
}

// New returns a new Service.
func New(accountManager *accounts.Manager) *Service {
	return &Service{
		g: newGenerator(),
	}
}

// Start is called when a service is started.
func (s *Service) Start(*p2p.Server) error {
	return nil
}

// Protocols returns a new protocols list. In this case, there are none.
func (s *Service) Protocols() []p2p.Protocol {
	return nil
}

// Stop is called when a service is stopped.
func (s *Service) Stop() error {
	return nil
}

// APIs returns a list of new APIs.
func (s *Service) APIs() []rpc.API {
	return []rpc.API{
		{
			Namespace: "statusaccounts",
			Version:   "0.1.0",
			Service:   &API{s},
			Public:    false,
		},
	}
}

// SetAccountManager sets the current account manager.
func (s *Service) SetAccountManager(am *staccount.Manager) {
	s.g.setAccountManager(am)
}
