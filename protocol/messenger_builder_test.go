package protocol

import (
	"crypto/ecdsa"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel"
	"go.uber.org/zap"

	"github.com/status-im/status-go/accounts-management/generator"
	"github.com/status-im/status-go/common/dbsetup"
	"github.com/status-im/status-go/crypto"
	"github.com/status-im/status-go/internal/db/appdatabase"
	"github.com/status-im/status-go/internal/db/multiaccounts"
	"github.com/status-im/status-go/internal/db/multiaccounts/accounts"
	settings2 "github.com/status-im/status-go/internal/db/multiaccounts/settings"
	"github.com/status-im/status-go/internal/db/walletdatabase"
	"github.com/status-im/status-go/internal/instrumentation/trace"
	"github.com/status-im/status-go/internal/testutils"
	"github.com/status-im/status-go/messaging"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/ens"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/sqlite"
	"github.com/status-im/status-go/rpc/network"
	"github.com/status-im/status-go/services/browsers"
	"github.com/status-im/status-go/services/wallet/token"
	"github.com/status-im/status-go/t/helpers"
)

type testMessengerConfig struct {
	name       string
	privateKey *ecdsa.PrivateKey
	logger     *zap.Logger

	unhandledMessagesTracker *unhandledMessagesTracker
	messagesOrderController  *MessagesOrderController

	appSettings  *settings2.Settings
	nodeConfig   *params.NodeConfig
	extraOptions []Option
}

func (tmc *testMessengerConfig) complete() error {
	if len(tmc.name) == 0 {
		tmc.name = uuid.NewString()[0:6]
	}

	if tmc.privateKey == nil {
		privateKey, err := crypto.GenerateKey()
		if err != nil {
			return err
		}
		tmc.privateKey = privateKey
	}

	if tmc.logger == nil {
		logger := testutils.MustCreateTestLogger()
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

	messaging, err := messagingEnv.NewTestCore(
		messaging.CoreParams{
			Identity:       config.privateKey,
			InstallationID: installationID,
			TimeSource:     &testTimeSource{},
		},
		messaging.WithLogger(config.logger),
		messaging.WithTracer(trace.NewTracer(otel.Tracer("messaging_"+config.name))),
		messaging.WithSQLitePersistence(appDb),
	)
	if err != nil {
		return nil, err
	}

	ensVerifier := ens.New(
		config.logger,
		&testTimeSource{},
		appDb,
		"",
		"",
	)

	tokenManager, err := token.NewTokenManager(walletDb, nil, nil, network.NewManager(appDb, nil), appDb, nil, nil, nil,
		nil, time.Hour, time.Hour)
	if err != nil {
		return nil, err
	}

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
		WithTokenManager(tokenManager),
		WithTracer(trace.NewTracer(otel.Tracer("messenger_" + config.name))),
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

	return m, nil
}

func newRunningTestMessenger(messagingEnv *messaging.TestMessagingEnvironment, config testMessengerConfig) (*Messenger, error) {
	m, err := newTestMessenger(messagingEnv, config)
	if err != nil {
		return nil, err
	}

	err = m.messaging.Start()
	if err != nil {
		return nil, err
	}

	_, err = m.Start()
	if err != nil {
		return nil, err
	}

	return m, nil
}

type unhandedMessage struct {
	*common.StatusMessage
	err error
}

type unhandledMessagesTracker struct {
	messages map[protobuf.ApplicationMetadataMessage_Type][]*unhandedMessage
}

func (u *unhandledMessagesTracker) addMessage(msg *common.StatusMessage, err error) {
	msgType := msg.ApplicationLayer.Type

	if _, exists := u.messages[msgType]; !exists {
		u.messages[msgType] = []*unhandedMessage{}
	}

	newMessage := &unhandedMessage{
		StatusMessage: msg,
		err:           err,
	}
	u.messages[msgType] = append(u.messages[msgType], newMessage)
}

func newTestSettings() *settings2.Settings {
	return &settings2.Settings{
		DisplayName:               DefaultProfileDisplayName,
		ProfilePicturesShowTo:     1,
		ProfilePicturesVisibility: 1,
		URLUnfurlingMode:          settings2.URLUnfurlingAlwaysAsk,
	}
}
