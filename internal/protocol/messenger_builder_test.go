package protocol

import (
	"crypto/ecdsa"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.uber.org/zap"

	"github.com/status-im/status-go/internal/accounts-management/generator"
	"github.com/status-im/status-go/internal/crypto"
	"github.com/status-im/status-go/internal/db/appdatabase"
	"github.com/status-im/status-go/internal/db/dbsetup"
	"github.com/status-im/status-go/internal/db/multiaccounts"
	"github.com/status-im/status-go/internal/db/multiaccounts/accounts"
	"github.com/status-im/status-go/internal/db/multiaccounts/settings"
	"github.com/status-im/status-go/internal/db/walletdb"
	"github.com/status-im/status-go/internal/instrumentation/trace"
	"github.com/status-im/status-go/internal/protocol/common"
	"github.com/status-im/status-go/internal/protocol/ens"
	"github.com/status-im/status-go/internal/protocol/protobuf"
	"github.com/status-im/status-go/internal/protocol/sqlite"
	"github.com/status-im/status-go/internal/rpc/network"
	networktestutil "github.com/status-im/status-go/internal/rpc/network/testutil"
	"github.com/status-im/status-go/internal/testutils"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/pkg/messaging"
	"github.com/status-im/status-go/pkg/services/browsers"
	"github.com/status-im/status-go/pkg/services/wallet/token"
)

type testMessengerConfig struct {
	name       string
	privateKey *ecdsa.PrivateKey
	logger     *zap.Logger

	unhandledMessagesTracker *unhandledMessagesTracker
	messagesOrderController  *MessagesOrderController

	appSettings      *settings.Settings
	nodeConfig       *params.NodeConfig
	extraOptions     []Option
	messagingOptions []messaging.Options
}

func (tmc *testMessengerConfig) complete(t *testing.T) error {
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
		t.Logf("created test logger - logger name: %s, test name: %s, ", tmc.name, t.Name())
		logger.Debug("created test logger", zap.String("name", tmc.name))
	}

	if tmc.appSettings == nil {
		tmc.appSettings = newTestSettings()
	}

	if tmc.nodeConfig == nil {
		tmc.nodeConfig = &params.NodeConfig{}
	}

	return nil
}

func newTestMessenger(t *testing.T, messagingEnv *messaging.TestMessagingEnvironment, config testMessengerConfig) (*Messenger, error) {
	err := config.complete(t)
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
	t.Cleanup(func() {
		err = madb.Close()
		assert.NoError(t, err)
	})

	walletDb, err := testutils.SetupTestMemorySQLDB(walletdb.DbInitializer{})
	if err != nil {
		return nil, err
	}
	t.Cleanup(func() {
		err = walletDb.Close()
		assert.NoError(t, err)
	})

	appDb, err := testutils.SetupTestMemorySQLDB(appdatabase.DbInitializer{})
	if err != nil {
		return nil, err
	}
	t.Cleanup(func() {
		err = appDb.Close()
		assert.NoError(t, err)
	})

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

	messagingCoreOptions := []messaging.Options{
		messaging.WithLogger(config.logger.Named("messaging")),
		messaging.WithTracer(trace.NewTracer(otel.Tracer("messaging_" + config.name))),
		messaging.WithSQLitePersistence(appDb),
	}
	messagingCoreOptions = append(messagingCoreOptions, config.messagingOptions...)

	messaging, err := messagingEnv.NewTestCore(
		messaging.CoreParams{
			Identity:       config.privateKey,
			InstallationID: installationID,
			TimeSource:     &testTimeSource{},
		},
		messagingCoreOptions...,
	)
	if err != nil {
		return nil, err
	}

	ensVerifier := ens.New(
		config.logger.Named("ens"),
		&testTimeSource{},
		appDb,
		"",
		"",
	)

	// TokenManager requires at least one active network.
	nm := network.NewManager(appDb, nil)
	err = nm.InitEmbeddedNetworks(networktestutil.MinimalActiveNetworks())
	if err != nil {
		return nil, err
	}

	tokenManager, err := token.NewTokenManager(walletDb, nil, nil, nm, appDb, nil, nil, nil,
		nil, time.Hour, time.Hour)
	if err != nil {
		return nil, err
	}

	options := []Option{
		WithCustomLogger(config.logger.Named("messenger")),
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

func newRunningTestMessenger(t *testing.T, messagingEnv *messaging.TestMessagingEnvironment, config testMessengerConfig) (*Messenger, error) {
	m, err := newTestMessenger(t, messagingEnv, config)
	require.NoError(t, err)

	err = m.messaging.Start()
	require.NoError(t, err)
	t.Cleanup(func() {
		err = m.messaging.Stop()
		assert.NoError(t, err)
	})

	_, err = m.Start()
	require.NoError(t, err)
	t.Cleanup(func() {
		err := m.Shutdown()
		assert.NoError(t, err)
	})

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

func newTestSettings() *settings.Settings {
	return &settings.Settings{
		DisplayName:                DefaultProfileDisplayName,
		ProfilePicturesShowTo:      1,
		ProfilePicturesVisibility:  1,
		URLUnfurlingMode:           settings.URLUnfurlingAlwaysAsk,
		AutoApplyKeypairMigrations: true,
	}
}
