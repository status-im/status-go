package accounts

import (
	"errors"
	"time"

	"go.uber.org/zap"

	"github.com/ethereum/go-ethereum/rpc"

	"github.com/golang/protobuf/proto"

	accsmanagement "github.com/status-im/status-go/internal/accounts-management"
	accsmanagementtypes "github.com/status-im/status-go/internal/accounts-management/types"
	"github.com/status-im/status-go/internal/crypto"
	"github.com/status-im/status-go/internal/crypto/types"
	"github.com/status-im/status-go/internal/db/multiaccounts"
	"github.com/status-im/status-go/internal/db/multiaccounts/accounts"
	"github.com/status-im/status-go/internal/db/multiaccounts/settings"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/pkg/pubsub"
	"github.com/status-im/status-go/protocol"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/syncing"
	"github.com/status-im/status-go/server"
)

// NewService initializes service instance.
func NewService(
	db *accounts.Database,
	mdb *multiaccounts.Database,
	manager *accsmanagement.AccountsManager,
	config *params.NodeConfig,
	publisher *pubsub.Publisher,
	mediaServer *server.MediaServer,
	logger *zap.Logger,
) *Service {
	s := &Service{
		db:          db,
		mdb:         mdb,
		manager:     manager,
		config:      config,
		mediaServer: mediaServer,
		publisher:   publisher,
		logger:      logger,
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
	account     *multiaccounts.Account
	messenger   *protocol.Messenger
	mediaServer *server.MediaServer
	publisher   *pubsub.Publisher
	logger      *zap.Logger
}

func (s *Service) Init(messenger *protocol.Messenger, account *multiaccounts.Account) {
	s.messenger = messenger
	s.account = account
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

func (s *Service) GetKeypairByKeyUID(keyUID string) (*accsmanagementtypes.Keypair, error) {
	return s.db.GetKeypairByKeyUID(keyUID)
}

func (s *Service) GetSettings() (settings.Settings, error) {
	return s.db.GetSettings()
}

func (s *Service) GetBackupPath() (string, error) {
	return s.db.BackupPath()
}

func (s *Service) SetBackupPath(path string) error {
	return s.db.SaveSettingField(settings.BackupPath, path)
}

func (s *Service) GetMessenger() *protocol.Messenger {
	return s.messenger
}

func (s *Service) VerifyPassword(password string) (bool, error) {
	address, err := s.db.GetChatAddress()
	if err != nil {
		return false, err
	}
	return s.manager.VerifyAccountPassword(address, password)
}

func (s *Service) prepareSyncSettingsMessages(currentClock uint64, prepareForBackup bool) (resultSync []*protobuf.SyncSetting, errorResult error) {
	var errs []error
	dbSettings, err := s.db.GetSettings()
	if err != nil {
		errorResult = err
		return
	}

	for _, sf := range settings.SettingFieldRegister {
		if !sf.CanSync(settings.FromStruct) {
			continue
		}

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
	errorResult = errors.Join(errs...)
	return
}

func (s *Service) ExportBackup() ([]byte, error) {
	backup := &protobuf.AccountsLocalBackup{}

	settings, err := s.prepareSyncSettingsMessages(uint64(time.Now().UnixMilli()), true)
	if err != nil {
		return nil, err
	}
	backup.Settings = append(backup.Settings, settings...)

	return proto.Marshal(backup)
}

func (s *Service) handleBackedUpSettings(message *protobuf.SyncSetting) error {
	if message == nil {
		return nil
	}

	// DisplayName is recovered via `protobuf.BackedUpProfile` message
	if message.GetType() == protobuf.SyncSetting_DISPLAY_NAME {
		return nil
	}

	settingField, err := syncing.ExtractAndSaveSyncSetting(s.db, s.logger, message)
	if err != nil {
		s.logger.Warn("failed to handle SyncSetting from backed up message", zap.Error(err))
		return nil
	}

	if settingField != nil && s.account != nil {
		if message.GetType() == protobuf.SyncSetting_PREFERRED_NAME && message.GetValueString() != "" {
			displayNameClock, err := s.db.GetSettingLastSynced(settings.DisplayName)
			if err != nil {
				s.logger.Warn("failed to get last synced clock for display name", zap.Error(err))
				return nil
			}
			// there is a race condition between display name and preferred name on updating m.account.Name, so we need to check the clock
			// there is also a similar check within SaveSyncDisplayName
			if displayNameClock < message.GetClock() {
				s.account.Name = message.GetValueString()
				err = s.mdb.SaveAccount(*s.account)
				if err != nil {
					s.logger.Warn("[HandleBackedUpSettings] failed to save account", zap.Error(err))
					return nil
				}
			}
		}

		// TODO use a new feed for accounts service events
		s.messenger.PublishSettingEvent(settingField)
	}

	return nil
}

func (s *Service) ImportBackup(data []byte) error {
	var backup protobuf.AccountsLocalBackup
	err := proto.Unmarshal(data, &backup)
	if err != nil {
		return err
	}
	var errs []error

	for _, setting := range backup.Settings {
		err = s.handleBackedUpSettings(setting)
		if err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
