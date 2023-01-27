package accounts

import (
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/status-im/status-go/multiaccounts/settings"
	"github.com/status-im/status-go/server"

	"github.com/status-im/status-go/account"
	"github.com/status-im/status-go/multiaccounts"
	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/protocol"
)

// NewService initializes service instance.
func NewService(db *accounts.Database, mdb *multiaccounts.Database, manager *account.GethManager, config *params.NodeConfig, feed *event.Feed, mediaServer *server.MediaServer) *Service {
	return &Service{db, mdb, manager, config, feed, nil, mediaServer}
}

// Service is a browsers service.
type Service struct {
	db          *accounts.Database
	mdb         *multiaccounts.Database
	manager     *account.GethManager
	config      *params.NodeConfig
	feed        *event.Feed
	messenger   *protocol.Messenger
	mediaServer *server.MediaServer
}

func (s *Service) Init(messenger *protocol.Messenger) {
	s.messenger = messenger
}

// Start a service.
func (s *Service) Start() error {
	return s.manager.InitKeystore(s.config.KeyStoreDir)
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
			Service:   NewSettingsAPI(&s.messenger, s.db, s.config),
		},
		{
			Namespace: "accounts",
			Version:   "0.1.0",
			Service:   NewAccountsAPI(s.manager, s.config, s.db, s.feed, &s.messenger),
		},
		{
			Namespace: "multiaccounts",
			Version:   "0.1.0",
			Service:   NewMultiAccountsAPI(s.mdb, s.mediaServer),
		},
	}
}

// Protocols returns list of p2p protocols.
func (s *Service) Protocols() []p2p.Protocol {
	return nil
}

func (s *Service) GetAccountsByKeyUID(keyUID string) ([]*accounts.Account, error) {

	return s.db.GetAccountsByKeyUID(keyUID)
}

func (s *Service) GetSettings() (settings.Settings, error) {
	return s.db.GetSettings()
}

func (s *Service) GetMessenger() *protocol.Messenger {
	return s.messenger
}
