package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/status-im/status-go/api"
	"github.com/status-im/status-go/logutils"
	"github.com/status-im/status-go/multiaccounts"
	"github.com/status-im/status-go/protocol/requests"
	"github.com/status-im/status-go/services/wakuv2ext"
	"github.com/status-im/status-go/telemetry"

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

func start(name string, port int, apiModules string, telemetryUrl string, useExistingAccount bool, keyUID string) (*StatusCLI, error) {
	var (
		rootDataDir = fmt.Sprintf("./test-%s", strings.ToLower(name))
		password    = "some-password"
	)
	setupLogger(name)
	nlog := logger.Named(name)
	nlog.Info("starting messenger")

	backend := api.NewGethStatusBackend()
	if useExistingAccount {
		if err := getAccountAndLogin(backend, name, rootDataDir, password, keyUID); err != nil {
			return nil, err
		}
	} else {
		acc, err := createAccountAndLogin(backend, name, rootDataDir, password, apiModules, telemetryUrl, port)
		if err != nil {
			return nil, err
		}
		nlog.Infof("account created, key UID: %v", acc.KeyUID)
	}

	wakuService := backend.StatusNode().WakuV2ExtService()
	if wakuService == nil {
		return nil, errors.New("waku service is not available")
	}

	if telemetryUrl != "" {
		telemetryClient := telemetry.NewClient(nlog.Desugar(), telemetryUrl, backend.SelectedAccountKeyID(), name, "cli")
		go telemetryClient.Start(context.Background())
		backend.StatusNode().WakuV2Service().SetStatusTelemetryClient(telemetryClient)
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

func getAccountAndLogin(b *api.GethStatusBackend, name, rootDataDir, password string, keyUID string) error {
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

	acc := accs[0] // use last if no keyUID is provided
	if keyUID != "" {
		found := false
		for _, a := range accs {
			if a.KeyUID == keyUID {
				acc = a
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("account not found for keyUID: %v", keyUID)
		}
	}

	return b.LoginAccount(&requests.Login{
		Password: password,
		KeyUID:   acc.KeyUID,
	})
}

func createAccountAndLogin(b *api.GethStatusBackend, name, rootDataDir, password, apiModules, telemetryUrl string, port int) (*multiaccounts.Account, error) {
	if err := os.MkdirAll(rootDataDir, os.ModePerm); err != nil {
		return nil, err
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
	return b.CreateAccountAndLogin(req)
}

func (cli *StatusCLI) stop() {
	err := cli.backend.StopNode()
	if err != nil {
		logger.Error(err)
	}
}
