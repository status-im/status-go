package accounts

import (
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/status-im/status-go/account"
	"github.com/status-im/status-go/multiaccounts"
	"github.com/status-im/status-go/multiaccounts/accounts"
)

// NewService initializes service instance.
func NewService(db *accounts.Database, mdb *multiaccounts.Database, manager *account.Manager, feed *event.Feed) *Service {
	return &Service{db, mdb, manager, feed}
}

// Service is a browsers service.
type Service struct {
	db      *accounts.Database
	mdb     *multiaccounts.Database
	manager *account.Manager
	feed    *event.Feed
}

// Start a service.
func (s *Service) Start() error {
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
			Service:   NewSettingsAPI(s.db),
		},
		{
			Namespace: "accounts",
			Version:   "0.1.0",
			Service:   NewAccountsAPI(s.db, s.feed),
		},
		{
			Namespace: "multiaccounts",
			Version:   "0.1.0",
			Service:   NewMultiAccountsAPI(s.mdb, s.manager),
		},
	}
}

// Protocols returns list of p2p protocols.
func (s *Service) Protocols() []p2p.Protocol {
	return nil
}
