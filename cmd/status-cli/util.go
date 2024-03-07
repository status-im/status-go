package main

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/status-im/status-go/api"
	"github.com/status-im/status-go/logutils"
	"github.com/status-im/status-go/protocol/requests"
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

	_ = setupLogger(name)

	path := fmt.Sprintf("./test-%s", strings.ToLower(name))
	err := os.MkdirAll(path, os.ModePerm)
	if err != nil {
		return nil, err
	}

	// "APIModules": "waku,wakuext,wakuv2,permissions,eth",
	// "HTTPEnabled": true,
	// "HTTPHost": "localhost",
	// "HTTPPort": 8545,
	// "WakuV2Config": {
	// 	"Enabled": true
	// },
	// "IPCEnabled": true,
	// "ListenAddr": ":30313"

	backend := api.NewGethStatusBackend()

	createAccountRequest := &requests.CreateAccount{
		DisplayName:           "some-display-name",
		CustomizationColor:    "#ffffff",
		Emoji:                 "some",
		Password:              "some-password",
		BackupDisabledDataDir: fmt.Sprintf("./test-%s", strings.ToLower(name)),
		NetworkID:             1,
		LogFilePath:           "log",
	}
	_, err = backend.CreateAccountAndLogin(createAccountRequest)
	if err != nil {
		return nil, err
	}

	wakuService := backend.StatusNode().WakuV2ExtService()
	if wakuService == nil {
		return nil, errors.New("waku service is not available")
	}
	wakuApi := wakuv2ext.NewPublicAPI(wakuService)
	messenger := wakuApi.Messenger()

	namedLogger.Info("messenger started, public key: ", messenger.IdentityPublicKeyString())

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
}
