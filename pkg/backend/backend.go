package backend

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path"
	"path/filepath"

	gethrpc "github.com/ethereum/go-ethereum/rpc"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/status-im/status-go/centralizedmetrics"
	"github.com/status-im/status-go/common"
	"github.com/status-im/status-go/internal/metrics"
	"github.com/status-im/status-go/ipfs"
	"github.com/status-im/status-go/logutils"
	"github.com/status-im/status-go/logutils/requestlog"
	"github.com/status-im/status-go/multiaccounts"
	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/multiaccounts/settings"
	"github.com/status-im/status-go/pkg/backend/rpc"
	"github.com/status-im/status-go/pkg/sentry"
	"github.com/status-im/status-go/pkg/version"
	"github.com/status-im/status-go/services/media"
	"github.com/status-im/status-go/services/rpcstats"
)

const (
	DefaultAPILogFile = "api.log"
)

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
	services  *services

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
	b.services = newServices(b)
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
		err = b.services.createMedia(cfg.mediaServerAddress, cfg.mediaServerAdvertizeHost, cfg.mediaServerAdvertizePort)
		if err != nil {
			return nil, errors.Wrap(err, "failed to start media server")
		}
	}

	// Register ourselves as a service
	err = b.rpcServer.RegisterName("backend", b.API())
	if err != nil {
		return nil, errors.Wrap(err, "failed to register root service")
	}

	err = b.services.Start()
	if err != nil {
		return nil, errors.Wrap(err, "failed to start services")
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

	b.rpcServer.Stop()
	b.services.Stop()
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

	for i, acc := range accs {
		b.setAccountsImageURLs(&acc)
		accs[i] = acc
	}

	return accs, err
}

// setAccountsImageURLs sets acc.Images using b.mediaService.
// TODO: This should be done using a multiaccounts service
func (b *StatusBackend) setAccountsImageURLs(acc *multiaccounts.Account) {
	if b.mediaService == nil {
		return
	}

	for k, v := range acc.Images {
		url := b.mediaService.MakeAccountImageURL(acc.KeyUID, v.Name, v.Clock)
		acc.Images[k].LocalURL = url
	}
}

type ServicesConfig struct {
	browserEnabled            bool
	permissionsServiceEnabled bool
	connectorEnabled          bool
	walletEnabled             bool
	wakuV2ExtEnabled          bool
}

// Register and start services, according to persisted settings
// Root service should only load the settings required to define the list of services to start
// Other services should load their settings through its persistence interfaces.
// NOTE: For now we always register and start all services, as we did before. (messenger and wallet started by client)
func (b *StatusBackend) startServices() error {
	// 1. Spawn settings service

	// 2. Read settings required to decide which allServices to start
	cfg, err := b.activeAccount.ServicesConfig()
	if err != nil {
		return errors.Wrap(err, "failed to load active account config")
	}

	accDB := b.activeAccount.accountsDB

	// 3. Start allServices
	b.services.createRPCStats()
	b.services.appgeneralService()
	b.services.personalService()
	b.services.statusPublicService()
	b.services.pendingTrackerService(&b.walletFeed)
	b.services.ensService(b.timeSourceNow())
	b.services.CommunityTokensService()
	b.services.stickersService(b.activeAccount.accountsDB)
	b.services.updatesService()
	b.services.accountsService(b.activeAccount.accountsDB, mediaServer)

	if cfg.browserEnabled {
		b.services.createBrowsersService()
	}

	if cfg.permissionsServiceEnabled {
		b.services.permissionsService()
	}

	if cfg.connectorEnabled {
		b.services.connectorService()
	}

	b.services.gifService(b.activeAccount.accountsDB)
	b.services.ChatService(b.activeAccount.accountsDB)
	b.services.ethService()

	// Wallet Service is used by wakuExtSrvc/wakuV2ExtSrvc
	// Keep this initialization before the other two
	if cfg.walletEnabled {
		b.services.createWallet()
	}

	// CollectiblesManager needs the WakuExt service to get metadata for
	// Community collectibles.
	// Messenger needs the CollectiblesManager to get the list of collectibles owned
	// by a certain account and check community entry permissions.
	// We handle circular dependency between the two by delaying initialization of the CommunityCollectibleInfoProvider
	// in the CollectiblesManager.
	if cfg.wakuV2ExtEnabled {
		b.services.wakuV2ExtService()
	}

	b.services.localNotificationsService()
	b.services.NewsFeedService()
	b.services.sharedUrlsService()

	// FIXME: refactor waku ext services to MessengerService.
	//   There should be no custom InitProtocol functions, services should be set up in messenger.NewService().
	//   And messenger should not be started from client, but just as a regular service.
	initProtocol()

	// Register ourselves as a service
	err = b.services.Register()
	if err != nil {
		return errors.Wrap(err, "failed to register services")
	}

	err = b.services.Start()
	if err != nil {
		return errors.Wrap(err, "failed to start services")
	}

	return nil
}

func appendIf(services []common.StatusService, service common.StatusService, condition bool) []common.StatusService {
	if !condition {
		return services
	}
	return append(services, service)
}
