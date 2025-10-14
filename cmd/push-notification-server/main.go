package main

import (
	"crypto/ecdsa"
	"database/sql"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/common/dbsetup"
	"github.com/status-im/status-go/crypto"
	"github.com/status-im/status-go/internal/timesource"
	"github.com/status-im/status-go/logutils"
	"github.com/status-im/status-go/messaging"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/pkg/sentry"
	"github.com/status-im/status-go/pkg/version"
	"github.com/status-im/status-go/protocol"
	"github.com/status-im/status-go/protocol/pushnotificationserver"
	"github.com/status-im/status-go/protocol/sqlite"
	mailserversDB "github.com/status-im/status-go/services/mailservers"
	"github.com/status-im/status-go/services/personal"
	"github.com/status-im/status-go/walletdatabase"
)

var (
	gorushURL       = flag.String("gorush-url", pushnotificationserver.DefaultGorushURL, "gorush server URL")
	dataDir         = flag.String("data-dir", "", "data directory")
	identity        = flag.String("identity", "", "Hex-encoded private key to use. When empty, an ephemeral key will be used")
	sentryEnabled   = flag.Bool("sentry", false, "Enable sentry panic reporting")
	logLevel        = flag.String("log-level", "INFO", `Log level, one of: "ERROR", "WARN", "INFO", "DEBUG"`)
	logNoColors     = flag.Bool("log-no-color", false, "Disables log colors")
	wakuFleet       = flag.String("waku-fleet", "status.prod", "Waku fleet to use")
	wakuFleetConfig = flag.String("waku-fleet-config", "", "path to Waku fleet config file")
	Port            = flag.Int("tcp-port", 30303, "Libp2p TCP port")
	UDPPort         = flag.Int("udp-port", 30303, "Libp2p UDP port")
	// TODO: add pprof and metrics

	logger *zap.Logger
)

const (
	exitCodeOK = iota
	exitCodeInvalidWakuFleetConfig
	exitCodeInvalidKey
	exitCodeCreateWakuFailed
	exitCodeStartMessagingFailed
	exitCodeDBMigrationFailed
	exitCodeCreateMessengerFailed
	exitCodeCreateDatabaseFailed
	exitCodeStartServerFailed
	exitCodeStartMessengerFailed
)

func init() {
	flag.Parse()
	logSettings := logutils.LogSettings{
		Enabled: true,
		Level:   *logLevel,
	}
	if err := logutils.OverrideRootLoggerWithConfig(logSettings); err != nil {
		panic(err)
	}

	logger = logutils.ZapLogger()
}

func main() {
	defer func() {
		_ = logger.Sync()
	}()

	if *sentryEnabled {
		sentry.MustInit(
			sentry.WithDefaultEnvironmentDSN(),
			sentry.WithContext("push-notification-server", version.Version()),
		)
		defer sentry.Recover()
	}

	if *wakuFleetConfig != "" {
		err := params.LoadWakuFleetsFromFile(*wakuFleetConfig)
		if err != nil {
			logger.Error("failed to load waku fleet config", zap.Error(err))
			os.Exit(exitCodeInvalidWakuFleetConfig)
		}
	}

	privateKey, installationID, err := parseNodeKey(*identity)
	if err != nil {
		logger.Error("failed to parse node key", zap.Error(err))
		os.Exit(exitCodeInvalidKey)
	}

	dbPath := path.Join(*dataDir, installationID+"-app")
	db, err := createAppDatabase(dbPath)
	if err != nil {
		logger.Error("failed to create database", zap.Error(err))
		os.Exit(exitCodeCreateDatabaseFailed)
	}

	walletDBPath := path.Join(*dataDir, installationID)
	walletDB, err := createWalletDatabase(walletDBPath)
	if err != nil {
		logger.Error("failed to create database", zap.Error(err))
		os.Exit(exitCodeCreateDatabaseFailed)
	}

	err = sqlite.Migrate(db)
	if err != nil {
		logger.Error("failed to migrate database", zap.Error(err))
		os.Exit(exitCodeDBMigrationFailed)
	}

	messaging, err := messaging.NewCore(
		messaging.CoreParams{
			Identity:    privateKey,
			DB:          db,
			Persistence: protocol.NewMessagingPersistence(db),
			NodeKey:     nil,
			WakuConfig: params.WakuV2Config{
				Enabled:        true,
				Host:           "0.0.0.0",
				Port:           *Port,
				UDPPort:        *UDPPort,
				LightClient:    false,
				DiscoveryLimit: 20,
				AutoUpdate:     true,
			},
			ClusterConfig: params.ClusterConfig{
				WakuNodes:            params.DefaultWakuNodes(*wakuFleet),
				DiscV5BootstrapNodes: params.DefaultDiscV5Nodes(*wakuFleet),
				ClusterID:            16,
			},
			InstallationID: installationID,
			TimeSource:     timesource.DefaultService(),
		},
		messaging.WithLogger(logger.Named("messaging")),
	)
	if err != nil {
		os.Exit(exitCodeCreateMessengerFailed)
	}

	err = messaging.API().Start()
	if err != nil {
		logger.Error("failed to start messaging", zap.Error(err))
		os.Exit(exitCodeStartMessagingFailed)
	}
	defer func() {
		err := messaging.API().Stop()
		if err != nil {
			logger.Error("failed to stop messaging", zap.Error(err))
		}
	}()

	// Set up the push notifications server
	config := &pushnotificationserver.Config{
		Enabled:   true,
		Identity:  privateKey,
		GorushURL: *gorushURL,
		Logger:    logger,
	}
	server := pushnotificationserver.New(config)

	// Set up the messenger
	options := []protocol.Option{
		protocol.WithDatabase(db),
		protocol.WithWalletDatabase(walletDB),
		protocol.WithMailserversDatabase(mailserversDB.NewDB(db)),
		protocol.WithDatasync(),
		protocol.WithMessageSigner(personal.New()),
		protocol.WithPushNotificationServer(server),
	}
	messenger, err := protocol.NewMessenger(privateKey, messaging.API(), installationID, options...)
	if err != nil {
		logger.Error("failed to create messenger", zap.Error(err))
		os.Exit(exitCodeCreateMessengerFailed)
	}

	// Start
	serverPersistence := pushnotificationserver.NewSQLitePersistence(db)
	err = server.Start(serverPersistence, messenger.Messaging())
	if err != nil {
		logger.Error("failed to start push notifications server", zap.Error(err))
		os.Exit(exitCodeStartServerFailed)
	}

	_, err = messenger.Start()
	if err != nil {
		fmt.Println("failed to start messenger", err)
		logger.Error("failed to start messenger", zap.Error(err))
		os.Exit(exitCodeStartMessengerFailed)
	}

	defer func() {
		err := messenger.Shutdown()
		if err != nil {
			logger.Error("failed to shutdown messenger", zap.Error(err))
		}
	}()

	cancelMessenger := make(chan struct{})
	messenger.StartRetrieveMessagesLoop(300*time.Millisecond, cancelMessenger)

	go func() {
		select {
		case <-cancelMessenger:
			return
		case <-time.After(10 * time.Second):
		}
		logger.Info("requesting history")
		response, err := messenger.RequestAllHistoricMessages(false)
		if err != nil {
			logger.Error("failed to request history", zap.Error(err))
			return
		}

		logger.Info("history fetched",
			zap.Any("response", response),
		)
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit
	close(cancelMessenger)

	logger.Info("push-notification-server finished")
	os.Exit(exitCodeOK)
}

func parseNodeKey(nodeKey string) (*ecdsa.PrivateKey, string, error) {
	// Check for environment variable if CLI flag is empty
	if nodeKey == "" {
		nodeKey = os.Getenv("STATUS_GO_NODE_KEY")
	}

	// If still empty, return error
	if nodeKey == "" {
		return nil, "", errors.New("Nodekey must be provided via -identity flag or STATUS_GO_NODE_KEY environment variable")
	}

	// Parse private key
	privateKey, err := crypto.HexToECDSA(nodeKey)
	if err != nil {
		return nil, "", errors.Wrap(err, "invalid node key")
	}

	// Generate installationID from public key, so it's always the same
	installationID, err := uuid.FromBytes(crypto.CompressPubkey(&privateKey.PublicKey)[:16])
	if err != nil {
		return nil, "", errors.Wrap(err, "failed to generate installation id")
	}

	return privateKey, installationID.String(), nil
}

func createAppDatabase(path string) (*sql.DB, error) {
	filename := path + ".db"
	appDB, err := appdatabase.InitializeDB(filename, "", dbsetup.ReducedKDFIterationsNumber)
	if err != nil {
		return nil, errors.Wrap(err, "failed to initialize app database")
	}

	return appDB, nil
}

func createWalletDatabase(path string) (*sql.DB, error) {
	filename := path + "-wallet.db"
	walletDB, err := walletdatabase.InitializeDB(filename, "", dbsetup.ReducedKDFIterationsNumber)
	if err != nil {
		logger.Error("failed to initialize wallet db", zap.Error(err))
		return nil, err
	}
	return walletDB, nil
}
