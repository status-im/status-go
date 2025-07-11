package accounts

import (
	"errors"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/golang/protobuf/proto"

	accsmanagement "github.com/status-im/status-go/accounts-management"
	"github.com/status-im/status-go/accounts-management/keystore/geth"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/multiaccounts"
	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/multiaccounts/settings"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/pkg/pubsub"
	"github.com/status-im/status-go/protocol"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/server"
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

// Service is an accounts service.
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
	fmt.Println("Initializing accounts service with messenger")
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

func (s *Service) GetBackupPath() (string, error) {
	return s.db.BackupPath()
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

func (s *Service) prepareSyncSettingsMessages(currentClock uint64, prepareForBackup bool) (resultSync []*protobuf.SyncSetting, errorResult error) {
	var errs []error
	dbSettings, err := s.db.GetSettings()
	if err != nil {
		errorResult = err
		return
	}

	for _, sf := range settings.SettingFieldRegister {
		if sf.CanSync(settings.FromStruct) {
			// DisplayName is backed up via `protobuf.BackedUpProfile` message.
			if prepareForBackup && sf.SyncProtobufFactory().SyncSettingProtobufType() == protobuf.SyncSetting_DISPLAY_NAME {
				continue
			}

			// Pull clock from the db
			clock, err := s.db.GetSettingLastSynced(sf)
			if err != nil {
				errorResult = err
				return
			}
			if clock == 0 {
				clock = currentClock
			}

			// Build protobuf
			_, sm, err := sf.SyncProtobufFactory().FromStruct()(dbSettings, clock, types.EncodeHex(crypto.FromECDSAPub(s.messenger.IdentityPublicKey())))
			if err != nil {
				// Collect errors to give other sync messages a chance to send
				errs = append(errs, err)
			}

			resultSync = append(resultSync, sm)
		}
	}
	errorResult = errors.Join(errs...)
	return
}

func (s *Service) ExportBackup() ([]byte, error) {
	backup := &protobuf.AccountsLocalBackup{}

	// TODO is using this for the clock ok?
	settings, err := s.prepareSyncSettingsMessages(uint64(time.Now().UnixMilli()), true)
	if err != nil {
		return nil, err
	}
	backup.Settings = append(backup.Settings, settings...)

	return proto.Marshal(backup)
}

func (s *Service) ImportBackup(data []byte) error {
	var backup protobuf.AccountsLocalBackup
	err := proto.Unmarshal(data, &backup)
	if err != nil {
		return err
	}
	var errs []error

	for _, setting := range backup.Settings {
		// TODO is it ok to use the messenger here? Otherwise, I have to duplicate a lot of code
		err = s.messenger.HandleBackedUpSettings(setting)
		if err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
