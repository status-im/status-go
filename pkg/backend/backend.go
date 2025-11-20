package backend

import (
	"context"
	"crypto/ecdsa"
	"database/sql"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/afex/hystrix-go/hystrix"
	"github.com/pkg/errors"

	"github.com/imdario/mergo"

	"github.com/ethereum/go-ethereum/common/hexutil"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	signercore "github.com/ethereum/go-ethereum/signer/core/apitypes"

	accsmanagement "github.com/status-im/status-go/accounts-management"
	accscommon "github.com/status-im/status-go/accounts-management/common"
	"github.com/status-im/status-go/accounts-management/generator"
	accsmanagementtypes "github.com/status-im/status-go/accounts-management/types"
	"github.com/status-im/status-go/api"
	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/centralizedmetrics"
	centralizedmetricscommon "github.com/status-im/status-go/centralizedmetrics/common"
	gocommon "github.com/status-im/status-go/common"
	"github.com/status-im/status-go/common/dbsetup"
	"github.com/status-im/status-go/connection"
	"github.com/status-im/status-go/crypto"
	"github.com/status-im/status-go/crypto/types"
	"github.com/status-im/status-go/images"
	"github.com/status-im/status-go/internal/metrics"
	"github.com/status-im/status-go/ipfs"
	"github.com/status-im/status-go/logutils"
	"github.com/status-im/status-go/logutils/requestlog"
	"github.com/status-im/status-go/multiaccounts"
	"github.com/status-im/status-go/multiaccounts/accounts"
	multiacccommon "github.com/status-im/status-go/multiaccounts/common"
	"github.com/status-im/status-go/multiaccounts/settings"
	"github.com/status-im/status-go/node/adapters"
	"github.com/status-im/status-go/nodecfg"
	"github.com/status-im/status-go/params"
	requests2 "github.com/status-im/status-go/pkg/backend/requests"
	"github.com/status-im/status-go/pkg/sentry"
	"github.com/status-im/status-go/pkg/version"
	"github.com/status-im/status-go/protocol"
	"github.com/status-im/status-go/protocol/communities"
	identityutils "github.com/status-im/status-go/protocol/identity"
	"github.com/status-im/status-go/protocol/identity/colorhash"
	"github.com/status-im/status-go/protocol/requests"
	"github.com/status-im/status-go/rpc"
	"github.com/status-im/status-go/server"
	"github.com/status-im/status-go/server/pairing/statecontrol"
	"github.com/status-im/status-go/services/ens"
	"github.com/status-im/status-go/services/ext"
	"github.com/status-im/status-go/services/personal"
	"github.com/status-im/status-go/services/typeddata"
	"github.com/status-im/status-go/services/wallet"
	"github.com/status-im/status-go/services/wallet/wallettypes"
	"github.com/status-im/status-go/signal"
	"github.com/status-im/status-go/sqlite"
	"github.com/status-im/status-go/transactions"
	"github.com/status-im/status-go/walletdatabase"
)

var (
	// ErrDBNotAvailable is returned if a method is called before the DB is available for usage
	ErrDBNotAvailable = errors.New("DB is unavailable")
)

const (
	DefaultAPILogFile = "api.log"

	walletAccountDefaultName                  = "Account 1"
	DefaultKeycardPairingDataFileRelativePath = "/keycard/pairings.json"
)

type LoginParams struct {
	ChatAddress  types.Address          `json:"chatAddress"`
	Password     string                 `json:"password"`
	MultiAccount *multiaccounts.Account `json:"multiAccount"`
}

type activeAccount struct {
	// Databases
	account  *multiaccounts.Account
	appDB    *sql.DB
	walletDB *sql.DB

	// Accounts Manager
	accountsManager *accsmanagement.AccountsManager
}

// StatusBackend is the root object that encapsulates all Status Backend functionality.
//   - Opens root database (in given directory)
//   - Creates an RPC server
//   - Registers its own API in RPC server, which is used for CreateAccount, Login, etc.
//   - On Login, it should start all required services, according to activeAccount settings.
//     This is why CreateAccount/Login/etc are not extracted as a separate service.
type StatusBackend struct {
	mu sync.Mutex

	// rootDataDir
	rootDataDir string
	logger      *zap.Logger

	// RPC server
	//rpcServer *gethrpc.Server
	services *Services

	// TODO:
	rootDB          *sql.DB                 // aka multiaccounts db, aka 'accounts.sql'
	multiaccountsDB *multiaccounts.Database // FIXME: Remove this pointer

	// Active (logged in) account
	activeAccount *activeAccount
	//account       *multiaccounts.Account
	//appDB         *sql.DB
	//walletDB      *sql.DB

	// FIXME: Extract to separate services
	ipfs               *ipfs.Downloader
	mediaServer        *server.MediaServer
	centralizedMetrics *centralizedmetrics.MetricService
	prometheusMetrics  *metrics.Server

	// FIXME: Old implementation below
	//config      *params.NodeConfig
	signer communities.MessageSigner
	//accountsManager          *accsmanagement.AccountsManager
	transactor               *transactions.Transactor
	connectionState          connection.State
	appState                 api.AppState
	LocalPairingStateManager *statecontrol.ProcessStateManager
	//sentryDSN                string
	//preLoginLogConfig *logutils.PreLoginLogConfig
}

// NewStatusBackend create a new StatusBackend instance
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

	if gocommon.IsMobilePlatform() {
		debug.SetMemoryLimit(1024 * 1024 * 150) // 150MB
	}

	err := b.initialize(&cfg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to initialize backend")
	}

	b.logger.Info("status backend initialized",
		zap.String("version", version.Version()),
		zap.String("commit", version.GitCommit()))

	return b, nil
}

func (b *StatusBackend) initialize(cfg *config) error {
	// Start RPC server
	// NOTE: After refactoring, all actions below will be called as part of some service
	b.services = NewServices(b, b.logger.Named("services"))
	//b.rpcServer = gethrpc.NewServer()
	b.Start()

	// Make sure the data directory exists
	err := os.MkdirAll(b.rootDataDir, 0700)
	if err != nil {
		return errors.Wrap(err, "failed to create data directory")
	}

	// Initialize logging
	logSettings := logutils.LogSettings{
		Enabled: true,
		Level:   cfg.LogLevel,
	}
	err = logutils.OverrideRootLoggerWithConfig(logSettings)
	if err != nil {
		return errors.Wrap(err, "failed to override root logger")
	}

	// Initialize API logging
	if cfg.APILoggingEnabled {
		logRequestsFile := path.Join(cfg.LogsDir, DefaultAPILogFile)
		err = requestlog.ConfigureAndEnableRequestLogging(logRequestsFile)
		if err != nil {
			return errors.Wrap(err, "failed to configure request logging")
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
		return err
	}
	b.rootDB = db.DB()
	b.multiaccountsDB = db

	// Initialize centralized metrics
	// NOTE: We always create it initially. Later we might remove it if account has privacy mode enabled.
	b.centralizedMetrics = centralizedmetrics.NewDefaultMetricService(b.multiaccountsDB.DB(), b.logger)
	err = b.centralizedMetrics.EnsureStarted() // TODO: Also don't even create the service, if it is disabled
	if err != nil {
		return errors.Wrap(err, "failed to initialize metrics")
	}

	// Initialize panic reporting // TODO: Extract as a separate service
	metricsInfo, err := b.CentralizedMetricsInfo()
	if err != nil {
		return errors.Wrap(err, "failed to get metrics info")
	}
	if metricsInfo.Enabled {
		err = sentry.Init(
			sentry.WithDSN(cfg.sentryDSN),
			sentry.WithDefaultContext(),
		)
		if err != nil {
			return errors.Wrap(err, "failed to initialize panic reporting")
		}
	}

	// Create IPFS downloader
	// TODO: No need to create it before login.
	b.ipfs = ipfs.NewDownloader(b.rootDataDir)

	// Create media server
	// TODO: Extract as a separate service
	if cfg.mediaServiceEnabled {
		err = b.startMediaServer(cfg.mediaServerAddress, cfg.mediaServerAdvertizeHost, cfg.mediaServerAdvertizePort)
		if err != nil {
			return errors.Wrap(err, "failed to start media server")
		}
	}

	// Register ourselves as a service
	err = b.services.RegisterName("backend", &API{backend: b})
	if err != nil {
		return errors.Wrap(err, "failed to register root service")
	}

	// NOTE: Old GetStatusBackend initialize code:
	//accountsManager, err := accsmanagement.NewAccountsManager(b.logger)
	//if err != nil {
	//	b.logger.Error("failed to create new *AccountsManager instance", zap.Error(err))
	//	return
	//}
	//
	//transactor := transactions.NewTransactor()
	//personalService := personal.NewServices()
	//services := node.NewServices(transactor, accountsManager, b.logger)
	//
	//b.services = services
	//b.accountsManager = accountsManager
	//b.transactor = transactor
	//b.signer = personalService
	//b.services.SetMultiaccountsDB(b.multiaccountsDB)
	//b.LocalPairingStateManager = new(statecontrol.ProcessStateManager)
	//b.LocalPairingStateManager.SetPairing(false)
	//b.LocalPairingStateManager.SetMessageSyncEnabled(false)

	return nil
}

func (b *StatusBackend) Start() {
	//settingsService :=
	//	b.rpcServer.RegisterName("settings", settingsService.API())
}

func (b *StatusBackend) Stop() error {
	var err error

	if b.services != nil {
		err = b.services.Stop()
		if err != nil {
			b.logger.Error("failed to stop services", zap.Error(err))
		}
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

func (b *StatusBackend) ListAccounts(ctx context.Context) ([]multiaccounts.Account, error) {
	accs, err := b.multiaccountsDB.GetAccounts()

	if b.mediaServer != nil {
		for i, acc := range accs {
			for j, img := range acc.Images {
				url := b.mediaServer.MakeAccountImageURL(acc.KeyUID, img.Name, img.Clock)
				accs[i].Images[j].LocalURL = url
			}
		}
	}

	return accs, err
}

//func (b StatusBackend) CreateKeycardAccount(ctx context.Context, keycardData *requests2.KeycardData) (*multiaccounts.Account, error) {
//	accountInfo := b.deriveFromKeycard(keycardData)
//	return b.StartNodeWithChatKeyOrMnemonic()
//}

// Services returns reference to node manager
func (b *StatusBackend) StatusNode() *Services {
	return b.services
}

// Transactor returns reference to a status transactor
func (b *StatusBackend) Transactor() *transactions.Transactor {
	return b.transactor
}

func (b *StatusBackend) MessageSigner() communities.MessageSigner {
	return b.signer
}

// IsNodeRunning confirm that node is running
func (b *StatusBackend) IsNodeRunning() bool {
	return b.services.IsRunning()
}

// StartNode start Status node, fails if node is already started
func (b *StatusBackend) StartNode(config *params.NodeConfig) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.startNode(config); err != nil {
		signal.SendNodeCrashed(err)
		return err
	}

	// Set initial connection state
	b.services.ConnectionChanged(b.connectionState)

	return nil
}

//func (b *StatusBackend) UpdateRootDataDir(datadir string) {
//	b.mu.Lock()
//	defer b.mu.Unlock()
//	b.rootDataDir = datadir
//	b.accountsManager.SetRootDataDir(datadir)
//}

func (b *StatusBackend) GetMultiaccountDB() *multiaccounts.Database {
	return b.multiaccountsDB
}

func (b *StatusBackend) OpenAccounts(thirdpartyServicesEnabled bool) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.multiaccountsDB != nil {
		return nil
	}
	db, err := multiaccounts.InitializeDB(filepath.Join(b.rootDataDir, "accounts.sql"))
	if err != nil {
		b.logger.Error("failed to initialize accounts db", zap.Error(err))
		return err
	}
	b.multiaccountsDB = db

	if thirdpartyServicesEnabled {
		b.centralizedMetrics = centralizedmetrics.NewDefaultMetricService(b.multiaccountsDB.DB(), b.logger)
		err = b.centralizedMetrics.EnsureStarted()
		if err != nil {
			return err
		}
	}

	// Probably we should iron out a bit better how to create/dispose of the status-service
	b.services.SetMultiaccountsDB(db)

	return nil
}

// TODO: Move to a CentralizedMetrics service
func (b *StatusBackend) CentralizedMetricsInfo() (*centralizedmetrics.MetricsInfo, error) {
	if b.centralizedMetrics == nil {
		return nil, errors.New("centralized metrics not initialized")
	}

	return b.centralizedMetrics.Info()
}

func (b *StatusBackend) ToggleCentralizedMetrics(isEnabled bool) error {
	if b.centralizedMetrics == nil {
		return errors.New("centralized metrics nil")
	}

	return b.centralizedMetrics.ToggleEnabled(isEnabled)
}

func (b *StatusBackend) AddCentralizedMetric(metric centralizedmetricscommon.Metric) error {
	if b.centralizedMetrics == nil {
		return errors.New("centralized metrics nil")
	}
	return b.centralizedMetrics.AddMetric(metric)

}

func (b *StatusBackend) GetAccounts() ([]multiaccounts.Account, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.multiaccountsDB == nil {
		return nil, errors.New("accounts db wasn't initialized")
	}
	return b.multiaccountsDB.GetAccounts()
}

func (b *StatusBackend) AcceptTerms() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.multiaccountsDB == nil {
		return errors.New("accounts db wasn't initialized")
	}

	accounts, err := b.multiaccountsDB.GetAccounts()
	if err != nil {
		return err
	}
	if len(accounts) == 0 {
		return errors.New("accounts is empty")
	}

	return b.multiaccountsDB.UpdateHasAcceptedTerms(accounts[0].KeyUID, true)
}

func (b *StatusBackend) StartPrometheusMetricsServer(address string) error {
	if b.prometheusMetrics != nil {
		return nil
	}
	b.prometheusMetrics = metrics.NewMetricsServer(address)
	go b.prometheusMetrics.Listen()
	return nil
}

func (b *StatusBackend) getAccountByKeyUID(keyUID string) (*multiaccounts.Account, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.multiaccountsDB == nil {
		return nil, errors.New("accounts db wasn't initialized")
	}
	as, err := b.multiaccountsDB.GetAccounts()
	if err != nil {
		return nil, err
	}
	for _, acc := range as {
		if acc.KeyUID == keyUID {
			for k, v := range acc.Images {
				acc.Images[k].LocalURL = b.mediaServer.MakeAccountImageURL(acc.KeyUID, v.Name, v.Clock)
			}
			return &acc, nil
		}
	}
	return nil, fmt.Errorf("account with keyUID %s not found", gocommon.TruncateWithDot(keyUID))
}

func (b *StatusBackend) SaveAccount(account multiaccounts.Account) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.multiaccountsDB == nil {
		return errors.New("accounts db wasn't initialized")
	}
	return b.multiaccountsDB.SaveAccount(account)
}

func (b *StatusBackend) runDBFileMigrations(account multiaccounts.Account, password string) (string, error) {
	// Migrate file path to fix issue https://github.com/status-im/status-go/issues/2027
	unsupportedPath := filepath.Join(b.rootDataDir, fmt.Sprintf("app-%x.sql", account.KeyUID))
	v3Path := filepath.Join(b.rootDataDir, fmt.Sprintf("%s.db", account.KeyUID))
	v4Path, err := b.getAppDBPath(account.KeyUID)
	if err != nil {
		return "", err
	}

	_, err = os.Stat(unsupportedPath)
	if err == nil {
		err := os.Rename(unsupportedPath, v3Path)
		if err != nil {
			return "", err
		}

		// rename journals as well, but ignore errors
		_ = os.Rename(unsupportedPath+"-shm", v3Path+"-shm")
		_ = os.Rename(unsupportedPath+"-wal", v3Path+"-wal")
	}

	if _, err = os.Stat(v3Path); err == nil {
		if err := appdatabase.MigrateV3ToV4(v3Path, v4Path, password, account.KDFIterations, signal.SendReEncryptionStarted, signal.SendReEncryptionFinished); err != nil {
			_ = os.Remove(v4Path)
			_ = os.Remove(v4Path + "-shm")
			_ = os.Remove(v4Path + "-wal")
			return "", errors.New("Failed to migrate v3 db to v4: " + err.Error())
		}
		_ = os.Remove(v3Path)
		_ = os.Remove(v3Path + "-shm")
		_ = os.Remove(v3Path + "-wal")
	}

	return v4Path, nil
}

func (b *StatusBackend) ensureDBsOpened(account multiaccounts.Account, password string) (err error) {
	// After wallet DB initial migration, the tables moved to wallet DB are removed from appDB
	// so better migrate wallet DB first to avoid removal if wallet DB migration fails
	if err = b.ensureWalletDBOpened(account, password); err != nil {
		return err
	}

	if err = b.ensureAppDBOpened(account, password); err != nil {
		return err
	}

	return nil
}

func (b *activeAccount) ensureAppDBOpened(rootDataDir string, password string) (err error) {
	if b.appDB != nil {
		return nil
	}

	if len(rootDataDir) == 0 {
		return errors.New("root datadir wasn't provided")
	}

	// WARNING: Do we still want this migration?
	//dbFilePath, err := b.runDBFileMigrations(account, password)
	//if err != nil {
	//	return errors.New("Failed to migrate db file: " + err.Error())
	//}

	dbFilePath, err := getAppDBPath(rootDataDir, b.account.KeyUID)
	if err != nil {
		return errors.Wrap(err, "failed to get database file path")
	}

	b.appDB, err = appdatabase.InitializeDB(dbFilePath, password, b.account.KDFIterations)
	if err != nil {
		return errors.Wrap(err, "failed to initialize db")
	}

	//b.services.SetAppDB(b.appDB)

	accountsDB, err := accounts.NewDB(b.appDB)
	if err != nil {
		return errors.Wrap(err, "failed to create new *Database instance")
	}

	b.accountsManager.SetPersistence(accountsDB)
	return nil
}

func fileExists(path string) bool {
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return false
	}

	return true
}

func (b *StatusBackend) walletDBExists(keyUID string) bool {
	path, err := b.getWalletDBPath(keyUID)
	if err != nil {
		return false
	}

	return fileExists(path)
}

func (b *StatusBackend) appDBExists(keyUID string) bool {
	path, err := b.getAppDBPath(keyUID)
	if err != nil {
		return false
	}

	return fileExists(path)
}

func (b *activeAccount) ensureWalletDBOpened(rootDataDir string, password string) (err error) {
	if b.walletDB != nil {
		return nil
	}

	dbWalletPath, err := getWalletDBPath(rootDataDir, b.account.KeyUID)
	if err != nil {
		return err
	}

	b.walletDB, err = walletdatabase.InitializeDB(dbWalletPath, password, b.account.KDFIterations)
	if err != nil {
		return errors.Wrap(err, "failed to initialize wallet db")
	}

	//b.services.SetWalletDB(b.walletDB)
	return nil
}

func (b activeAccount) getSettings() (*settings.Settings, error) {
	accountsDB, err := accounts.NewDB(b.appDB)
	if err != nil {
		return nil, errors.Wrap(err, "failed to wrap accounts db")
	}

	s, err := accountsDB.GetSettings()
	if err != nil {
		return nil, err
	}

	return &s, nil
}

// getNodeConfig combines default configuration with settings in the account database.
// This is a temporary solution and NodeConfig should be dropped in the future.
// Though a similar approach should be implemented:
// - get defaults
// - override wtih settings from DB
// - start corresponding services.
// Important difference: we don't load the settings of the services themselves, only enabled/disabled.
// Services should load their settings/persistence with through interfaces.
// TEMP: rootDataDir
func (b *activeAccount) getNodeConfig(request requests2.Login, rootDataDir string) (*params.NodeConfig, error) {

	accountSettings, err := b.getSettings()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get account settings")
	}

	nodeConfig := &params.NodeConfig{
		// why we need this? relate PR: https://github.com/status-im/status-go/pull/4014
		KeycardPairingDataFile: filepath.Join(rootDataDir, DefaultKeycardPairingDataFileRelativePath),
	}

	nodeConfig.WalletConfig = buildWalletConfig(&request.WalletConfig, &request.WalletSecretsConfig)

	// UpdateNodeConfigFleet
	fleet := accountSettings.GetFleet()
	if !params.IsFleetSupported(fleet) {
		fleet = DefaultFleet
	}
	err = SetFleet(fleet, nodeConfig)
	if err != nil {
		return nil, errors.Wrap(err, "failed to set fleet")
	}

	nodeConfig, err = b.loadNodeConfig(nodeConfig)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load node config")
	}

	if request.RuntimeLogLevel != "" {
		nodeConfig.LogLevel = request.RuntimeLogLevel
	}

	//if nodeConfig.WakuV2Config.Enabled && request.WakuV2Nameserver != "" {
	//	nodeConfig.WakuV2Config.Nameserver = request.WakuV2Nameserver
	//}
	//
	//nodeConfig.ShhextConfig.BandwidthStatsEnabled = request.BandwidthStatsEnabled

	// Override networks
	nodeConfig.Networks = api.BuildDefaultNetworks(&request.WalletSecretsConfig, accountSettings.ThirdpartyServicesEnabled)

	if request.APIConfig != nil {
		overrideApiConfig(nodeConfig, request.APIConfig)
	}

	return nodeConfig, nil
}

func (b *activeAccount) GetChatAddress() (types.Address, error) {
	accountsDB, err := accounts.NewDB(b.appDB)
	if err != nil {
		return types.Address{}, errors.Wrap(err, "failed to wrap accounts db")
	}

	return accountsDB.GetChatAddress()
}

func (b *activeAccount) GetWalletAddress() (types.Address, error) {
	accountsDB, err := accounts.NewDB(b.appDB)
	if err != nil {
		return types.Address{}, errors.Wrap(err, "failed to wrap accounts db")
	}
	return accountsDB.GetWalletAddress()
}

func (b *StatusBackend) SetupLogSettings() error {
	_ = logutils.ZapLogger().Sync()
	return logutils.OverrideRootLoggerWithConfig(b.config.ProfileLogSettings())
}

func (b *activeAccount) OverwriteNodeConfigValues(conf *params.NodeConfig, n *params.NodeConfig) (*params.NodeConfig, error) {
	if err := mergo.Merge(conf, n, mergo.WithOverride); err != nil {
		return nil, err
	}

	conf.Networks = n.Networks

	if err := nodecfg.SaveNodeConfig(b.appDB, conf); err != nil {
		return nil, err
	}

	return conf, nil
}

func (b *StatusBackend) updateAccountColorHashAndColorID(keyUID string, accountsDB *accounts.Database) (*multiaccounts.Account, error) {
	multiAccount, err := b.getAccountByKeyUID(keyUID)
	if err != nil {
		return nil, err
	}
	if multiAccount.ColorHash == nil {
		keypair, err := accountsDB.GetKeypairByKeyUID(keyUID)
		if err != nil {
			return nil, err
		}
		chatAcc := keypair.GetChatAccount()
		if chatAcc == nil {
			return nil, errors.New("chat account not found")
		}
		if err = EnrichMultiAccountByPublicKey(multiAccount, chatAcc.PublicKey); err != nil {
			return nil, err
		}
		if err = b.multiaccountsDB.UpdateAccount(*multiAccount); err != nil {
			return nil, err
		}
	}
	return multiAccount, nil
}

func (b *StatusBackend) overrideNetworks(conf *params.NodeConfig, request *requests2.Login, thirdpartyServicesEnabled bool) {
	conf.Networks = api.BuildDefaultNetworks(&request.WalletSecretsConfig, thirdpartyServicesEnabled)
}

func (b *StatusBackend) LoginAccount(request *requests2.Login) error {
	err := b.loginAccount(request)
	if err != nil {
		// Stop node for clean up
		_ = b.StopNode()
	}
	if b.LocalPairingStateManager.IsPairing() {
		return nil
	}
	err = b.LoggedIn(request.KeyUID, err)
	if err != nil {
		return errors.Wrap(err, "failed to send LoggedIn signal")
	}

	return nil
}

func (b *StatusBackend) loginAccount(request *requests2.Login) error {
	if err := request.Validate(); err != nil {
		return err
	}

	if request.Mnemonic != "" {
		generatedAccount, generatedAccountInfo, err := b.generateAccount(request.Mnemonic)
		if err != nil {
			return errors.Wrap(err, "failed to generate account info")
		}

		if generatedAccountInfo.KeyUID != request.KeyUID {
			return errors.New("mnemonic does not match this account")
		}

		_, generatedDerivedAccountsInfo, err := generateDerivedAddresses(generatedAccount, api.paths)
		if err != nil {
			return errors.Wrap(err, "failed to derive children accounts")
		}

		request.Password = generatedDerivedAccountsInfo[accscommon.PathEIP1581Encryption].PublicKey
		request.KeycardWhisperPrivateKey = generatedDerivedAccountsInfo[accscommon.PathEIP1581Chat].PrivateKey
	}

	acc := multiaccounts.Account{
		KeyUID:        request.KeyUID,
		KDFIterations: request.KdfIterations,
	}

	if acc.KDFIterations == 0 {
		var err error
		acc.KDFIterations, err = b.multiaccountsDB.GetAccountKDFIterationsNumber(acc.KeyUID)
		if err != nil {
			return errors.Wrap(err, "failed to get account kdf iterations number")
		}
	}

	b.UpdateRootDataDir(b.rootDataDir)

	err := b.ensureDBsOpened(acc, request.Password)
	if err != nil {
		return errors.Wrap(err, "failed to open database")
	}

	//defaultCfg := &params.NodeConfig{
	//	// why we need this? relate PR: https://github.com/status-im/status-go/pull/4014
	//	KeycardPairingDataFile: filepath.Join(b.rootDataDir, api.DefaultKeycardPairingDataFileRelativePath),
	//}
	//
	//defaultCfg.WalletConfig = api.buildWalletConfig(&request.WalletConfig, &request.WalletSecretsConfig)
	//
	//err = b.UpdateNodeConfigFleet(acc, request.Password, defaultCfg)
	//if err != nil {
	//	return errors.Wrap(err, "failed to update node config fleet")
	//}
	//
	//err = b.loadNodeConfig(defaultCfg)
	//if err != nil {
	//	return errors.Wrap(err, "failed to load node config")
	//}
	//
	//if request.RuntimeLogLevel != "" {
	//	b.config.LogLevel = request.RuntimeLogLevel
	//}
	//
	//if b.config.WakuV2Config.Enabled && request.WakuV2Nameserver != "" {
	//	b.config.WakuV2Config.Nameserver = request.WakuV2Nameserver
	//}
	//
	//b.config.ShhextConfig.BandwidthStatsEnabled = request.BandwidthStatsEnabled
	//
	//accountSettings, err := b.GetSettings()
	//if err != nil {
	//	return errors.Wrap(err, "failed to load accountSettings")
	//}
	//
	//b.overrideNetworks(b.config, request, accountSettings.ThirdpartyServicesEnabled)
	//
	//if request.APIConfig != nil {
	//	api.overrideApiConfig(b.config, request.APIConfig)
	//}

	accountsDB, err := accounts.NewDB(b.appDB)
	if err != nil {
		return errors.Wrap(err, "failed to create accounts db")
	}

	multiAccount, err := b.updateAccountColorHashAndColorID(acc.KeyUID, accountsDB)
	if err != nil {
		return errors.Wrap(err, "failed to update account color hash and color id")
	}
	b.account = multiAccount

	err = b.StartNode(b.config)
	if err != nil {
		b.logger.Info("failed to start node")
		return errors.Wrap(err, "failed to start node")
	}

	chatAddr, err := accountsDB.GetChatAddress()
	if err != nil {
		return errors.Wrap(err, "failed to get chat address")
	}
	walletAddr, err := accountsDB.GetWalletAddress()
	if err != nil {
		return errors.Wrap(err, "failed to get wallet address")
	}
	login := LoginParams{
		Password:    request.Password,
		ChatAddress: chatAddr,
		MainAccount: walletAddr,
	}

	err = b.SelectAccount(login, request.ChatPrivateKey())
	if err != nil {
		return errors.Wrap(err, "failed to select account")
	}

	err = b.multiaccountsDB.UpdateAccountTimestamp(acc.KeyUID, time.Now().Unix())
	if err != nil {
		b.logger.Error("failed to update account")
		return errors.Wrap(err, "failed to update account")
	}

	return nil
}

// UpdateNodeConfigFleet loads the fleet from the settings and updates the node configuration
// If the fleet in settings is empty, or not supported anymore, it will be overridden with the default fleet.
// In that case settings fleet value remain the same, only runtime node configuration is updated.
func (b *StatusBackend) UpdateNodeConfigFleet(acc multiaccounts.Account, password string, config *params.NodeConfig) error {
	if config == nil {
		return nil
	}

	err := b.ensureDBsOpened(acc, password)
	if err != nil {
		return err
	}

	accountSettings, err := b.GetSettings()
	if err != nil {
		return err
	}

	fleet := accountSettings.GetFleet()

	if !params.IsFleetSupported(fleet) {
		b.logger.Warn("fleet is not supported, overriding with default value",
			zap.String("fleet", fleet),
			zap.String("defaultFleet", DefaultFleet))
		fleet = DefaultFleet
	}

	err = SetFleet(fleet, config)
	if err != nil {
		return err
	}

	return nil
}

// Deprecated: Use loginAccount instead
func (b *StatusBackend) startNodeWithAccount(acc multiaccounts.Account, password string, inputNodeCfg *params.NodeConfig, chatKey *ecdsa.PrivateKey) error {
	b.UpdateRootDataDir(b.rootDataDir)

	err := b.ensureDBsOpened(acc, password)
	if err != nil {
		return err
	}

	err = b.loadNodeConfig(inputNodeCfg)
	if err != nil {
		return err
	}

	accountsDB, err := accounts.NewDB(b.appDB)
	if err != nil {
		return err
	}

	if acc.ColorHash == nil {
		multiAccount, err := b.updateAccountColorHashAndColorID(acc.KeyUID, accountsDB)
		if err != nil {
			return err
		}
		acc = *multiAccount
	}

	b.account = &acc

	err = b.StartNode(b.config)
	if err != nil {
		b.logger.Info("failed to start node", zap.Error(err))
		return err
	}

	chatAddr, err := accountsDB.GetChatAddress()
	if err != nil {
		return err
	}
	walletAddr, err := accountsDB.GetWalletAddress()
	if err != nil {
		return err
	}

	login := LoginParams{
		Password:    password,
		ChatAddress: chatAddr,
		MainAccount: walletAddr,
	}

	err = b.SelectAccount(login, chatKey)
	if err != nil {
		return err
	}

	err = b.multiaccountsDB.UpdateAccountTimestamp(acc.KeyUID, time.Now().Unix())
	if err != nil {
		b.logger.Info("failed to update account")
		return err
	}

	return nil
}

func (b *StatusBackend) accountsDB() (*accounts.Database, error) {
	return accounts.NewDB(b.appDB)
}

func (b *StatusBackend) GetSettings() (*settings.Settings, error) {
	accountsDB, err := b.accountsDB()
	if err != nil {
		return nil, err
	}

	s, err := accountsDB.GetSettings()
	if err != nil {
		return nil, err
	}

	return &s, nil
}

func (b *StatusBackend) GetEnsUsernames() ([]*ens.UsernameDetail, error) {
	db := ens.NewEnsDatabase(b.appDB)
	removed := false
	return db.GetEnsUsernames(&removed)
}

func (b *StatusBackend) StartNodeWithAccount(acc multiaccounts.Account, password string, nodecfg *params.NodeConfig, chatKey *ecdsa.PrivateKey) error {
	err := b.startNodeWithAccount(acc, password, nodecfg, chatKey)
	if err != nil {
		// Stop node for clean up
		_ = b.StopNode()
	}
	// get logged in
	if !b.LocalPairingStateManager.IsPairing() {
		return b.LoggedIn(acc.KeyUID, err)
	}
	return err
}

func (b *StatusBackend) LoggedIn(keyUID string, err error) error {
	if err != nil {
		signal.SendLoggedIn(nil, nil, nil, err)
		return err
	}
	s, err := b.GetSettings()
	if err != nil {
		return err
	}
	acc, err := b.getAccountByKeyUID(keyUID)
	if err != nil {
		return err
	}

	ensUsernames, err := b.GetEnsUsernames()
	if err != nil {
		return err
	}
	var ensUsernamesJSON json.RawMessage
	if ensUsernames != nil {
		ensUsernamesJSON, err = json.Marshal(ensUsernames)
		if err != nil {
			return err
		}
	}
	signal.SendLoggedIn(acc, s, ensUsernamesJSON, nil)
	return nil
}

func (b *StatusBackend) ExportUnencryptedDatabase(acc multiaccounts.Account, password, directory string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.appDB != nil {
		return nil
	}
	if len(b.rootDataDir) == 0 {
		return errors.New("root datadir wasn't provided")
	}

	dbPath, err := b.runDBFileMigrations(acc, password)
	if err != nil {
		return err
	}

	err = sqlite.DecryptDB(dbPath, directory, password, acc.KDFIterations)
	if err != nil {
		b.logger.Error("failed to initialize db", zap.Error(err))
		return err
	}
	return nil
}

func (b *StatusBackend) ImportUnencryptedDatabase(acc multiaccounts.Account, password, databasePath string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.appDB != nil {
		return nil
	}

	path, err := b.getAppDBPath(acc.KeyUID)
	if err != nil {
		return err
	}

	err = sqlite.EncryptDB(databasePath, path, password, acc.KDFIterations, signal.SendReEncryptionStarted, signal.SendReEncryptionFinished)
	if err != nil {
		b.logger.Error("failed to initialize db", zap.Error(err))
		return err
	}
	return nil
}

func (b *StatusBackend) reEncryptKeyStoreDir(currentPassword string, newPassword string) error {
	err := b.accountsManager.ReEncryptKeyStoreDir(currentPassword, newPassword)
	if err != nil {
		return fmt.Errorf("ReEncryptKeyStoreDir error: %v", err)
	}
	return nil
}

func (b *StatusBackend) ChangeDatabasePassword(keyUID string, password string, newPassword string) error {
	acc, err := b.multiaccountsDB.GetAccount(keyUID)
	if err != nil {
		return err
	}

	internalDbPath, err := dbsetup.GetDBFilename(b.appDB)
	if err != nil {
		return fmt.Errorf("failed to get database file name, %w", err)
	}

	appDBPath, err := b.getAppDBPath(keyUID)
	if err != nil {
		return err
	}

	// In order to overcome Mac OS symlink issue, we check if the internalDbPath contains the appDBPath.
	// Cause on macOS, `/var` is actually a symlink to `/private/var`.
	isCurrentAccount := strings.Contains(internalDbPath, appDBPath)

	restartNode := func() {
		if isCurrentAccount {
			pass := password
			if err == nil {
				pass = newPassword
			}

			err := b.StopNode()
			if err != nil {
				b.logger.Error("failed to stop node", zap.Error(err))
				return
			}

			// TODO https://github.com/status-im/status-go/issues/3906
			// Fix restarting node, as it always fails but the error is ignored
			// because UI calls Logout and Quit afterwards. It should not be UI-dependent
			// and should be handled gracefully here if it makes sense to run dummy node after
			// logout
			err = b.startNodeWithAccount(*acc, pass, b.config, nil)
			if err != nil {
				b.logger.Error("failed to start node", zap.Error(err))
				return
			}
		}
	}
	defer restartNode()

	logout := func() {
		if isCurrentAccount {
			_ = b.Logout()
		}
	}
	noLogout := func() {}

	// First change app DB password, because it also reencrypts the keystore,
	// otherwise if we call changeWalletDbPassword first and logout, we will fail
	// to reencrypt	the keystore
	err = b.changeAppDBPassword(acc, logout, password, newPassword)
	if err != nil {
		return err
	}

	// Already logged out but pass a param to decouple the logic for testing
	err = b.changeWalletDBPassword(acc, noLogout, password, newPassword)
	if err != nil {
		// Revert the password to original
		err2 := b.changeAppDBPassword(acc, noLogout, newPassword, password)
		if err2 != nil {
			b.logger.Error("failed to revert app db password", zap.Error(err2))
		}

		return err
	}

	return nil
}

func (b *StatusBackend) changeAppDBPassword(account *multiaccounts.Account, logout func(), password string, newPassword string) error {
	tmpDbPath, cleanup, err := b.createTempDBFile("v4.db")
	if err != nil {
		return err
	}
	defer cleanup()

	dbPath, err := b.getAppDBPath(account.KeyUID)
	if err != nil {
		return err
	}

	// Exporting database to a temporary file with a new password
	err = sqlite.ExportDB(dbPath, password, account.KDFIterations, tmpDbPath, newPassword, signal.SendReEncryptionStarted, signal.SendReEncryptionFinished)
	if err != nil {
		return err
	}

	err = b.reEncryptKeyStoreDir(password, newPassword)
	if err != nil {
		return err
	}

	// Replacing the old database with the new one requires closing all connections to the database
	// This is done by stopping the node and restarting it with the new DB
	logout()

	// Replacing the old database files with the new ones, ignoring the wal and shm errors
	replaceCleanup, err := replaceDBFile(dbPath, tmpDbPath)
	if replaceCleanup != nil {
		defer replaceCleanup()
	}

	if err != nil {
		// Restore the old account
		_ = b.reEncryptKeyStoreDir(newPassword, password)
		return err
	}

	return nil
}

func (b *StatusBackend) changeWalletDBPassword(account *multiaccounts.Account, logout func(), password string, newPassword string) error {
	tmpDbPath, cleanup, err := b.createTempDBFile("wallet.db")
	if err != nil {
		return err
	}
	defer cleanup()

	dbPath, err := b.getWalletDBPath(account.KeyUID)
	if err != nil {
		return err
	}

	// Exporting database to a temporary file with a new password
	err = sqlite.ExportDB(dbPath, password, account.KDFIterations, tmpDbPath, newPassword, signal.SendReEncryptionStarted, signal.SendReEncryptionFinished)
	if err != nil {
		return err
	}

	// Replacing the old database with the new one requires closing all connections to the database
	// This is done by stopping the node and restarting it with the new DB
	logout()

	// Replacing the old database files with the new ones, ignoring the wal and shm errors
	replaceCleanup, err := replaceDBFile(dbPath, tmpDbPath)
	if replaceCleanup != nil {
		defer replaceCleanup()
	}
	return err
}

func (b *StatusBackend) createTempDBFile(pattern string) (tmpDbPath string, cleanup func(), err error) {
	if len(b.rootDataDir) == 0 {
		err = errors.New("root datadir wasn't provided")
		return
	}
	rootDataDir := b.rootDataDir
	//On iOS, the rootDataDir value does not contain a trailing slash.
	//This is causing an incorrectly formatted temporary file path to be generated, leading to an "operation not permitted" error.
	//e.g. value of rootDataDir is `/var/mobile/.../12906D5A-E831-49E9-BBE7-5FFE8E805D8A/Library`,
	//the file path generated is something like `/var/mobile/.../12906D5A-E831-49E9-BBE7-5FFE8E805D8A/123-v4.db`
	//which removed `Library` from the path.
	if !strings.HasSuffix(rootDataDir, "/") {
		rootDataDir += "/"
	}
	file, err := os.CreateTemp(filepath.Dir(rootDataDir), "*-"+pattern)
	if err != nil {
		return
	}
	err = file.Close()
	if err != nil {
		return
	}

	tmpDbPath = file.Name()
	cleanup = func() {
		filePath := file.Name()
		_ = os.Remove(filePath)
		_ = os.Remove(filePath + "-wal")
		_ = os.Remove(filePath + "-shm")
		_ = os.Remove(filePath + "-journal")
	}
	return
}

func replaceDBFile(dbPath string, newDBPath string) (cleanup func(), err error) {
	err = os.Rename(newDBPath, dbPath)
	if err != nil {
		return
	}

	cleanup = func() {
		_ = os.Remove(dbPath + "-wal")
		_ = os.Remove(dbPath + "-shm")
		_ = os.Rename(newDBPath+"-wal", dbPath+"-wal")
		_ = os.Rename(newDBPath+"-shm", dbPath+"-shm")
	}

	return
}

func (b *StatusBackend) ConvertToKeycardAccount(account multiaccounts.Account, s settings.Settings, keycardUID string, oldPassword string, newPassword string) error {
	messenger := b.Messenger()
	if messenger == nil {
		return errors.New("cannot resolve messenger instance")
	}

	err := b.multiaccountsDB.UpdateAccountKeycardPairing(account.KeyUID, account.KeycardPairing)
	if err != nil {
		return err
	}

	err = b.ensureDBsOpened(account, oldPassword)
	if err != nil {
		return err
	}

	accountDB, err := accounts.NewDB(b.appDB)
	if err != nil {
		return err
	}

	keypair, err := accountDB.GetKeypairByKeyUID(account.KeyUID)
	if err != nil {
		if err == accsmanagementtypes.ErrDbKeypairNotFound {
			return errors.New("cannot convert an unknown keypair")
		}
		return err
	}

	err = accountDB.SaveSettingField(settings.KeycardInstanceUID, s.KeycardInstanceUID)
	if err != nil {
		return err
	}

	err = accountDB.SaveSettingField(settings.KeycardPairedOn, s.KeycardPairedOn)
	if err != nil {
		return err
	}

	err = accountDB.SaveSettingField(settings.KeycardPairing, s.KeycardPairing)
	if err != nil {
		return err
	}

	err = accountDB.SaveSettingField(settings.Mnemonic, nil)
	if err != nil {
		return err
	}

	err = accountDB.SaveSettingField(settings.ProfileMigrationNeeded, false)
	if err != nil {
		return err
	}

	displayName, err := accountDB.DisplayName()
	if err != nil {
		return err
	}

	kc := accsmanagementtypes.Keycard{
		KeycardUID:    keycardUID,
		KeycardName:   displayName,
		KeycardLocked: false,
		KeyUID:        account.KeyUID,
	}

	for _, acc := range keypair.Accounts {
		kc.AccountsAddresses = append(kc.AccountsAddresses, acc.Address)
	}
	err = messenger.SaveOrUpdateKeycard(context.Background(), &kc, oldPassword)
	if err != nil {
		return err
	}

	err = b.closeDBs()
	if err != nil {
		return err
	}

	return b.ChangeDatabasePassword(account.KeyUID, oldPassword, newPassword)
}

// CreateAccountAndLogin creates a new account and logs in with it.
func (b *StatusBackend) CreateAccountAndLogin(request *requests2.CreateAccount) (*multiaccounts.Account, error) {
	validation := &requests2.CreateAccountValidation{
		AllowEmptyDisplayName: true,
	}
	if err := request.Validate(validation); err != nil {
		return nil, err
	}

	mnemonic, err := accscommon.CreateRandomMnemonicWithDefaultLength()
	if err != nil {
		return nil, err
	}

	return b.StartNodeWithChatKeyOrMnemonic(
		request,
		mnemonic,
		nil,
		false,
	)
}

// RestoreAccountAndLogin restores an account and logs in with it.
func (b *StatusBackend) RestoreAccountAndLogin(request *requests.RestoreAccount) (*multiaccounts.Account, error) {
	if err := request.Validate(false); err != nil {
		return nil, err
	}

	return b.StartNodeWithChatKeyOrMnemonic(
		&request.CreateAccount,
		request.Mnemonic,
		nil,
		true,
	)
}

func (b *StatusBackend) RestoreKeycardAccountAndLogin(request *requests.RestoreAccount) (*multiaccounts.Account, error) {
	if err := request.Validate(true); err != nil {
		return nil, err
	}

	return b.StartNodeWithChatKeyOrMnemonic(
		&request.CreateAccount,
		request.Mnemonic,
		request.Keycard,
		true,
	)
}

func (b *StatusBackend) GetKeyUIDByMnemonic(mnemonic string) (string, error) {
	genAccount, err := generator.CreateAccountFromMnemonic(mnemonic, "")
	if err != nil {
		return "", err
	}

	accInfo := genAccount.ToIdentifiedAccountInfo()

	return accInfo.KeyUID, nil
}

func (b *StatusBackend) generateAccount(mnemonic string) (genAcc *generator.Account, accInfo generator.GeneratedAccountInfo, err error) {
	finalMnemonic := mnemonic
	if mnemonic == "" {
		finalMnemonic, err = accscommon.CreateRandomMnemonicWithDefaultLength()
		if err != nil {
			return
		}
	}

	genAcc, err = generator.CreateAccountFromMnemonic(finalMnemonic, "")
	if err != nil {
		return
	}

	accInfo = genAcc.ToGeneratedAccountInfo(finalMnemonic)
	return
}

func generateDerivedAddresses(genAcc *generator.Account, paths []string) (genDerivedAccounts map[string]*generator.Account, genDerivedAccountsInfo map[string]generator.AccountInfo, err error) {
	genDerivedAccounts, err = generator.DeriveChildrenFromAccount(genAcc, paths)
	if err != nil {
		return
	}

	genDerivedAccountsInfo = make(map[string]generator.AccountInfo, 0)
	for path, acc := range genDerivedAccounts {
		genDerivedAccountsInfo[path] = acc.ToAccountInfo()
	}

	return
}

func (b *StatusBackend) buildAccount(request *requests2.CreateAccount, keyUID string, chatKey string) (*multiaccounts.Account, error) {
	err := b.OpenAccounts(request.ThirdpartyServicesEnabled)
	if err != nil {
		return nil, err
	}

	acc := &multiaccounts.Account{
		KeyUID:                  keyUID,
		Name:                    request.DisplayName,
		CustomizationColor:      multiacccommon.CustomizationColor(request.CustomizationColor),
		CustomizationColorClock: 1,
		KDFIterations:           request.KdfIterations,
		Timestamp:               time.Now().Unix(),
	}

	if acc.KDFIterations == 0 {
		acc.KDFIterations = dbsetup.ReducedKDFIterationsNumber
	}

	count, err := b.multiaccountsDB.GetAccountsCount()
	if err != nil {
		return nil, err
	}
	if count == 0 {
		acc.HasAcceptedTerms = true
	}

	if request.ImagePath != "" {
		imageCropRectangle := request.ImageCropRectangle
		if imageCropRectangle == nil {
			// Default crop rectangle used by mobile
			imageCropRectangle = &requests2.ImageCropRectangle{
				Ax: 0,
				Ay: 0,
				Bx: 1000,
				By: 1000,
			}
		}

		iis, err := images.GenerateIdentityImages(request.ImagePath,
			imageCropRectangle.Ax, imageCropRectangle.Ay, imageCropRectangle.Bx, imageCropRectangle.By)

		if err != nil {
			return nil, err
		}
		acc.Images = iis
	}

	var err error
	acc.ColorHash, err = colorhash.GenerateFor(chatKey)
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate color hash")
	}

	acc.ColorID, err = identityutils.ToColorID(chatKey)
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate color id")
	}

	return acc, nil
}

func prepareDefaultSettings(request *requests2.CreateAccount, mnemonic string, keyUID string, masterAddress string,
	derivedAddresses map[string]generator.AccountInfo, restoreAccount bool) (*settings.Settings, error) {
	s, err := DefaultSettings(keyUID, masterAddress, derivedAddresses)
	if err != nil {
		return nil, err
	}

	s.DeviceName = request.DeviceName
	s.DisplayName = request.DisplayName
	s.PreviewPrivacy = request.PreviewPrivacy
	//s.CurrentNetwork = request.CurrentNetwork
	//s.TestNetworksEnabled = request.TestNetworksEnabled
	//s.AutoRefreshTokensEnabled = request.AutoRefreshTokensEnabled
	if !restoreAccount {
		s.Mnemonic = &mnemonic
		s.MnemonicWasNotShown = true
	}

	//if request.WakuV2Fleet != "" {
	//	s.Fleet = &request.WakuV2Fleet
	//}

	s.ThirdpartyServicesEnabled = request.ThirdpartyServicesEnabled

	return s, nil
}

func prepareConfig(request *requests2.CreateAccount, keyUID string, installationID string) (*params.NodeConfig, error) {
	nodeConfig, err := DefaultNodeConfig(installationID, keyUID, request)
	if err != nil {
		return nil, err
	}

	return nodeConfig, nil
}

func (b *StatusBackend) prepareWalletAccount(request *requests2.CreateAccount) *accsmanagementtypes.AccountCreationDetails {
	return &accsmanagementtypes.AccountCreationDetails{
		Path:    accscommon.PathDefaultWalletAccount,
		Name:    api.walletAccountDefaultName,
		ColorID: request.CustomizationColor,
	}
}

func prepareKeypair(request *requests2.CreateAccount, keyUID string, masterAddress string,
	derivedAddresses map[string]generator.AccountInfo, restoreAccount bool) (keypair *accsmanagementtypes.Keypair, err error) {
	// set up keypair
	keypair = &accsmanagementtypes.Keypair{
		Name:                    request.DisplayName,
		KeyUID:                  keyUID,
		Type:                    accsmanagementtypes.KeypairTypeProfile,
		DerivedFrom:             masterAddress,
		LastUsedDerivationIndex: 0,
	}

	// add chat account
	chatDerivedAccount := derivedAddresses[accscommon.PathEIP1581Chat]
	keypair.Accounts = append(keypair.Accounts, &accsmanagementtypes.Account{
		PublicKey: types.Hex2Bytes(chatDerivedAccount.PublicKey),
		KeyUID:    keypair.KeyUID,
		Address:   types.HexToAddress(chatDerivedAccount.Address),
		Chat:      true,
		Path:      accscommon.PathEIP1581Chat,
		Position:  -1, // When creating a new account, the chat account should have position -1, cause it doesn't participate
		Operable:  accsmanagementtypes.AccountFullyOperable,
	})

	// add wallet account
	walletDerivedAccount := derivedAddresses[accscommon.PathDefaultWalletAccount]
	keypair.Accounts = append(keypair.Accounts, &accsmanagementtypes.Account{
		PublicKey:          types.Hex2Bytes(walletDerivedAccount.PublicKey),
		KeyUID:             keypair.KeyUID,
		Address:            types.HexToAddress(walletDerivedAccount.Address),
		ColorID:            multiacccommon.CustomizationColor(request.CustomizationColor),
		Wallet:             true,
		Path:               accscommon.PathDefaultWalletAccount,
		Name:               walletAccountDefaultName,
		AddressWasNotShown: !restoreAccount,
		Position:           0, // When creating a new account, the wallet account should have position 0, cause it's the default wallet account
		Operable:           accsmanagementtypes.AccountFullyOperable,
	})

	return
}

func (b *StatusBackend) prepareForKeycard(request *requests2.CreateAccount, multiAccount *multiaccounts.Account,
	settings *settings.Settings) error {
	if request.KeycardInstanceUID == "" {
		return nil
	}

	if request.KeycardPairingKey != "" {
		// KeycardPairingKey is used only on mobile
		settings.KeycardPairing = request.KeycardPairingKey
		multiAccount.KeycardPairing = request.KeycardPairingKey
	} else {
		// KeycardPairingDataFile is used only on desktop
		keycardPairingDataFile := filepath.Join(b.rootDataDir, DefaultKeycardPairingDataFileRelativePath)
		if request.KeycardPairingDataFile != nil {
			keycardPairingDataFile = *request.KeycardPairingDataFile
		}

		// KeycardPairingDataFile is used only on desktop
		kp := wallet.NewKeycardPairings()
		kp.SetKeycardPairingsFile(keycardPairingDataFile)
		pairings, err := kp.GetPairings()
		if err != nil {
			return errors.Wrap(err, "failed to get keycard pairings")
		}

		keycard, ok := pairings[request.KeycardInstanceUID]
		if !ok {
			return errors.New("keycard not found in pairings file")
		}

		settings.KeycardPairing = keycard.Key
		multiAccount.KeycardPairing = keycard.Key
	}

	settings.KeycardInstanceUID = request.KeycardInstanceUID
	settings.KeycardPairedOn = time.Now().Unix()

	return nil
}

func (b *StatusBackend) ConvertToRegularAccount(mnemonic string, currPassword string, newPassword string) error {
	messenger := b.Messenger()
	if messenger == nil {
		return errors.New("cannot resolve messenger instance")
	}

	mnemonicNoExtraSpaces := strings.Join(strings.Fields(mnemonic), " ")
	_, generatedAccountInfo, err := b.generateAccount(mnemonicNoExtraSpaces)
	if err != nil {
		return err
	}

	kdfIterations, err := b.multiaccountsDB.GetAccountKDFIterationsNumber(generatedAccountInfo.KeyUID)
	if err != nil {
		return err
	}

	err = b.ensureDBsOpened(multiaccounts.Account{KeyUID: generatedAccountInfo.KeyUID, KDFIterations: kdfIterations}, currPassword)
	if err != nil {
		return err
	}

	db, err := accounts.NewDB(b.appDB)
	if err != nil {
		return err
	}

	knownAccounts, err := db.GetActiveAccounts()
	if err != nil {
		return err
	}

	// We add these two paths, cause others will be added via `StoreAccount` function call
	var paths []string
	paths = append(paths, accscommon.PathWalletRoot, accscommon.PathEIP1581Root)
	for _, acc := range knownAccounts {
		if generatedAccountInfo.KeyUID == acc.KeyUID {
			paths = append(paths, acc.Path)
		}
	}

	_, _, err = b.accountsManager.StoreKeystoreFilesForMnemonic(mnemonicNoExtraSpaces, currPassword, paths)
	if err != nil {
		return err
	}

	err = b.multiaccountsDB.UpdateAccountKeycardPairing(generatedAccountInfo.KeyUID, "")
	if err != nil {
		return err
	}

	err = messenger.DeleteAllKeycardsWithKeyUID(context.Background(), generatedAccountInfo.KeyUID)
	if err != nil {
		return err
	}

	err = db.SaveSettingField(settings.KeycardInstanceUID, "")
	if err != nil {
		return err
	}

	err = db.SaveSettingField(settings.KeycardPairedOn, 0)
	if err != nil {
		return err
	}

	err = db.SaveSettingField(settings.KeycardPairing, "")
	if err != nil {
		return err
	}

	err = db.SaveSettingField(settings.ProfileMigrationNeeded, false)
	if err != nil {
		return err
	}

	err = b.closeDBs()
	if err != nil {
		return err
	}

	return b.ChangeDatabasePassword(generatedAccountInfo.KeyUID, currPassword, newPassword)
}

func (b *StatusBackend) VerifyProfilePassword(password string) (bool, error) {
	return b.services.AccountService().VerifyPassword(password)
}

func (b *StatusBackend) VerifyDatabasePassword(keyUID string, password string) error {
	kdfIterations, err := b.multiaccountsDB.GetAccountKDFIterationsNumber(keyUID)
	if err != nil {
		return err
	}

	if !b.appDBExists(keyUID) || !b.walletDBExists(keyUID) {
		return errors.New("one or more databases not created")
	}

	err = b.ensureDBsOpened(multiaccounts.Account{KeyUID: keyUID, KDFIterations: kdfIterations}, password)
	if err != nil {
		return err
	}

	err = b.closeDBs()
	if err != nil {
		return err
	}

	return nil
}

//func EnrichMultiAccountByPublicKey(account *multiaccounts.Account, chatPublicKey types.HexBytes) error {
//	pk := string(chatPublicKey.Bytes())
//	colorHash, err := colorhash.GenerateFor(pk)
//	if err != nil {
//		return err
//	}
//	account.ColorHash = colorHash
//
//	colorID, err := identityutils.ToColorID(pk)
//	if err != nil {
//		return err
//	}
//	account.ColorID = colorID
//
//	return nil
//}

type accountWithInfo struct {
	keyUID           string
	masterAddress    string
	derivedAddresses map[string]generator.AccountInfo
}

func (b *StatusBackend) deriveFromKeycard(keycardData *requests2.KeycardData) *accountWithInfo {
	derivedAddresses := map[string]generator.AccountInfo{
		accscommon.PathWalletRoot: {
			Address: keycardData.WalletRootAddress,
		},
		accscommon.PathEIP1581Root: {
			Address: keycardData.Eip1581Address,
		},
		accscommon.PathEIP1581Chat: {
			Address:    keycardData.WhisperAddress,
			PublicKey:  keycardData.WhisperPublicKey,
			PrivateKey: keycardData.WhisperPrivateKey,
		},
		accscommon.PathDefaultWalletAccount: {
			Address:   keycardData.WalletAddress,
			PublicKey: keycardData.WalletPublicKey,
		},
		accscommon.PathEIP1581Encryption: {
			PublicKey: keycardData.EncryptionPublicKey,
		},
	}

	return &accountWithInfo{
		keyUID:           keycardData.KeyUID,
		masterAddress:    keycardData.Address,
		derivedAddresses: derivedAddresses,
	}
}

func deriveFromMnemonic(mnemonic string) (info *accountWithInfo, err error) {
	genMasterAcc, err := generator.CreateAccountFromMnemonic(mnemonic, "")
	if err != nil {
		return nil, err
	}

	derivationPaths := []string{
		accscommon.PathWalletRoot,
		accscommon.PathEIP1581Root,
		accscommon.PathEIP1581Chat,
		accscommon.PathDefaultWalletAccount,
		accscommon.PathEIP1581Encryption,
	}
	_, derivedAddresses, err := generateDerivedAddresses(genMasterAcc, derivationPaths)
	if err != nil {
		return nil, err
	}

	return &accountWithInfo{
		keyUID:           genMasterAcc.KeyUID(),
		masterAddress:    genMasterAcc.Address().Hex(),
		derivedAddresses: derivedAddresses,
	}, nil
}

func (b *StatusBackend) StartNodeWithChatKeyOrMnemonic(
	ctx context.Context,
	request *requests2.CreateAccount,
	mnemonic string, // empty mnemonic is used for keycard account, not empty for regular account
	keycardData *requests2.KeycardData, // nil for regular account, not nil for account with already set keycard
	restoreAccount bool,
) (*multiaccounts.Account, error) {
	var (
		err         error
		isKeycard   = request.KeycardInstanceUID != ""
		accountInfo *accountWithInfo
		//chatPrivateKey         *ecdsa.PrivateKey // set only for keycard account
		chatPublicKey          string
		keypairToStoreDirectly *accsmanagementtypes.Keypair
	)

	if keycardData != nil { // means that the keycard is already set, details already on it
		accountInfo = b.deriveFromKeycard(keycardData)
	} else {
		accountInfo, err = deriveFromMnemonic(mnemonic)
		if err != nil {
			return nil, err
		}
	}

	var (
		keyUID           = accountInfo.keyUID
		masterAddress    = accountInfo.masterAddress
		derivedAddresses = accountInfo.derivedAddresses
	)

	if isKeycard {
		genChatAccount, err := generator.CreateAccountFromPrivateKey(derivedAddresses[accscommon.PathEIP1581Chat].PrivateKey)
		if err != nil {
			return nil, err
		}

		//chatPrivateKey = genChatAccount.PrivateKey()
		chatPublicKey = genChatAccount.PublicKeyHex()

		request.Password = derivedAddresses[accscommon.PathEIP1581Encryption].PublicKey
	} else {
		chatPublicKey = derivedAddresses[accscommon.PathEIP1581Chat].PublicKey
	}

	settings, err := prepareDefaultSettings(request, mnemonic, keyUID, masterAddress, derivedAddresses, restoreAccount)
	if err != nil {
		return nil, errors.Wrap(err, "failed to prepare settings")
	}

	acc, err := b.buildAccount(request, keyUID, chatPublicKey)
	if err != nil {
		return nil, err
	}

	if acc.Name == "" {
		acc.Name = settings.Name
	}

	//err = EnrichMultiAccountByPublicKey(acc, chatPublicKey)
	//if err != nil {
	//	return nil, err
	//}

	//nodeConfig, err := prepareConfig(request, keyUID, settings.InstallationID)
	//if err != nil {
	//	return nil, errors.Wrap(err, "failed to prepare node config")
	//}

	// Create app database
	appDB, err := b.createAppDatabase(acc.KeyUID, acc.KDFIterations, request.Password)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create app database")
	}
	defer func() {
		err := appDB.Close()
		if err != nil {
			b.logger.Error("failed to close app database", zap.Error(err))
		}
	}()

	accdb, err := accounts.NewDB(appDB)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create accounts db")
	}

	// Create wallet database
	walletDB, err := b.createWalletDatabase(acc.KeyUID, acc.KDFIterations, request.Password)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create wallet database")
	}
	defer func() {
		err := walletDB.Close()
		if err != nil {
			b.logger.Error("failed to close wallet database", zap.Error(err))
		}
	}()

	// Set accounts management persistence
	accountsManager, err := accsmanagement.NewAccountsManager(b.logger.Named("accounts-manager"))
	if err != nil {
		return nil, errors.Wrap(err, "failed to create AccountsManager")
	}

	accountsManager.SetRootDataDir(b.rootDataDir)
	accountsManager.SetPersistence(accdb)
	defer accountsManager.Logout()

	//

	if isKeycard {
		err = b.prepareForKeycard(request, acc, settings)
		if err != nil {
			return nil, errors.Wrap(err, "failed to prepare for keycard")
		}

		keypairToStoreDirectly, err = prepareKeypair(request, keyUID, masterAddress, derivedAddresses, restoreAccount)
		if err != nil {
			return nil, errors.Wrap(err, "failed to prepare keypair")
		}
	} else {
		walletAccount := b.prepareWalletAccount(request)
		_, err := accountsManager.CreateKeypairFromMnemonicAndStore(mnemonic, request.Password,
			request.DisplayName, walletAccount, true, 0)
		if err != nil {
			return nil, err
		}
	}

	//err = b.StartNodeWithAccountAndInitialConfig(
	//	acc,
	//	request.Password,
	//	*settings,
	//	nodeConfig,
	//	keypairToStoreDirectly,
	//	chatPrivateKey,
	//)

	err = b.multiaccountsDB.SaveAccount(*acc)
	if err != nil {
		return nil, errors.Wrap(err, "failed to save account")
	}

	err = accdb.CreateSettings(*settings, params.NodeConfig{}) // FIXME: Remove deprecated NodeConfig argument
	if err != nil {
		return nil, errors.Wrap(err, "failed to create settings")
	}

	if keypairToStoreDirectly != nil {
		err = accdb.SaveOrUpdateKeypair(keypairToStoreDirectly)
		if err != nil {
			return nil, errors.Wrap(err, "failed to save keypair")
		}
	}

	return acc, err
}

func (b *StatusBackend) createAppDatabase(keyUID string, kdfIterations int, password string) (*sql.DB, error) {
	// WARNING: Decide if we want to drop this migration
	//dbFilePath, err := s.runDBFileMigrations(account, password)
	//if err != nil {
	//	return errors.New("Failed to migrate db file: " + err.Error())
	//}

	dbFilePath, err := b.getAppDBPath(keyUID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get database file path")
	}

	db, err := appdatabase.InitializeDB(dbFilePath, password, kdfIterations)
	if err != nil {
		b.logger.Error("failed to initialize db", zap.Error(err))
		return nil, errors.Wrap(err, "failed to initialize db")
	}

	//accountsDB, err := accounts.NewDB(s.db)
	//if err != nil {
	//	s.logger.Error("failed to create new *Database instance", zap.Error(err))
	//	return
	//}
	//s.accountsManager.SetPersistence(accountsDB)

	return db, nil
}

func (b *StatusBackend) createWalletDatabase(keyUID string, kdfIterations int, password string) (*sql.DB, error) {
	dbWalletPath, err := b.getWalletDBPath(keyUID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get wallet database file path")
	}

	db, err := walletdatabase.InitializeDB(dbWalletPath, password, kdfIterations)
	if err != nil {
		b.logger.Error("failed to initialize wallet db", zap.Error(err))
		return nil, errors.Wrap(err, "failed to initialize wallet db")
	}

	return db, nil
}

func (b *StatusBackend) StartNodeWithAccountAndInitialConfig(
	multiAccount *multiaccounts.Account,
	password string,
	settings settings.Settings,
	nodecfg *params.NodeConfig,
	keypair *accsmanagementtypes.Keypair,
	chatKey *ecdsa.PrivateKey,
) error {
	err := b.ensureDBsOpened(*multiAccount, password)
	if err != nil {
		return err
	}

	err = b.SaveAccount(*multiAccount)
	if err != nil {
		return err
	}

	err = b.saveKeypairAndSettings(settings, nodecfg, keypair)
	if err != nil {
		return err
	}

	err = b.StartNodeWithAccount(*multiAccount, password, nodecfg, chatKey)
	if err != nil {
		b.logger.Error("start node with account and initial config", zap.Error(err))
		return err
	}

	return nil
}

func (b *StatusBackend) saveKeypairAndSettings(settings settings.Settings, nodecfg *params.NodeConfig, keypair *accsmanagementtypes.Keypair) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	accdb, err := accounts.NewDB(b.appDB)
	if err != nil {
		return err
	}
	err = accdb.CreateSettings(settings, *nodecfg)
	if err != nil {
		return err
	}

	if keypair != nil {
		return accdb.SaveOrUpdateKeypair(keypair)
	}

	return nil
}

func (b *activeAccount) loadNodeConfig(inputNodeCfg *params.NodeConfig) (*params.NodeConfig, error) {
	conf, err := nodecfg.GetNodeConfigFromDB(b.appDB)
	if err != nil {
		return nil, err
	}

	if inputNodeCfg != nil {
		// If an installationID is provided, we override it
		if conf != nil && conf.ShhextConfig.InstallationID != "" {
			inputNodeCfg.ShhextConfig.InstallationID = conf.ShhextConfig.InstallationID
		}

		conf, err = b.OverwriteNodeConfigValues(conf, inputNodeCfg)
		if err != nil {
			return nil, errors.Wrap(err, "failed to overwrite node config")
		}

		if inputNodeCfg.RuntimeLogLevel != "" {
			conf.LogLevel = inputNodeCfg.RuntimeLogLevel
		}
	}

	//conf.RootDataDir = b.rootDataDir

	//if _, err = os.Stat(conf.RootDataDir); os.IsNotExist(err) {
	//	if err := os.MkdirAll(conf.RootDataDir, os.ModePerm); err != nil {
	//		b.logger.Warn("failed to create data directory", zap.Error(err))
	//		return err
	//	}
	//}

	return conf, nil
}

func (b *StatusBackend) GetNodeConfig() (*params.NodeConfig, error) {
	return nodecfg.GetNodeConfigFromDB(b.appDB)
}

func (b *StatusBackend) startNode(config *params.NodeConfig) (err error) {
	b.logger.Info("status-go version details",
		zap.String("version", version.Version()),
		zap.String("commit", version.GitCommit()))
	b.logger.Debug("starting node with config", zap.Stringer("config", config))
	// Update config with some defaults.
	if err := config.UpdateWithDefaults(); err != nil {
		return err
	}

	// Updating node config
	b.config = config

	b.logger.Debug("updated config with defaults", zap.Stringer("config", config))

	// Start by validating configuration
	if err := config.Validate(); err != nil {
		return err
	}

	if err = b.services.Start(config); err != nil {
		return
	}

	b.transactor.SetEthClientGetter(b.services.RPCClient(), rpc.DefaultCallTimeout)

	signal.SendNodeStarted()

	if b.services.WalletService() != nil {
		b.services.WalletService().KeycardPairings().SetKeycardPairingsFile(config.KeycardPairingDataFile)
	}

	if b.prometheusMetrics != nil {
		b.prometheusMetrics.RegisterHandler("waku", b.wakuMetricsHandler())
	}

	signal.SendNodeReady()
	return nil
}

// StopNode stop Status node. Stopped node cannot be resumed.
func (b *StatusBackend) StopNode() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.stopNode()
}

func (b *StatusBackend) stopNode() error {
	if b.services == nil || !b.IsNodeRunning() {
		return nil
	}
	if !b.LocalPairingStateManager.IsPairing() {
		defer signal.SendNodeStopped()
	}

	return b.services.Stop()
}

// RestartNode restart running Status node, fails if node is not running
func (b *StatusBackend) RestartNode() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if !b.IsNodeRunning() {
		return ErrNoRunningNode
	}

	if err := b.stopNode(); err != nil {
		return err
	}

	return b.startNode(b.config)
}

// CallRPC executes public RPC requests on node's in-proc RPC server.
func (b *StatusBackend) CallInProcessRPC(inputJSON string) string {
	return b.services.CallInProcessRPC(inputJSON)
}

// @deprecated
// SendTransaction creates a new transaction and waits until it's complete.
func (b *StatusBackend) SendTransaction(sendArgs wallettypes.SendTxArgs, password string) (hash types.Hash, err error) {
	return types.Hash{}, errors.New("method not supported")
}

func (b *StatusBackend) SendTransactionWithChainID(chainID uint64, sendArgs wallettypes.SendTxArgs, password string) (hash types.Hash, err error) {
	verifiedAccount, err := b.getVerifiedWalletAccount(sendArgs.From.String(), password)
	if err != nil {
		return hash, err
	}

	hash, _, err = b.transactor.SendTransactionWithChainID(chainID, sendArgs, -1, verifiedAccount)
	return hash, err
}

// @deprecated
func (b *StatusBackend) SendTransactionWithSignature(sendArgs wallettypes.SendTxArgs, sig []byte) (hash types.Hash, err error) {
	return types.Hash{}, errors.New("method not supported")
}

// @deprecated
// HashTransaction validate the transaction and returns new sendArgs and the transaction hash.
func (b *StatusBackend) HashTransaction(sendArgs wallettypes.SendTxArgs) (wallettypes.SendTxArgs, types.Hash, error) {
	return wallettypes.SendTxArgs{}, types.Hash{}, errors.New("method not supported")
}

// SignMessage checks the pwd vs the selected account and passes on the signParams
// to personalAPI for message signature
func (b *StatusBackend) SignMessage(rpcParams personal.SignParams) (types.HexBytes, error) {
	verifiedAccount, err := b.getVerifiedWalletAccount(rpcParams.Address, rpcParams.Password)
	if err != nil {
		return types.HexBytes{}, err
	}
	return b.signer.Sign(rpcParams, verifiedAccount)
}

// Recover calls the personalAPI to return address associated with the private
// key that was used to calculate the signature in the message
func (b *StatusBackend) Recover(rpcParams personal.RecoverParams) (types.Address, error) {
	return b.signer.Recover(rpcParams)
}

// HashTypedData generates the hash of TypedData.
func (b *StatusBackend) HashTypedData(typed typeddata.TypedData) (types.Hash, error) {
	chain := new(big.Int).SetUint64(b.StatusNode().Config().NetworkID)
	hash, err := typeddata.ValidateAndHash(typed, chain)
	if err != nil {
		return types.Hash{}, err
	}
	return types.Hash(hash), err
}

// HashTypedDataV4 generates the hash of TypedData.
func (b *StatusBackend) HashTypedDataV4(typed signercore.TypedData) (types.Hash, error) {
	chain := new(big.Int).SetUint64(b.StatusNode().Config().NetworkID)
	hash, err := typeddata.HashTypedDataV4(typed, chain)
	if err != nil {
		return types.Hash{}, err
	}
	return types.Hash(hash), err
}

func (b *StatusBackend) getVerifiedWalletAccount(address, password string) (*generator.Account, error) {
	return b.accountsManager.GetVerifiedWalletAccount(types.HexToAddress(address), password)
}

// ConnectionChange handles network state changes logic.
func (b *StatusBackend) ConnectionChange(typ string, expensive bool) {
	b.mu.Lock()
	defer b.mu.Unlock()

	state := connection.State{
		Type:      connection.NewType(typ),
		Expensive: expensive,
	}
	if typ == connection.None {
		state.Offline = true
	}

	b.logger.Info("Network state change", zap.Stringer("old", b.connectionState), zap.Stringer("new", state))

	if b.connectionState.Offline && !state.Offline {
		//  flush hystrix if we are going again online, since it doesn't behave
		// well when offline
		hystrix.Flush()
	}

	b.connectionState = state
	b.services.ConnectionChanged(state)

	// logic of handling state changes here
	// restart node? force peers reconnect? etc
}

// AppStateChange handles app state changes (background/foreground).
// state values: see https://facebook.github.io/react-native/docs/appstate.html
func (b *StatusBackend) AppStateChange(state api.AppState) {
	if !state.IsValid() {
		b.logger.Warn("invalid app state, not reporting app state change", zap.Any("state", state))
		return
	}

	var messenger *protocol.Messenger

	b.appState = state

	if b.services == nil {
		b.logger.Warn("services nil, not reporting app state change")
		return
	}

	if b.services.WakuV2ExtService() != nil {
		messenger = b.services.WakuV2ExtService().Messenger()
	}

	if messenger == nil {
		b.logger.Warn("messenger nil, not reporting app state change")
		return
	}

	if state == api.AppStateForeground {
		messenger.ToForeground()
	} else {
		messenger.ToBackground()
	}

	// TODO: put node in low-power mode if the app is in background (or inactive)
	// and normal mode if the app is in foreground.
}

func (b *StatusBackend) StopLocalNotifications() error {
	if b.services == nil {
		return nil
	}
	return b.services.StopLocalNotifications()
}

func (b *StatusBackend) StartLocalNotifications() error {
	if b.services == nil {
		return nil
	}
	return b.services.StartLocalNotifications()

}

// Logout clears whisper identities.
func (b *StatusBackend) Logout() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.logger.Debug("logging out")
	if b.services != nil {
		err := b.services.Cleanup()
		if err != nil {
			return err
		}
	}
	err := b.closeDBs()
	if err != nil {
		return err
	}

	b.activeAccount.accountsManager.Logout()
	b.activeAccount = nil

	if b.services != nil {
		if err := b.services.Stop(); err != nil {
			return err
		}
		b.services = nil
	}

	if !b.LocalPairingStateManager.IsPairing() {
		signal.SendNodeStopped()
	}

	// re-initialize the node, at some point we should better manage the lifecycle
	if err = b.initialize(); err != nil {
		return err
	}

	return nil
}

func (b *StatusBackend) closeDBs() error {
	err := b.closeWalletDB()
	if err != nil {
		return err
	}
	return b.closeAppDB()
}

func (b *StatusBackend) closeAppDB() error {
	if b.appDB != nil {
		err := b.appDB.Close()
		if err != nil {
			return err
		}
		b.appDB = nil
		return nil
	}
	return nil
}

func (b *StatusBackend) closeWalletDB() error {
	if b.walletDB != nil {
		err := b.walletDB.Close()
		if err != nil {
			return err
		}
		b.walletDB = nil
	}
	return nil
}

// SelectAccount selects current wallet and chat accounts, by verifying that each address has corresponding account which can be decrypted
// using provided password. Once verification is done, the decrypted chat key is injected into Whisper (as a single identity,
// all previous identities are removed).
func (b *StatusBackend) SelectAccount(loginParams LoginParams, privateKey *ecdsa.PrivateKey) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	err := b.accountsManager.SetChatAccount(loginParams.ChatAddress, loginParams.Password, privateKey)
	if err != nil {
		return err
	}

	if loginParams.MultiAccount != nil {
		b.account = loginParams.MultiAccount
	}

	if err := b.initProtocol(); err != nil {
		return err
	}

	if err = b.services.StartLocalBackup(); err != nil {
		return err
	}

	return nil
}

func (b *StatusBackend) GetActiveAccount() (*multiaccounts.Account, error) {
	if b.activeAccount == nil {
		return nil, errors.New("master key account is nil in the StatusBackend")
	}

	return b.activeAccount.account, nil
}

func (b *StatusBackend) LocalPairingStarted() error {
	if b.account == nil {
		return errors.New("master key account is nil in the StatusBackend")
	}

	accountDB, err := accounts.NewDB(b.appDB)
	if err != nil {
		return err
	}

	return accountDB.MnemonicWasShown()
}

func (services *Services) initProtocol(activeAccount *activeAccount, mediaServer *server.MediaServer, multiaccountsDB *multiaccounts.Database, metricsEnabled bool) error {
	st := services.WakuV2ExtService()
	if st == nil {
		return nil
	}
	chatAccount, err := activeAccount.accountsManager.SelectedChatAccount()
	if err != nil {
		return err
	}
	identity := chatAccount.PrivateKey()

	params := ext.InitProtocolParams{
		Identity:               identity,
		AppDB:                  activeAccount.appDB,
		WalletDB:               activeAccount.walletDB,
		HTTPServer:             mediaServer,
		MultiAccountDB:         multiaccountsDB,
		Account:                activeAccount.account,
		AccountsManager:        activeAccount.accountsManager,
		RPCClient:              services.RPCClient(),
		WalletService:          services.WalletService(),
		CommunityTokensService: services.CommunityTokensService(),
		AccountsPublisher:      services.AccountsPublisher(),
		TimeSource:             services.TimeSource(),
		MetricsEnabled:         metricsEnabled,
		TokenManager:           api.NewCommunitiesTokenManager(services.TokenManager()),
		TokenBalanceManager:    api.NewCommunitiesTokenBalanceManager(services.TokenBalancesFetcher(), services.TokenBalancesStorage()),
		NetworkManager:         api.NewCommunitiesNetworkManager(services.RPCClient().GetNetworkManager()),
	}
	err = st.InitProtocol(params)
	if err != nil {
		return err
	}

	messenger := st.Messenger()
	// Init public status api
	services.StatusPublicService().Init(messenger)
	services.AccountService().Init(messenger, activeAccount.account)
	// Init chat service
	accDB, err := accounts.NewDB(activeAccount.appDB)
	if err != nil {
		return err
	}
	services.ChatService(accDB).Init(messenger)
	services.EnsService().Init(messenger.SyncEnsNamesWithDispatchMessage)
	services.CommunityTokensService().Init(messenger)
	services.SharedUrlsService().SetDataProvider(adapters.NewSharedUrlsMessengerAdapter(messenger))
	services.NewsFeedService().SetActivityCenter(adapters.NewNewsFeedActivityCenterAdapter(messenger))

	return nil
}

func (b *StatusBackend) InstallationID() string {
	m := b.Messenger()
	if m != nil {
		return m.InstallationID()
	}
	return ""
}

func (b *StatusBackend) KeyUID() string {
	m := b.Messenger()
	if m != nil {
		return m.KeyUID()
	}
	return ""
}

// ExtractGroupMembershipSignatures extract signatures from tuples of content/signature
func (b *StatusBackend) ExtractGroupMembershipSignatures(signaturePairs [][2]string) ([]string, error) {
	return crypto.ExtractSignatures(signaturePairs)
}

// SignGroupMembership signs a piece of data containing membership information
func (b *StatusBackend) SignGroupMembership(content string) (string, error) {
	selectedChatAccount, err := b.accountsManager.SelectedChatAccount()
	if err != nil {
		return "", err
	}

	return crypto.SignStringAsHex(content, selectedChatAccount.PrivateKey())
}

func (b *StatusBackend) Messenger() *protocol.Messenger {
	statusNode := b.StatusNode()
	if statusNode != nil {
		accountService := statusNode.AccountService()
		if accountService != nil {
			return accountService.GetMessenger()
		}
	}
	return nil
}

// SignHash exposes vanilla ECDSA signing for signing a message for Swarm
func (b *StatusBackend) SignHash(hexEncodedHash string) (string, error) {
	hash, err := hexutil.Decode(hexEncodedHash)
	if err != nil {
		return "", fmt.Errorf("SignHash: could not unmarshal the input: %v", err)
	}

	chatAccount, err := b.accountsManager.SelectedChatAccount()
	if err != nil {
		return "", fmt.Errorf("SignHash: could not select account: %v", err.Error())
	}

	signature, err := ethcrypto.Sign(hash, chatAccount.PrivateKey())
	if err != nil {
		return "", fmt.Errorf("SignHash: could not sign the hash: %v", err)
	}

	hexEncodedSignature := types.EncodeHex(signature)
	return hexEncodedSignature, nil
}

func (b *StatusBackend) SwitchFleet(fleet string, conf *params.NodeConfig) error {
	if b.appDB == nil {
		return ErrDBNotAvailable
	}

	accountDB, err := accounts.NewDB(b.appDB)
	if err != nil {
		return err
	}

	err = accountDB.SaveSetting("fleet", fleet)
	if err != nil {
		return err
	}

	err = nodecfg.SaveNodeConfig(b.appDB, conf)
	if err != nil {
		return err
	}

	return nil
}

func getAppDBPath(rootDataDir string, keyUID string) (string, error) {
	if len(rootDataDir) == 0 {
		return "", errors.New("root datadir wasn't provided")
	}

	return filepath.Join(rootDataDir, fmt.Sprintf("%s-v4.db", keyUID)), nil
}

func (b *StatusBackend) getAppDBPath(keyUID string) (string, error) {
	return getAppDBPath(b.rootDataDir, keyUID)
}

func getWalletDBPath(rootDataDir string, keyUID string) (string, error) {
	if len(rootDataDir) == 0 {
		return "", errors.New("root datadir wasn't provided")
	}

	return filepath.Join(rootDataDir, fmt.Sprintf("%s-wallet.db", keyUID)), nil
}

func (b *StatusBackend) getWalletDBPath(keyUID string) (string, error) {
	return getWalletDBPath(b.rootDataDir, keyUID)
}

func (b *StatusBackend) SetSentryDSN(dsn string) {
	b.sentryDSN = dsn
}

func (b *StatusBackend) EnablePanicReporting() error {
	return sentry.Init(
		sentry.WithDSN(b.sentryDSN),
		sentry.WithDefaultContext(),
	)
}

func (b *StatusBackend) DisablePanicReporting() error {
	return sentry.Close()
}

func (b *StatusBackend) TogglePanicReporting(enabled bool) error {
	if enabled {
		return b.EnablePanicReporting()
	}
	return b.DisablePanicReporting()
}

func (b *StatusBackend) SetProfileLogLevel(level string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	err := nodecfg.SetLogLevel(b.appDB, level)
	if err != nil {
		return err
	}
	b.config.LogLevel = level

	return logutils.OverrideRootLoggerWithConfig(b.config.ProfileLogSettings())
}

func (b *StatusBackend) SetLogNamespaces(namespaces string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	err := nodecfg.SetLogNamespaces(b.appDB, namespaces)
	if err != nil {
		return err
	}
	b.config.LogNamespaces = namespaces

	return logutils.OverrideRootLoggerWithConfig(b.config.ProfileLogSettings())
}

func (b *StatusBackend) SetProfileLogEnabled(enabled bool) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	err := nodecfg.SetLogEnabled(b.appDB, enabled)
	if err != nil {
		return err
	}
	b.config.LogEnabled = enabled

	return logutils.OverrideRootLoggerWithConfig(b.config.ProfileLogSettings())
}

func (b *StatusBackend) SetPreLoginLogEnabled(enabled bool) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.preLoginLogConfig.SetEnabled(enabled)
	return logutils.OverrideRootLoggerWithConfig(b.preLoginLogConfig.ConvertToLogSettings())
}

func (b *StatusBackend) SetPreLoginLogLevel(level string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.preLoginLogConfig.SetLevel(level); err != nil {
		return err
	}
	return logutils.OverrideRootLoggerWithConfig(b.preLoginLogConfig.ConvertToLogSettings())
}

func (b *StatusBackend) wakuMetricsHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if b.StatusNode() == nil {
			b.logger.Error("failed to get waku metrics: Services is nil")
			return
		}

		if b.StatusNode().WakuV2ExtService() == nil {
			b.logger.Error("failed to get waku metrics: WakuV2ExtService is nil")
			return
		}

		wakuMetrics := b.StatusNode().WakuV2ExtService().Metrics()
		if wakuMetrics != "" {
			_, err := w.Write([]byte(wakuMetrics))

			if err != nil {
				b.logger.Error("failed to write waku metrics", zap.Error(err))
			}
		}
	})
}
