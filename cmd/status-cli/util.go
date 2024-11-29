package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/status-im/status-go/api"
	"github.com/status-im/status-go/logutils"
	"github.com/status-im/status-go/multiaccounts"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/protocol/requests"
	"github.com/status-im/status-go/services/wakuv2ext"
	"github.com/status-im/status-go/telemetry"

	"github.com/urfave/cli/v2"
)

func setupLogger(file string) *zap.Logger {
	logFile := fmt.Sprintf("%s.log", strings.ToLower(file))
	logSettings := logutils.LogSettings{
		Enabled:         true,
		Level:           "DEBUG",
		File:            logFile,
		MaxSize:         100,
		MaxBackups:      3,
		CompressRotated: true,
	}
	if err := logutils.OverrideRootLoggerWithConfig(logSettings); err != nil {
		zap.S().Fatalf("Error initializing logger: %v", err)
	}
	return logutils.ZapLogger()
}

type StartParams struct {
	Name         string
	Port         int
	APIModules   string
	TelemetryURL string
	KeyUID       string
	Fleet        string
}

func start(p StartParams, logger *zap.SugaredLogger) (*StatusCLI, error) {
	var (
		rootDataDir = fmt.Sprintf("./test-%s", strings.ToLower(p.Name))
		password    = "some-password"
	)
	setupLogger(p.Name)
	logger.Info("starting messenger")

	backend := api.NewGethStatusBackend(logutils.ZapLogger())
	if p.KeyUID != "" {
		if err := getAccountAndLogin(backend, p.Name, rootDataDir, password, p.KeyUID); err != nil {
			return nil, err
		}
		logger.Infof("existing account, key UID: %v", p.KeyUID)
	} else {
		acc, err := createAccountAndLogin(backend, rootDataDir, password, p)
		if err != nil {
			return nil, err
		}
		logger.Infof("account created, key UID: %v", acc.KeyUID)
	}

	wakuService := backend.StatusNode().WakuV2ExtService()
	if wakuService == nil {
		return nil, errors.New("waku service is not available")
	}

	if p.TelemetryURL != "" {
		telemetryLogger, err := getLogger(true)
		if err != nil {
			return nil, err
		}
		waku := backend.StatusNode().WakuV2Service()
		telemetryClient := telemetry.NewClient(telemetryLogger, p.TelemetryURL, backend.SelectedAccountKeyID(), p.Name, "cli", telemetry.WithPeerID(waku.PeerID().String()))
		telemetryClient.Start(context.Background())
		backend.StatusNode().WakuV2Service().SetStatusTelemetryClient(telemetryClient)
	}
	wakuAPI := wakuv2ext.NewPublicAPI(wakuService)

	messenger := wakuAPI.Messenger()
	if _, err := wakuAPI.StartMessenger(); err != nil {
		return nil, err
	}

	logger.Info("messenger started, public key: ", messenger.IdentityPublicKeyString())
	time.Sleep(WaitingInterval)

	data := StatusCLI{
		name:      p.Name,
		messenger: messenger,
		backend:   backend,
		logger:    logger,
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

	var acc multiaccounts.Account
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

	return b.LoginAccount(&requests.Login{
		Password: password,
		KeyUID:   acc.KeyUID,
	})
}

func createAccountAndLogin(b *api.GethStatusBackend, rootDataDir, password string, p StartParams) (*multiaccounts.Account, error) {
	if err := os.MkdirAll(rootDataDir, os.ModePerm); err != nil {
		return nil, err
	}

	req := &requests.CreateAccount{
		DisplayName:        p.Name,
		CustomizationColor: "#ffffff",
		Password:           password,
		RootDataDir:        rootDataDir,
		LogFilePath:        "log",
		APIConfig: &requests.APIConfig{
			APIModules:  p.APIModules,
			HTTPEnabled: true,
			HTTPHost:    "127.0.0.1",
			HTTPPort:    p.Port,
		},
		TelemetryServerURL: p.TelemetryURL,
	}
	return b.CreateAccountAndLogin(req,
		params.WithFleet(p.Fleet),
		params.WithDiscV5BootstrapNodes(params.DefaultDiscV5Nodes(p.Fleet)),
		params.WithWakuNodes(params.DefaultWakuNodes(p.Fleet)),
	)
}

func (cli *StatusCLI) stop() {
	err := cli.backend.StopNode()
	if err != nil {
		cli.logger.Error(err)
	}
}

func getLogger(debug bool) (*zap.Logger, error) {
	at := zap.NewAtomicLevel()
	if debug {
		at.SetLevel(zap.DebugLevel)
	} else {
		at.SetLevel(zap.InfoLevel)
	}
	config := zap.NewDevelopmentConfig()
	config.Level = at
	rawLogger, err := config.Build()
	if err != nil {
		return nil, fmt.Errorf("initializing logger: %v", err)
	}
	return rawLogger, nil
}

func getSLogger(debug bool) (*zap.SugaredLogger, error) {
	l, err := getLogger(debug)
	if err != nil {
		return nil, err
	}
	return l.Sugar(), nil
}

func flagsUsed(cCtx *cli.Context) string {
	var sb strings.Builder
	for _, flag := range cCtx.Command.Flags {
		if flag != nil && len(flag.Names()) > 0 {
			fName := flag.Names()[0]
			fmt.Fprintf(&sb, "\t-%s %v\n", fName, cCtx.Value(fName))
		}
	}
	return sb.String()
}
