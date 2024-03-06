package main

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/status-im/status-go/api"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/logutils"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/services/wakuv2ext"
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

	// userLogger := setupLogger(name)

	// privateKey, err := crypto.GenerateKey()
	// if err != nil {
	// 	return nil, err
	// }

	// config := &wakuv2.Config{}
	// config.EnableDiscV5 = true
	// config.DiscV5BootstrapNodes = api.DefaultWakuNodes[api.DefaultFleet]
	// config.DiscoveryLimit = 20
	// config.LightClient = cCtx.Bool(LightFlag)
	// node, err := wakuv2.New("", "", config, userLogger, nil, nil, nil, nil)
	// if err != nil {
	// 	return nil, err
	// }

	// err = node.Start()
	// if err != nil {
	// 	return nil, err
	// }

	// appDb, err := helpers.SetupTestMemorySQLDB(appdatabase.DbInitializer{})
	// if err != nil {
	// 	return nil, err
	// }
	// walletDb, err := helpers.SetupTestMemorySQLDB(walletdatabase.DbInitializer{})
	// if err != nil {
	// 	return nil, err
	// }
	// madb, err := multiaccounts.InitializeDB(dbsetup.InMemoryPath)
	// if err != nil {
	// 	return nil, err
	// }
	// acc := generator.NewAccount(privateKey, nil)
	// iai := acc.ToIdentifiedAccountInfo("")
	// opitons := []protocol.Option{
	// 	protocol.WithCustomLogger(userLogger),
	// 	protocol.WithDatabase(appDb),
	// 	protocol.WithWalletDatabase(walletDb),
	// 	protocol.WithMultiAccounts(madb),
	// 	protocol.WithAccount(iai.ToMultiAccount()),
	// 	protocol.WithDatasync(),
	// 	protocol.WithToplevelDatabaseMigrations(),
	// 	protocol.WithBrowserDatabase(nil),
	// }
	// messenger, err := protocol.NewMessenger(
	// 	fmt.Sprintf("%s-node", strings.ToLower(name)),
	// 	privateKey,
	// 	gethbridge.NewNodeBridge(nil, nil, node),
	// 	uuid.New().String(),
	// 	nil,
	// 	opitons...,
	// )
	// if err != nil {
	// 	return nil, err
	// }

	nodeConfig, err := params.NewNodeConfigWithDefaultsAndFiles(
		fmt.Sprintf("./test-%s", name),
		params.GoerliNetworkID,
		[]params.Option{},
		[]string{},
	)
	if err != nil {
		return nil, err
	}
	nodeConfig.Name = fmt.Sprintf("%s-status-cli", name)

	// "APIModules": "waku,wakuext,wakuv2,permissions,eth",
	// "HTTPEnabled": true,
	// "HTTPHost": "localhost",
	// "HTTPPort": 8545,
	// "WakuV2Config": {
	// 	"Enabled": true
	// },
	// "IPCEnabled": true,
	// "ListenAddr": ":30313"

	nodeConfig.APIModules = "waku,wakuext,wakuv2,permissions,eth"
	nodeConfig.HTTPEnabled = true
	nodeConfig.HTTPHost = "localhost"
	nodeConfig.HTTPPort = 8545
	nodeConfig.WakuV2Config.Enabled = true

	namedLogger.Info("config", nodeConfig)

	backend := api.NewGethStatusBackend()
	err = backend.AccountManager().InitKeystore(nodeConfig.KeyStoreDir)
	if err != nil {
		return nil, err
	}

	err = backend.StartNode(nodeConfig)
	if err != nil {
		return nil, err
	}

	wakuService := backend.StatusNode().WakuV2ExtService()
	if wakuService == nil {
		return nil, errors.New("waku service is not available")
	}
	wakuApi := wakuv2ext.NewPublicAPI(wakuService)

	// _, err = wakuApi.StartMessenger()
	// if err != nil {
	// 	return nil, err
	// }
	messenger := wakuApi.Messenger()

	// err = backend.StartNode(nodeConfig)
	// if err != nil {
	// 	return nil, err
	// }

	// err = messenger.Init()
	// if err != nil {
	// 	return nil, err
	// }

	// _, err = messenger.Start()
	// if err != nil {
	// 	return nil, err
	// }

	id := types.EncodeHex(crypto.FromECDSAPub(messenger.IdentityPublicKey()))
	namedLogger.Info("messenger started, public key: ", id)

	time.Sleep(WaitingInterval)

	data := StatusCLI{
		name:      name,
		messenger: messenger,
		backend:   backend,
		logger:    namedLogger,
	}

	return &data, nil
}

func stopMessenger(cli *StatusCLI) {
	err := cli.backend.StopNode()
	if err != nil {
		logger.Error(err)
	}

	// err = cli.waku.Stop()
	// if err != nil {
	// 	logger.Error(err)
	// }
}
