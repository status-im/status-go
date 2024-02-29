package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/status-im/status-go/account/generator"
	"github.com/status-im/status-go/api"
	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/common/dbsetup"
	gethbridge "github.com/status-im/status-go/eth-node/bridge/geth"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/logutils"
	"github.com/status-im/status-go/multiaccounts"
	"github.com/status-im/status-go/protocol"
	"github.com/status-im/status-go/t/helpers"
	"github.com/status-im/status-go/wakuv2"
	"github.com/status-im/status-go/walletdatabase"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
)

func setupLogger(file string) *zap.Logger {
	logFile := fmt.Sprintf("%s.log", strings.ToLower(file))
	logSettings := logutils.LogSettings{
		Enabled:         true,
		MobileSystem:    false,
		Level:           "DEBUG",
		File:            logFile,
		MaxSize:         100,
		MaxBackups:      3,
		CompressRotated: true,
	}
	if err := logutils.OverrideRootLogWithConfig(logSettings, false); err != nil {
		logger.Fatalf("Error initializing logger: %v", err)
	}

	newLogger := logutils.ZapLogger()

	return newLogger
}

func startMessenger(cCtx *cli.Context, name string) (*StatusCLI, error) {
	namedLogger := logger.Named(name)
	namedLogger.Info("starting messager")

	userLogger := setupLogger(name)

	privateKey, err := crypto.GenerateKey()
	if err != nil {
		return nil, err
	}

	config := &wakuv2.Config{}
	config.EnableDiscV5 = true
	config.DiscV5BootstrapNodes = api.DefaultWakuNodes[api.DefaultFleet]
	config.DiscoveryLimit = 20
	config.LightClient = cCtx.Bool(LightFlag)
	node, err := wakuv2.New("", "", config, userLogger, nil, nil, nil, nil)
	if err != nil {
		return nil, err
	}

	err = node.Start()
	if err != nil {
		return nil, err
	}

	appDb, err := helpers.SetupTestMemorySQLDB(appdatabase.DbInitializer{})
	if err != nil {
		return nil, err
	}
	walletDb, err := helpers.SetupTestMemorySQLDB(walletdatabase.DbInitializer{})
	if err != nil {
		return nil, err
	}
	madb, err := multiaccounts.InitializeDB(dbsetup.InMemoryPath)
	if err != nil {
		return nil, err
	}
	acc := generator.NewAccount(privateKey, nil)
	iai := acc.ToIdentifiedAccountInfo("")
	opitons := []protocol.Option{
		protocol.WithCustomLogger(userLogger),
		protocol.WithDatabase(appDb),
		protocol.WithWalletDatabase(walletDb),
		protocol.WithMultiAccounts(madb),
		protocol.WithAccount(iai.ToMultiAccount()),
		protocol.WithDatasync(),
		protocol.WithToplevelDatabaseMigrations(),
		protocol.WithBrowserDatabase(nil),
	}
	messenger, err := protocol.NewMessenger(
		fmt.Sprintf("%s-node", strings.ToLower(name)),
		privateKey,
		gethbridge.NewNodeBridge(nil, nil, node),
		uuid.New().String(),
		nil,
		opitons...,
	)
	if err != nil {
		return nil, err
	}

	err = messenger.Init()
	if err != nil {
		return nil, err
	}

	_, err = messenger.Start()
	if err != nil {
		return nil, err
	}

	id := types.EncodeHex(crypto.FromECDSAPub(messenger.IdentityPublicKey()))
	namedLogger.Info("messenger started, public key: ", id)

	time.Sleep(WaitingInterval)

	data := StatusCLI{
		name:      name,
		messenger: messenger,
		waku:      node,
		logger:    namedLogger,
	}

	return &data, nil
}

func stopMessenger(cli *StatusCLI) {
	err := cli.messenger.Shutdown()
	if err != nil {
		logger.Error(err)
	}

	err = cli.waku.Stop()
	if err != nil {
		logger.Error(err)
	}
}
