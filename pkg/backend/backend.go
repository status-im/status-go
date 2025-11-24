package backend

import (
	"context"
	"database/sql"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/ethereum/go-ethereum/event"
	gethrpc "github.com/ethereum/go-ethereum/rpc"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/status-im/status-go/centralizedmetrics"
	"github.com/status-im/status-go/internal/metrics"
	"github.com/status-im/status-go/ipfs"
	"github.com/status-im/status-go/logutils"
	"github.com/status-im/status-go/logutils/requestlog"
	"github.com/status-im/status-go/multiaccounts"
	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/pkg/backend/jsonrpc"
	"github.com/status-im/status-go/pkg/sentry"
	"github.com/status-im/status-go/pkg/version"
	"github.com/status-im/status-go/rpc"
	"github.com/status-im/status-go/services/wallet/community"
	"github.com/status-im/status-go/services/wallet/token"
	"github.com/status-im/status-go/transactions"
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
	rpcClient          *rpc.Client
	ipfs               *ipfs.Downloader
	centralizedMetrics *centralizedmetrics.MetricService
	prometheusMetrics  *metrics.Server
	transactor         *transactions.Transactor
	tokenManager       *token.Manager
	cancelTokenManager context.CancelFunc // TODO: This is a temporary solution. TokenManager should be properly stopped.

	// FIXME: Replace events feed with pubsub - https://github.com/status-im/status-go/issues/6744
	// FIXME: It should not be a part of the backend, but rather a part of the wallet service
	walletFeed event.Feed
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

// FIXME: Also implement Logout. This should keep the main services running, but stop the services started at login.
func (b *StatusBackend) Shutdown() error {
	var err error

	b.rpcServer.Stop()
	b.services.Stop()
	b.services = nil

	b.rpcClient.Stop()
	b.rpcClient = nil

	b.cancelTokenManager()

	b.transactor = nil

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
	codec := jsonrpc.NewSingleRequestCodec(inputJSON)
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
	if b.services.mediaService == nil {
		return
	}

	for k, v := range acc.Images {
		url := b.services.mediaService.MakeAccountImageURL(acc.KeyUID, v.Name, v.Clock)
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

func (b *StatusBackend) createRPCClient() error {
	rpcClient, err := rpc.NewClient(rpc.ClientConfig{
		Networks:          b.activeAccount.nodeConfig.Networks,
		DB:                b.activeAccount.appDB,
		AccountsPublisher: b.services.accountsSrvc.Publisher(),
	})
	if err != nil {
		return err
	}

	b.rpcClient = rpcClient
	return nil
}

func (b *StatusBackend) createTokenManager() error {
	accDB, err := accounts.NewDB(b.activeAccount.appDB)
	if err != nil {
		return err
	}

	b.tokenManager = token.NewTokenManager(
		b.activeAccount.walletDB,
		b.rpcClient,
		community.NewManager(b.activeAccount.appDB, b.services.mediaService, nil),
		b.rpcClient.GetNetworkManager(),
		b.activeAccount.appDB,
		b.services.mediaService,
		&b.walletFeed,
		b.services.accountsSrvc.Publisher(),
		accDB,
		token.NewPersistence(b.activeAccount.walletDB),
	)

	return nil
}

func (b *StatusBackend) tokenManagerAutoRefreshInterval() (time.Duration, time.Duration) {
	const (
		defaultAutoRefreshInterval      = 30 * time.Minute // interval after which we should fetch the token lists from the remote source (or use the default one if remote source is not set)
		defaultAutoRefreshCheckInterval = 3 * time.Minute  // interval after which we should check if we should trigger the auto-refresh
	)

	autoRefreshInterval := defaultAutoRefreshInterval
	autoRefreshCheckInterval := defaultAutoRefreshCheckInterval

	configInterval := b.activeAccount.nodeConfig.WalletConfig.TokensListsAutoRefreshInterval
	configCheckInterval := b.activeAccount.nodeConfig.WalletConfig.TokensListsAutoRefreshCheckInterval

	if configInterval > 0 && configCheckInterval > 0 && configInterval > configCheckInterval {
		autoRefreshInterval = time.Duration(configInterval) * time.Second
		autoRefreshCheckInterval = time.Duration(configCheckInterval) * time.Second
	}

	return autoRefreshInterval, autoRefreshCheckInterval
}

// Register and start services, according to persisted settings
// Root service should only load the settings required to define the list of services to start
// Other services should load their settings through its persistence interfaces.
// NOTE: For now we always register and start all services, as we did before. (messenger and wallet started by client)
func (b *StatusBackend) startServices() error {
	// 0. Start prerequisites that are not yet extracted as services
	err := b.createRPCClient()
	if err != nil {
		return errors.Wrap(err, "failed to create rpc client")
	}

	err = b.createTokenManager()
	if err != nil {
		return errors.Wrap(err, "failed to create token manager")
	}

	b.transactor = transactions.NewTransactor()

	// TODO: 1. Spawn settings service

	// 2. Read settings required to decide which allServices to start
	cfg, err := b.activeAccount.ServicesConfig()
	if err != nil {
		return errors.Wrap(err, "failed to load active account config")
	}

	// 3. Start allServices
	b.services.createRPCStats()
	b.services.createAppgeneralService()
	b.services.createPersonalService()
	b.services.createStatusPublicService()
	b.services.createPendingTrackerService()
	b.services.createEnsService()
	b.services.createCommunityTokensService()
	b.services.createStickersService()
	b.services.createUpdatesService()
	b.services.createAccountsService()

	if cfg.browserEnabled {
		b.services.createBrowsersService()
	}

	if cfg.permissionsServiceEnabled {
		b.services.createPermissionsService()
	}

	if cfg.connectorEnabled {
		b.services.createConnectorService()
	}

	b.services.createGifService(b.activeAccount.accountsDB)
	b.services.createChatService(b.activeAccount.accountsDB)
	b.services.createEthService()

	// Wallet Service is used by wakuExtSrvc/wakuV2ExtSrvc
	// Keep this initialization before the other two
	if cfg.walletEnabled {
		b.services.createWalletService()
	}

	// CollectiblesManager needs the WakuExt service to get metadata for
	// Community collectibles.
	// Messenger needs the CollectiblesManager to get the list of collectibles owned
	// by a certain account and check community entry permissions.
	// We handle circular dependency between the two by delaying initialization of the CommunityCollectibleInfoProvider
	// in the CollectiblesManager.
	if cfg.wakuV2ExtEnabled {
		err := b.services.createWakuExtService()
		if err != nil {
			return errors.Wrap(err, "failed to initialize waku v2 extension service")
		}
	}

	b.services.createLocalNotificationsService()
	b.services.createNewsFeedService()
	b.services.createSharedUrlsService()

	// Register all created services
	err = b.services.Register()
	if err != nil {
		return errors.Wrap(err, "failed to register services")
	}

	// TEMP: Start non-services
	b.rpcClient.Start()

	ctx, cancel := context.WithCancel(context.Background())
	interval, checkInterval := b.tokenManagerAutoRefreshInterval()
	b.tokenManager.Start(ctx, interval, checkInterval)
	b.cancelTokenManager = cancel

	// Start all created services
	err = b.services.Start()
	if err != nil {
		return errors.Wrap(err, "failed to start services")
	}

	return nil
}
