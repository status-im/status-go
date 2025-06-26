package accounts

import (
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/status-im/status-go/accounts-management/keystore/geth"
	"github.com/status-im/status-go/multiaccounts/settings"
	"github.com/status-im/status-go/pkg/pubsub"
	"github.com/status-im/status-go/server"

	accsmanagement "github.com/status-im/status-go/accounts-management"
	"github.com/status-im/status-go/multiaccounts"
	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/protocol"
)

// NewService initializes service instance.
func NewService(db *accounts.Database, mdb *multiaccounts.Database, manager *accsmanagement.AccountsManager,
	config *params.NodeConfig, publisher *pubsub.Publisher, mediaServer *server.MediaServer) *Service {
	s := &Service{
		db:          db,
		mdb:         mdb,
		manager:     manager,
		config:      config,
		mediaServer: mediaServer,
		publisher:   publisher,
	}
	db.SetSettingsNotifier(func(setting settings.SettingField, val interface{}) {
		if s.publisher != nil {
			pubsub.Publish(s.publisher, settings.EventSettingChanged{
				Setting: setting,
				Value:   val,
			})
		}
	})
	return s
}

// Service is a browsers service.
type Service struct {
	db          *accounts.Database
	mdb         *multiaccounts.Database
	manager     *accsmanagement.AccountsManager
	config      *params.NodeConfig
	messenger   *protocol.Messenger
	mediaServer *server.MediaServer
	publisher   *pubsub.Publisher
}

func (s *Service) Init(messenger *protocol.Messenger) {
	s.messenger = messenger
}

// Start a service.
func (s *Service) Start() error {
	keystoreAdapter, err := geth.NewGethKeystoreAdapter(s.config.KeyStoreDir, keystore.LightScryptN, keystore.LightScryptP)
	if err != nil {
		return err
	}
	s.manager.SetKeystore(keystoreAdapter)
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
			Service:   NewSettingsAPI(&s.messenger, s.db, s.config),
		},
		{
			Namespace: "accounts",
			Version:   "0.1.0",
			Service:   s.AccountsAPI(),
		},
		{
			Namespace: "multiaccounts",
			Version:   "0.1.0",
			Service:   NewMultiAccountsAPI(s.mdb, s.mediaServer),
		},
	}
}

func (s *Service) AccountsAPI() *API {
	return NewAccountsAPI(s.manager, s.config, s.db, &s.messenger, s.publisher)
}

func (s *Service) GetKeypairByKeyUID(keyUID string) (*accounts.Keypair, error) {

	return s.db.GetKeypairByKeyUID(keyUID)
}

func (s *Service) GetSettings() (settings.Settings, error) {
	return s.db.GetSettings()
}

func (s *Service) GetMessenger() *protocol.Messenger {
	return s.messenger
}

func (s *Service) VerifyPassword(password string) bool {
	address, err := s.db.GetChatAddress()
	if err != nil {
		return false
	}
	ok, err := s.manager.VerifyAccountPassword(address, password)
	return ok && err == nil
}
