package protocol

import (
	"crypto/ecdsa"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/status-im/status-go/account/generator"
	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/common/dbsetup"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/multiaccounts"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/tt"
	v1protocol "github.com/status-im/status-go/protocol/v1"
	"github.com/status-im/status-go/t/helpers"
	"github.com/status-im/status-go/walletdatabase"
)

type testMessengerConfig struct {
	name       string
	privateKey *ecdsa.PrivateKey
	logger     *zap.Logger

	unhandledMessagesTracker *unhandledMessagesTracker
	extraOptions             []Option
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

	return nil
}

func newTestMessenger(waku types.Waku, config testMessengerConfig) (*Messenger, error) {
	err := config.complete()
	if err != nil {
		return nil, err
	}

	acc := generator.NewAccount(config.privateKey, nil)
	iai := acc.ToIdentifiedAccountInfo("")

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

	options := []Option{
		WithCustomLogger(config.logger),
		WithDatabase(appDb),
		WithWalletDatabase(walletDb),
		WithMultiAccounts(madb),
		WithAccount(iai.ToMultiAccount()),
		WithDatasync(),
		WithToplevelDatabaseMigrations(),
		WithBrowserDatabase(nil),
	}
	options = append(options, config.extraOptions...)

	m, err := NewMessenger(
		config.name,
		config.privateKey,
		&testNode{shh: waku},
		uuid.New().String(),
		nil,
		options...,
	)
	if err != nil {
		return nil, err
	}

	if config.unhandledMessagesTracker != nil {
		m.unhandledMessagesTracker = config.unhandledMessagesTracker.addMessage
	}

	err = m.Init()
	if err != nil {
		return nil, err
	}

	return m, nil
}

type unhandedMessage struct {
	*v1protocol.StatusMessage
	err error
}

type unhandledMessagesTracker struct {
	messages map[protobuf.ApplicationMetadataMessage_Type][]*unhandedMessage
}

func (u *unhandledMessagesTracker) addMessage(msg *v1protocol.StatusMessage, err error) {
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
