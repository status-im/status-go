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
	"github.com/status-im/status-go/protocol/tt"
	"github.com/status-im/status-go/t/helpers"
	"github.com/status-im/status-go/walletdatabase"
)

type testMessengerConfig struct {
	name       string
	privateKey *ecdsa.PrivateKey
	logger     *zap.Logger
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
		tmc.logger = logger.With(zap.String("name", tmc.name))
	}

	return nil
}

func newTestMessenger(waku types.Waku, config testMessengerConfig, extraOptions []Option) (*Messenger, error) {
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
	options = append(options, extraOptions...)

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

	err = m.Init()
	if err != nil {
		return nil, err
	}

	return m, nil
}
