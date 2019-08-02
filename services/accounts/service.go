package accounts

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/status-im/status-go/multiaccounts"
	"github.com/status-im/status-go/multiaccounts/accounts"
)

// NewService initializes service instance.
func NewService(db *accounts.Database, mdb *multiaccounts.Database, login common.Address) *Service {
	return &Service{db, mdb, login}
}

// Service is a browsers service.
type Service struct {
	db    *accounts.Database
	mdb   *multiaccounts.Database
	login common.Address
}

// Start a service.
func (s *Service) Start(*p2p.Server) error {
	return nil
}

// Stop a service.
func (s *Service) Stop() error {
	return nil
}

// APIs returns list of available RPC APIs.
func (s *Service) APIs() []rpc.API {
	return []rpc.API{
		{
			Namespace: "settings",
			Version:   "0.1.0",
			Service:   NewAPI(s.db),
		},
		{
			Namespace: "accounts",
			Version:   "0.1.0",
			Service:   NewAccountsAPI(s.db),
		},
		{
			Namespace: "multiaccounts",
			Version:   "0.1.0",
			Service:   NewMultiAccountsAPI(s.mdb, s.login),
		},
	}
}

// Protocols returns list of p2p protocols.
func (s *Service) Protocols() []p2p.Protocol {
	return nil
}
