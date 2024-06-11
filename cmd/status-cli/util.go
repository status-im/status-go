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

	return logutils.ZapLogger()
}

func start(name string, port int, apiModules string, telemetryUrl string, useLastAccount bool) (*StatusCLI, error) {
	var (
		rootDataDir = fmt.Sprintf("./test-%s", strings.ToLower(name))
		password    = "some-password"
	)
	setupLogger(name)
	nlog := logger.Named(name)
	nlog.Info("starting messager")

	backend := api.NewGethStatusBackend()
	if useLastAccount {
		if err := getLastAccountAndLogin(backend, name, rootDataDir, password); err != nil {
			return nil, err
		}
	} else {
		if err := createAccountAndLogin(backend, name, rootDataDir, password, apiModules, telemetryUrl, port); err != nil {
			return nil, err
		}
	}

	wakuService := backend.StatusNode().WakuV2ExtService()
	if wakuService == nil {
		return nil, errors.New("waku service is not available")
	}
	wakuAPI := wakuv2ext.NewPublicAPI(wakuService)

	messenger := wakuAPI.Messenger()
	if _, err := wakuAPI.StartMessenger(); err != nil {
		return nil, err
	}

	nlog.Info("messenger started, public key: ", messenger.IdentityPublicKeyString())
	time.Sleep(WaitingInterval)

	data := StatusCLI{
		name:      name,
		messenger: messenger,
		backend:   backend,
		logger:    nlog,
	}

	return &data, nil
}

func getLastAccountAndLogin(b *api.GethStatusBackend, name, rootDataDir, password string) error {
	b.UpdateRootDataDir(rootDataDir)
	if err := b.OpenAccounts(); err != nil {
		return fmt.Errorf("name '%v' might not have an account: trying to find: %v: %w", name, rootDataDir, err)
	}
	accs, err := b.GetAccounts()
	if err != nil {
		return err
	}
	if len(accs) == 0 {
		return errors.New("no accounts found")
	}

	return b.LoginAccount(&requests.Login{
		Password: password,
		KeyUID:   accs[0].KeyUID,
	})
}

func createAccountAndLogin(b *api.GethStatusBackend, name, rootDataDir, password, apiModules, telemetryUrl string, port int) error {

	if err := os.MkdirAll(rootDataDir, os.ModePerm); err != nil {
		return err
	}

	req := &requests.CreateAccount{
		DisplayName:        name,
		CustomizationColor: "#ffffff",
		Emoji:              "some",
		Password:           password,
		RootDataDir:        rootDataDir,
		LogFilePath:        "log",
		APIConfig: &requests.APIConfig{
			APIModules: apiModules,
			HTTPHost:   "127.0.0.1",
			HTTPPort:   port,
		},
		TelemetryServerURL: telemetryUrl,
	}
	if _, err := b.CreateAccountAndLogin(req); err != nil {
		return err
	}
	return nil
}

func (cli *StatusCLI) stop() {
	err := cli.backend.StopNode()
	if err != nil {
		logger.Error(err)
	}
}
