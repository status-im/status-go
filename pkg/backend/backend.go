package backend

import (
	"database/sql"
	"os"
	"path"
	"path/filepath"

	gethrpc "github.com/ethereum/go-ethereum/rpc"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/status-im/status-go/centralizedmetrics"
	"github.com/status-im/status-go/internal/metrics"
	"github.com/status-im/status-go/ipfs"
	"github.com/status-im/status-go/logutils"
	"github.com/status-im/status-go/logutils/requestlog"
	"github.com/status-im/status-go/multiaccounts"
	"github.com/status-im/status-go/pkg/backend/rpc"
	"github.com/status-im/status-go/pkg/sentry"
	"github.com/status-im/status-go/pkg/services/root"
	"github.com/status-im/status-go/pkg/version"
	"github.com/status-im/status-go/server"
)

const (
	DefaultAPILogFile = "api.log"
)

type StatusBackend struct {
	rootDataDir string
	logger      *zap.Logger

	// Databases
	rootDB          *sql.DB // aka multiaccounts db, aka 'accounts.sql'
	appDB           *sql.DB
	walletDB        *sql.DB
	multiaccountsDB *multiaccounts.Database // FIXME: Remove this pointer

	// RPC server
	rpcServer *gethrpc.Server

	// Services
	rootService *root.Service

	// FIXME: Extract to separate services
	ipfs               *ipfs.Downloader
	mediaServer        *server.MediaServer
	centralizedMetrics *centralizedmetrics.MetricService
	prometheusMetrics  *metrics.Server
}

func NewStatusBackend(rootDataDir string, opts ...Option) (*StatusBackend, error) {
	cfg := config{
		logger:            zap.NewNop(),
		LogLevel:          "INFO",
		APILoggingEnabled: false,
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	b := &StatusBackend{
		rootDataDir: rootDataDir, // FIXME: Make it absolute path
		logger:      cfg.logger,
	}

	// Start RPC server
	// NOTE: After refactoring, all actions below will be called as part of some service
	b.rpcServer = gethrpc.NewServer()
	b.Start()

	// Make sure the data directory exists
	err := os.MkdirAll(b.rootDataDir, 0700)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create data directory")
	}

	// Initialize logging
	logSettings := logutils.LogSettings{
		Enabled: true,
		Level:   cfg.LogLevel,
	}
	err = logutils.OverrideRootLoggerWithConfig(logSettings)
	if err != nil {
		return nil, errors.Wrap(err, "failed to override root logger")
	}

	// Initialize API logging
	if cfg.APILoggingEnabled {
		logRequestsFile := path.Join(cfg.LogsDir, DefaultAPILogFile)
		err = requestlog.ConfigureAndEnableRequestLogging(logRequestsFile)
		if err != nil {
			return nil, errors.Wrap(err, "failed to configure request logging")
		}
	}

	// Initialize metrics
	if cfg.MetricsEnabled { // TODO: Move to a separate service
		// Start metrics server
		b.prometheusMetrics = metrics.NewMetricsServer(cfg.MetricsAddress)
		go b.prometheusMetrics.Listen()
		logutils.ZapLogger().Info("metrics prometheus server started",
			zap.String("address", cfg.MetricsAddress))
	}

	// Create databases
	db, err := multiaccounts.InitializeDB(filepath.Join(b.rootDataDir, "accounts.sql"))
	if err != nil {
		b.logger.Error("failed to initialize accounts db", zap.Error(err))
		return nil, err
	}
	b.rootDB = db.DB()
	b.multiaccountsDB = db

	// Initialize centralized metrics
	// NOTE: We always create it initially. Later we might remove it if account has privacy mode enabled.
	b.centralizedMetrics = centralizedmetrics.NewDefaultMetricService(b.multiaccountsDB.DB(), b.logger)
	err = b.centralizedMetrics.EnsureStarted() // TODO: Also don't even create the service, if it is disabled
	if err != nil {
		return nil, errors.Wrap(err, "failed to initialize metrics")
	}

	// Initialize panic reporting // TODO: Extract as a separate service
	metricsInfo, err := b.CentralizedMetricsInfo()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get metrics info")
	}
	if metricsInfo.Enabled {
		err = sentry.Init(
			sentry.WithDSN(cfg.sentryDSN),
			sentry.WithDefaultContext(),
		)
		if err != nil {
			return nil, errors.Wrap(err, "failed to initialize panic reporting")
		}
	}

	// Create IPFS downloader
	b.ipfs = ipfs.NewDownloader(b.rootDataDir)

	// Create media server
	// TODO: Extract as a separate service
	if cfg.mediaServiceEnabled {
		err = b.startMediaServer(cfg.mediaServerAddress, cfg.mediaServerAdvertizeHost, cfg.mediaServerAdvertizePort)
		if err != nil {
			return nil, errors.Wrap(err, "failed to start media server")
		}
	}

	// Create root service
	b.rootService, err = root.NewService(
		b.rootDataDir,
		b.logger.Named("root"),
		b.multiaccountsDB,
		b.mediaServer)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create root service")
	}

	err = b.rpcServer.RegisterName("root", b.rootService.API())
	if err != nil {
		return nil, errors.Wrap(err, "failed to register root service")
	}

	b.logger.Info("status backend initialized",
		zap.String("version", version.Version()),
		zap.String("commit", version.GitCommit()))

	return b, nil
}

func (b *StatusBackend) Start() {
	//settingsService :=
	//	b.rpcServer.RegisterName("settings", settingsService.API())
}

func (b *StatusBackend) Stop() error {
	var err error

	if b.rpcServer != nil {
		b.rpcServer.Stop()
	}

	if b.ipfs != nil {
		b.ipfs.Stop()
	}

	if b.mediaServer != nil {
		err = b.mediaServer.Stop()
		if err != nil {
			b.logger.Error("failed to stop media server", zap.Error(err))
		}
	}

	if b.centralizedMetrics != nil {
		b.centralizedMetrics.Stop()
	}

	if b.prometheusMetrics != nil {
		err = b.prometheusMetrics.Stop()
		if err != nil {
			b.logger.Error("failed to stop prometheus metrics server", zap.Error(err))
		}
	}

	return nil
}

func (b *StatusBackend) startMediaServer(address, advertizeHost string, advertizePort int) error {
	if b.mediaServer != nil {
		if err := b.mediaServer.Stop(); err != nil {
			return err
		}
	}

	opts := []server.MediaServerOption{
		server.WithMediaServerDisableTLS(false),
		server.WithMediaServerAddress(address),
		server.WithMediaServerAdvertizeAddress(advertizeHost, advertizePort),
	}
	mediaServer, err := server.NewMediaServer(nil, nil, b.multiaccountsDB, nil, opts...)
	if err != nil {
		return err
	}
	mediaServer.SetDataProviders(b.appDB, b.walletDB, b.ipfs)

	b.mediaServer = mediaServer

	if err := b.mediaServer.Start(); err != nil {
		return err
	}

	return nil
}

// TODO: Move to a CentralizedMetrics service
func (b *StatusBackend) CentralizedMetricsInfo() (*centralizedmetrics.MetricsInfo, error) {
	return b.centralizedMetrics.Info()
}

func (b *StatusBackend) RootService() *root.API {
	return b.rootService.API()
}

func (b *StatusBackend) CallInProcessRPC(inputJSON string) string {
	codec := rpc.NewSingleRequestCodec(inputJSON)
	b.rpcServer.ServeCodec(codec.GethCodec(), 0)
	return codec.Output()
}
