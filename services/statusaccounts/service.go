package statusaccounts

import (
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rpc"
	staccount "github.com/status-im/status-go/account"
)

type Service struct {
	g *generator
}

func New(accountManager *accounts.Manager) *Service {
	return &Service{
		g: newGenerator(),
	}
}

func (s *Service) Start(*p2p.Server) error {
	return nil
}

func (s *Service) Protocols() []p2p.Protocol {
	return nil
}

func (s *Service) Stop() error {
	return nil
}

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
