package protocol

import (
	"crypto/ecdsa"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/status-im/status-go/accounts-management/generator"
	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/common/dbsetup"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/messaging"
	messagingtypes "github.com/status-im/status-go/messaging/types"
	"github.com/status-im/status-go/multiaccounts"
	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/multiaccounts/settings"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/protocol/encryption"
	"github.com/status-im/status-go/protocol/ens"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/sqlite"
	"github.com/status-im/status-go/protocol/tt"
	"github.com/status-im/status-go/services/browsers"
	"github.com/status-im/status-go/t/helpers"
	"github.com/status-im/status-go/walletdatabase"
)

type testMessengerConfig struct {
	name       string
	privateKey *ecdsa.PrivateKey
	logger     *zap.Logger

	unhandledMessagesTracker *unhandledMessagesTracker
	messagesOrderController  *MessagesOrderController

	appSettings  *settings.Settings
	nodeConfig   *params.NodeConfig
	extraOptions []Option
}

func (tmc *testMessengerConfig) complete() error {
	if len(tmc.name) == 0 {
		tmc.name = uuid.NewString()
	}

	if tmc.privateKey == nil {
		privateKey, err := crypto.GenerateKey()
		if err != nil {
			return err
		}
		tmc.privateKey = privateKey
	}

	if tmc.logger == nil {
		logger := tt.MustCreateTestLogger()
		tmc.logger = logger.Named(tmc.name)
	}

	if tmc.appSettings == nil {
		tmc.appSettings = newTestSettings()
	}

	if tmc.nodeConfig == nil {
		tmc.nodeConfig = &params.NodeConfig{}
	}

	return nil
}

func newTestMessenger(messagingEnv *messaging.TestMessagingEnvironment, config testMessengerConfig) (*Messenger, error) {
	err := config.complete()
	if err != nil {
		return nil, err
	}

	acc := generator.NewAccount(config.privateKey, nil)
	iai := acc.ToIdentifiedAccountInfo()
	multiAcc := &multiaccounts.Account{
		Timestamp: time.Now().Unix(),
		KeyUID:    iai.KeyUID,
	}

	madb, err := multiaccounts.InitializeDB(dbsetup.InMemoryPath)
	if err != nil {
		return nil, err
	}
	walletDb, err := helpers.SetupTestMemorySQLDB(walletdatabase.DbInitializer{})
	if err != nil {
		return nil, err
	}
	appDb, err := helpers.SetupTestMemorySQLDB(appdatabase.DbInitializer{})
	if err != nil {
		return nil, err
	}

	err = sqlite.Migrate(appDb)
	if err != nil {
		return nil, errors.Wrap(err, "failed to apply migrations")
	}

	if config.appSettings.Networks == nil {
		networks := new(json.RawMessage)
		if err := networks.UnmarshalJSON([]byte("net")); err != nil {
			return nil, err
		}

		config.appSettings.Networks = networks
	}

	sDB, err := accounts.NewDB(appDb)
	if err != nil {
		return nil, err
	}

	err = sDB.CreateSettings(*config.appSettings, *config.nodeConfig)
	if err != nil {
		return nil, err
	}

	installationID := uuid.New().String()

	// Initialize encryption layer.
	encryptionProtocol := encryption.New(
		appDb,
		installationID,
		config.logger,
	)

	messaging, err := messagingEnv.NewTestCore(
		config.privateKey,
		appDb,
		NewMessagingPersistence(appDb),
		encryptionProtocol,
		messaging.WithLogger(config.logger))
	if err != nil {
		return nil, err
	}

	ensVerifier := ens.New(
		config.logger,
		messaging.API(), // timesource
		appDb,
		"",
		"",
	)

	options := []Option{
		WithCustomLogger(config.logger),
		WithDatabase(appDb),
		WithWalletDatabase(walletDb),
		WithBrowserDatabase(browsers.NewDB(appDb)),
		WithMultiAccounts(madb),
		WithAccount(multiAcc),
		WithDatasync(),
		WithCuratedCommunitiesUpdateLoop(false),
		WithStubOnlineChecker(),
		WithENSVerifier(ensVerifier),
		WithMessageSigner(NewSignerStub()),
	}
	options = append(options, config.extraOptions...)

	m, err := NewMessenger(
		config.privateKey,
		messaging.API(),
		installationID,
		options...,
	)
	if err != nil {
		return nil, err
	}

	if config.unhandledMessagesTracker != nil {
		m.unhandledMessagesTracker = config.unhandledMessagesTracker.addMessage
	}

	if config.messagesOrderController != nil {
		m.retrievedMessagesIteratorFactory = config.messagesOrderController.newMessagesIterator
	}

	err = m.settings.SetUseMailservers(false)
	if err != nil {
		return nil, err
	}

	err = m.InitInstallations()
	if err != nil {
		return nil, err
	}

	err = m.InitFilters()
	if err != nil {
		return nil, err
	}

	return m, nil
}

func newRunningTestMessenger(messagingEnv *messaging.TestMessagingEnvironment, config testMessengerConfig) (*Messenger, error) {
	m, err := newTestMessenger(messagingEnv, config)
	if err != nil {
		return nil, err
	}

	m.EnableBackedupMessagesProcessing()

	_, err = m.Start()
	if err != nil {
		return nil, err
	}

	return m, nil
}

type unhandedMessage struct {
	*messagingtypes.Message
	err error
}

type unhandledMessagesTracker struct {
	messages map[protobuf.ApplicationMetadataMessage_Type][]*unhandedMessage
}

func (u *unhandledMessagesTracker) addMessage(msg *messagingtypes.Message, err error) {
	msgType := msg.ApplicationLayer.Type

	if _, exists := u.messages[msgType]; !exists {
		u.messages[msgType] = []*unhandedMessage{}
	}

	newMessage := &unhandedMessage{
		Message: msg,
		err:     err,
	}
	u.messages[msgType] = append(u.messages[msgType], newMessage)
}

func newTestSettings() *settings.Settings {
	return &settings.Settings{
		DisplayName:               DefaultProfileDisplayName,
		ProfilePicturesShowTo:     1,
		ProfilePicturesVisibility: 1,
		URLUnfurlingMode:          settings.URLUnfurlingAlwaysAsk,
	}
}
