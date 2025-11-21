package backend

import (
	"context"
	"database/sql"
	"os"
	"path"
	"path/filepath"

	gethrpc "github.com/ethereum/go-ethereum/rpc"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	accscommon "github.com/status-im/status-go/accounts-management/common"
	"github.com/status-im/status-go/centralizedmetrics"
	"github.com/status-im/status-go/common"
	"github.com/status-im/status-go/internal/metrics"
	"github.com/status-im/status-go/ipfs"
	"github.com/status-im/status-go/logutils"
	"github.com/status-im/status-go/logutils/requestlog"
	"github.com/status-im/status-go/multiaccounts"
	requests2 "github.com/status-im/status-go/pkg/backend/requests"
	"github.com/status-im/status-go/pkg/backend/rpc"
	"github.com/status-im/status-go/pkg/sentry"
	"github.com/status-im/status-go/pkg/version"
	"github.com/status-im/status-go/protocol/requests"
	"github.com/status-im/status-go/services/media"
)

const (
	DefaultAPILogFile = "api.log"
)

type StatusBackendService interface {
	Start() error
	Stop() error
	API() interface{}
	//Metrics() // TODO: Prometheus metrics
}

type StatusBackend struct {
	rootDataDir string
	logger      *zap.Logger

	// Databases
	rootDB          *sql.DB                 // aka multiaccounts db, aka 'accounts.sql'
	multiaccountsDB *multiaccounts.Database // FIXME: Remove this pointer

	// Active Account
	activeAccount *ActiveAccount

	// RPC server
	rpcServer *gethrpc.Server
	services  []StatusBackendService // NOTE: Not sure if we need it. We'll still have to keep pointers to each service.

	// Services
	mediaService *media.Service

	// FIXME: Extract to separate services
	ipfs               *ipfs.Downloader
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
	// TODO: No need to create it before login.
	b.ipfs = ipfs.NewDownloader(b.rootDataDir)

	// Create media service
	if cfg.mediaServiceEnabled {
		err = b.createMediaService(cfg.mediaServerAddress, cfg.mediaServerAdvertizeHost, cfg.mediaServerAdvertizePort)
		if err != nil {
			return nil, errors.Wrap(err, "failed to start media server")
		}
	}

	// Register ourselves as a service
	err = b.rpcServer.RegisterName("backend", b.API())
	if err != nil {
		return nil, errors.Wrap(err, "failed to register root service")
	}

	for _, service := range b.services {
		err = service.Start()
		if err != nil {
			return nil, errors.Wrap(err, "failed to start service")
		}
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

func (b *StatusBackend) Shutdown() error {
	var err error

	if b.rpcServer != nil {
		b.rpcServer.Stop()
	}

	for _, service := range b.services {
		err = service.Stop()
		if err != nil {
			b.logger.Error("failed to stop service", zap.Error(err))
		}
	}
	b.services = nil

	if b.ipfs != nil {
		b.ipfs.Stop()
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

func (b *StatusBackend) API() *API {
	return &API{
		backend: b,
	}
}

func (b *StatusBackend) createMediaService(address, advertizeHost string, advertizePort int) error {
	if b.mediaService != nil {
		// Media service should only be spawned once
		return errors.New("media server is already running")
	}

	opts := []media.Option{
		media.WithLogger(b.logger.Named("media-server")),
		media.WithDisableTLS(false),
		media.WithServerAddress(address),
		media.WithServerAdvertizeAddress(advertizeHost, advertizePort),
	}

	mediaService, err := media.NewService(nil, b.ipfs, b.multiaccountsDB, nil, opts...)
	if err != nil {
		return err
	}
	//mediaService.SetDataProviders(b.ActiveAccount.appDB, b.ActiveAccount.walletDB, b.ipfs)

	// Register service
	err = b.rpcServer.RegisterName("media", mediaService.API())
	if err != nil {
		return errors.Wrap(err, "failed to register media server")
	}

	// Start service
	err = b.mediaService.Start()
	if err != nil {
		return err
	}

	b.services = append(b.services, mediaService)
	b.mediaService = mediaService
	return nil
}

// TODO: Move to a CentralizedMetrics service
func (b *StatusBackend) CentralizedMetricsInfo() (*centralizedmetrics.MetricsInfo, error) {
	return b.centralizedMetrics.Info()
}

func (b *StatusBackend) CallInProcessRPC(inputJSON string) string {
	codec := rpc.NewSingleRequestCodec(inputJSON)
	b.rpcServer.ServeCodec(codec.GethCodec(), 0)
	return codec.Output()
}

func (b *StatusBackend) ListAccounts(ctx context.Context) ([]multiaccounts.Account, error) {
	accs, err := b.multiaccountsDB.GetAccounts()

	if b.mediaService != nil {
		for i, acc := range accs {
			for j, images := range acc.Images {
				url := b.mediaService.MakeAccountImageURL(acc.KeyUID, images.Name, images.Clock)
				accs[i].Images[j].LocalURL = url
			}
		}
	}

	return accs, err
}

func (b *StatusBackend) CreateAccount(ctx context.Context, request *requests2.CreateAccount, keycardData *requests2.KeycardData) (*multiaccounts.Account, error) {

	//creator := NewAccountCreator(b.rootDataDir, b.logger.Named("account-creator"), b.multiaccountsDB, b.mediaService)
	//return creator.StartNodeWithChatKeyOrMnemonic(ctx, request, mnemonic, keycardData, true)

	activeAcc, err := CreateAccount(b.rootDataDir, b.logger, request)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create account")
	}

	// We don't need the account, so we can immediately close it
	err = activeAcc.Close()
	if err != nil {
		b.logger.Error("failed to close account", zap.Error(err))
	}

	// Save account to multiaccounts database
	err = b.multiaccountsDB.SaveAccount(*activeAcc.account)
	if err != nil {
		return nil, errors.Wrap(err, "failed to save account")
	}

	return activeAcc.account, nil
}

func (b *StatusBackend) Login(request *requests.Login) error {
	// Get account from database
	acc, err := b.multiaccountsDB.GetAccount(request.KeyUID)
	if err != nil {
		return errors.Wrap(err, "failed to get account")
	}
	if acc == nil {
		return errors.New("account not found")
	}

	// Set runtime parameters
	if request.RuntimeLogLevel != "" {
		// TODO
	}

	// Open databases
	activeAccount, err := Login(b.rootDataDir, b.logger, acc, request.Password)
	if err != nil {
		return errors.Wrap(err, "failed to login")
	}

	// Run migrations

	// Register and start services, according to persisted settings
	// Root service should only load the settings required to define the list of services to start
	// Other services should load their settings through its persistence interfaces.
	// NOTE: For now we always register and start all services, as we did before.
	//   (messenger and wallet started by client)

	// Send LoggedIn signal to the client
	// TODO: Perhaps this signal makes no sense when Login is a sync call
	err = b.LoggedIn(request.KeyUID, err)
	if err != nil {
		return errors.Wrap(err, "failed to send LoggedIn signal")
	}

	return nil
}
