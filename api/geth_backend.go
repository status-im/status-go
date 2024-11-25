package api

import (
	"context"
	"crypto/ecdsa"
	"database/sql"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/afex/hystrix-go/hystrix"
	"github.com/pkg/errors"

	"github.com/imdario/mergo"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	signercore "github.com/ethereum/go-ethereum/signer/core/apitypes"

	"github.com/status-im/status-go/account"
	"github.com/status-im/status-go/account/generator"
	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/centralizedmetrics"
	centralizedmetricscommon "github.com/status-im/status-go/centralizedmetrics/common"
	"github.com/status-im/status-go/common/dbsetup"
	"github.com/status-im/status-go/connection"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/images"
	"github.com/status-im/status-go/internal/sentry"
	"github.com/status-im/status-go/internal/version"
	"github.com/status-im/status-go/logutils"
	"github.com/status-im/status-go/multiaccounts"
	"github.com/status-im/status-go/multiaccounts/accounts"
	multiacccommon "github.com/status-im/status-go/multiaccounts/common"
	"github.com/status-im/status-go/multiaccounts/settings"
	"github.com/status-im/status-go/node"
	"github.com/status-im/status-go/nodecfg"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/protocol"
	identityutils "github.com/status-im/status-go/protocol/identity"
	"github.com/status-im/status-go/protocol/identity/colorhash"
	"github.com/status-im/status-go/protocol/requests"
	"github.com/status-im/status-go/rpc"
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
	// ErrWhisperClearIdentitiesFailure clearing whisper identities has failed.
	ErrWhisperClearIdentitiesFailure = errors.New("failed to clear whisper identities")
	// ErrWhisperIdentityInjectionFailure injecting whisper identities has failed.
	ErrWhisperIdentityInjectionFailure = errors.New("failed to inject identity into Whisper")
	// ErrWakuIdentityInjectionFailure injecting whisper identities has failed.
	ErrWakuIdentityInjectionFailure = errors.New("failed to inject identity into waku")
	// ErrUnsupportedRPCMethod is for methods not supported by the RPC interface
	ErrUnsupportedRPCMethod = errors.New("method is unsupported by RPC interface")
	// ErrRPCClientUnavailable is returned if an RPC client can't be retrieved.
	// This is a normal situation when a node is stopped.
	ErrRPCClientUnavailable = errors.New("JSON-RPC client is unavailable")
	// ErrDBNotAvailable is returned if a method is called before the DB is available for usage
	ErrDBNotAvailable = errors.New("DB is unavailable")
)

var _ StatusBackend = (*GethStatusBackend)(nil)

// GethStatusBackend implements the Status.im service over go-ethereum
type GethStatusBackend struct {
	mu sync.Mutex
	// rootDataDir is the same for all networks.
	rootDataDir string
	appDB       *sql.DB
	walletDB    *sql.DB
	config      *params.NodeConfig

	statusNode               *node.StatusNode
	personalAPI              *personal.PublicAPI
	multiaccountsDB          *multiaccounts.Database
	account                  *multiaccounts.Account
	accountManager           *account.GethManager
	transactor               *transactions.Transactor
	connectionState          connection.State
	appState                 AppState
	selectedAccountKeyID     string
	allowAllRPC              bool // used only for tests, disables api method restrictions
	LocalPairingStateManager *statecontrol.ProcessStateManager
	centralizedMetrics       *centralizedmetrics.MetricService
	sentryDSN                string

	logger *zap.Logger
}

// NewGethStatusBackend create a new GethStatusBackend instance
func NewGethStatusBackend(logger *zap.Logger) *GethStatusBackend {
	logger = logger.Named("GethStatusBackend")
	backend := &GethStatusBackend{
		logger: logger,
	}
	backend.initialize()

	logger.Info("Status backend initialized",
		zap.String("backend geth version", version.Version()),
		zap.String("commit", version.GitCommit()),
		zap.String("IpfsGatewayURL", params.IpfsGatewayURL))

	return backend
}

func (b *GethStatusBackend) initialize() {
	accountManager := account.NewGethManager(b.logger)
	transactor := transactions.NewTransactor()
	personalAPI := personal.NewAPI()
	statusNode := node.New(transactor, b.logger)

	b.statusNode = statusNode
	b.accountManager = accountManager
	b.transactor = transactor
	b.personalAPI = personalAPI
	b.statusNode.SetMultiaccountsDB(b.multiaccountsDB)
	b.LocalPairingStateManager = new(statecontrol.ProcessStateManager)
	b.LocalPairingStateManager.SetPairing(false)
}

// StatusNode returns reference to node manager
func (b *GethStatusBackend) StatusNode() *node.StatusNode {
	return b.statusNode
}

// AccountManager returns reference to account manager
func (b *GethStatusBackend) AccountManager() *account.GethManager {
	return b.accountManager
}

// Transactor returns reference to a status transactor
func (b *GethStatusBackend) Transactor() *transactions.Transactor {
	return b.transactor
}

// SelectedAccountKeyID returns a Whisper key ID of the selected chat key pair.
func (b *GethStatusBackend) SelectedAccountKeyID() string {
	return b.selectedAccountKeyID
}

// IsNodeRunning confirm that node is running
func (b *GethStatusBackend) IsNodeRunning() bool {
	return b.statusNode.IsRunning()
}

// StartNode start Status node, fails if node is already started
func (b *GethStatusBackend) StartNode(config *params.NodeConfig) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.startNode(config); err != nil {
		signal.SendNodeCrashed(err)
		return err
	}
	return nil
}

func (b *GethStatusBackend) UpdateRootDataDir(datadir string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.rootDataDir = datadir
}

func (b *GethStatusBackend) GetMultiaccountDB() *multiaccounts.Database {
	return b.multiaccountsDB
}

func (b *GethStatusBackend) OpenAccounts() error {
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

	b.centralizedMetrics = centralizedmetrics.NewDefaultMetricService(b.multiaccountsDB.DB(), b.logger)
	err = b.centralizedMetrics.EnsureStarted()
	if err != nil {
		return err
	}

	// Probably we should iron out a bit better how to create/dispose of the status-service
	b.statusNode.SetMultiaccountsDB(db)

	err = b.statusNode.StartMediaServerWithoutDB()
	if err != nil {
		b.logger.Error("failed to start media server without app db", zap.Error(err))
		return err
	}

	return nil
}

func (b *GethStatusBackend) CentralizedMetricsInfo() (*centralizedmetrics.MetricsInfo, error) {
	if b.centralizedMetrics == nil {
		return nil, errors.New("centralized metrics not initialized")
	}

	return b.centralizedMetrics.Info()
}

func (b *GethStatusBackend) ToggleCentralizedMetrics(isEnabled bool) error {
	if b.centralizedMetrics == nil {
		return errors.New("centralized metrics nil")
	}

	return b.centralizedMetrics.ToggleEnabled(isEnabled)
}

func (b *GethStatusBackend) AddCentralizedMetric(metric centralizedmetricscommon.Metric) error {
	if b.centralizedMetrics == nil {
		return errors.New("centralized metrics nil")
	}
	return b.centralizedMetrics.AddMetric(metric)

}

func (b *GethStatusBackend) GetAccounts() ([]multiaccounts.Account, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.multiaccountsDB == nil {
		return nil, errors.New("accounts db wasn't initialized")
	}
	return b.multiaccountsDB.GetAccounts()
}

func (b *GethStatusBackend) AcceptTerms() error {
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

func (b *GethStatusBackend) getAccountByKeyUID(keyUID string) (*multiaccounts.Account, error) {
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
			return &acc, nil
		}
	}
	return nil, fmt.Errorf("account with keyUID %s not found", keyUID)
}

func (b *GethStatusBackend) SaveAccount(account multiaccounts.Account) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.multiaccountsDB == nil {
		return errors.New("accounts db wasn't initialized")
	}
	return b.multiaccountsDB.SaveAccount(account)
}

func (b *GethStatusBackend) DeleteMultiaccount(keyUID string, keyStoreDir string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.multiaccountsDB == nil {
		return errors.New("accounts db wasn't initialized")
	}

	err := b.multiaccountsDB.DeleteAccount(keyUID)
	if err != nil {
		return err
	}

	appDbPath, err := b.getAppDBPath(keyUID)
	if err != nil {
		return err
	}

	walletDbPath, err := b.getWalletDBPath(keyUID)
	if err != nil {
		return err
	}

	dbFiles := []string{
		filepath.Join(b.rootDataDir, fmt.Sprintf("app-%x.sql", keyUID)),
		filepath.Join(b.rootDataDir, fmt.Sprintf("app-%x.sql-shm", keyUID)),
		filepath.Join(b.rootDataDir, fmt.Sprintf("app-%x.sql-wal", keyUID)),
		filepath.Join(b.rootDataDir, fmt.Sprintf("%s.db", keyUID)),
		filepath.Join(b.rootDataDir, fmt.Sprintf("%s.db-shm", keyUID)),
		filepath.Join(b.rootDataDir, fmt.Sprintf("%s.db-wal", keyUID)),
		appDbPath,
		appDbPath + "-shm",
		appDbPath + "-wal",
		walletDbPath,
		walletDbPath + "-shm",
		walletDbPath + "-wal",
	}
	for _, path := range dbFiles {
		if _, err := os.Stat(path); err == nil {
			err = os.Remove(path)
			if err != nil {
				return err
			}
		}
	}

	if b.account != nil && b.account.KeyUID == keyUID {
		// reset active account
		b.account = nil
	}

	return os.RemoveAll(keyStoreDir)
}

func (b *GethStatusBackend) DeleteImportedKey(address, password, keyStoreDir string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	err := filepath.Walk(keyStoreDir, func(path string, fileInfo os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if strings.Contains(fileInfo.Name(), address) {
			_, err := b.accountManager.VerifyAccountPassword(keyStoreDir, "0x"+address, password)
			if err != nil {
				b.logger.Error("failed to verify account", zap.String("account", address), zap.Error(err))
				return err
			}

			return os.Remove(path)
		}
		return nil
	})

	return err
}

func (b *GethStatusBackend) runDBFileMigrations(account multiaccounts.Account, password string) (string, error) {
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

func (b *GethStatusBackend) ensureDBsOpened(account multiaccounts.Account, password string) (err error) {
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

func (b *GethStatusBackend) ensureAppDBOpened(account multiaccounts.Account, password string) (err error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.appDB != nil {
		return nil
	}
	if len(b.rootDataDir) == 0 {
		return errors.New("root datadir wasn't provided")
	}

	dbFilePath, err := b.runDBFileMigrations(account, password)
	if err != nil {
		return errors.New("Failed to migrate db file: " + err.Error())
	}

	appdatabase.CurrentAppDBKeyUID = account.KeyUID
	b.appDB, err = appdatabase.InitializeDB(dbFilePath, password, account.KDFIterations)
	if err != nil {
		b.logger.Error("failed to initialize db", zap.Error(err))
		return err
	}
	b.statusNode.SetAppDB(b.appDB)
	return nil
}

func fileExists(path string) bool {
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return false
	}

	return true
}

func (b *GethStatusBackend) walletDBExists(keyUID string) bool {
	path, err := b.getWalletDBPath(keyUID)
	if err != nil {
		return false
	}

	return fileExists(path)
}

func (b *GethStatusBackend) appDBExists(keyUID string) bool {
	path, err := b.getAppDBPath(keyUID)
	if err != nil {
		return false
	}

	return fileExists(path)
}

func (b *GethStatusBackend) ensureWalletDBOpened(account multiaccounts.Account, password string) (err error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.walletDB != nil {
		return nil
	}

	dbWalletPath, err := b.getWalletDBPath(account.KeyUID)
	if err != nil {
		return err
	}

	b.walletDB, err = walletdatabase.InitializeDB(dbWalletPath, password, account.KDFIterations)
	if err != nil {
		b.logger.Error("failed to initialize wallet db", zap.Error(err))
		return err
	}
	b.statusNode.SetWalletDB(b.walletDB)
	return nil
}

func (b *GethStatusBackend) setupLogSettings() error {
	logSettings := logutils.LogSettings{
		Enabled:         b.config.LogEnabled,
		Level:           b.config.LogLevel,
		File:            b.config.LogFile,
		MaxSize:         b.config.LogMaxSize,
		MaxBackups:      b.config.LogMaxBackups,
		CompressRotated: b.config.LogCompressRotated,
	}
	if err := logutils.OverrideRootLoggerWithConfig(logSettings); err != nil {
		return err
	}
	return nil
}

// Deprecated: Use StartNodeWithAccount instead.
func (b *GethStatusBackend) StartNodeWithKey(acc multiaccounts.Account, password string, keyHex string, nodecfg *params.NodeConfig) error {
	if acc.KDFIterations == 0 {
		kdfIterations, err := b.multiaccountsDB.GetAccountKDFIterationsNumber(acc.KeyUID)
		if err != nil {
			return err
		}

		acc.KDFIterations = kdfIterations
	}

	chatKey, err := ethcrypto.HexToECDSA(keyHex)
	if err != nil {
		return err
	}

	err = b.startNodeWithAccount(acc, password, nodecfg, chatKey)
	if err != nil {
		// Stop node for clean up
		_ = b.StopNode()
	}
	// get logged in
	if b.LocalPairingStateManager.IsPairing() {
		return nil
	}
	return b.LoggedIn(acc.KeyUID, err)
}

func (b *GethStatusBackend) OverwriteNodeConfigValues(conf *params.NodeConfig, n *params.NodeConfig) (*params.NodeConfig, error) {
	if err := mergo.Merge(conf, n, mergo.WithOverride); err != nil {
		return nil, err
	}

	conf.Networks = n.Networks

	if err := b.saveNodeConfig(conf); err != nil {
		return nil, err
	}

	return conf, nil
}

func (b *GethStatusBackend) updateAccountColorHashAndColorID(keyUID string, accountsDB *accounts.Database) (*multiaccounts.Account, error) {
	multiAccount, err := b.getAccountByKeyUID(keyUID)
	if err != nil {
		return nil, err
	}
	if multiAccount.ColorHash == nil {
		keypair, err := accountsDB.GetKeypairByKeyUID(keyUID)
		if err != nil {
			return nil, err
		}
		publicKey := keypair.GetChatPublicKey()
		if publicKey == nil {
			return nil, errors.New("chat public key not found")
		}
		if err = enrichMultiAccountByPublicKey(multiAccount, publicKey); err != nil {
			return nil, err
		}
		if err = b.multiaccountsDB.UpdateAccount(*multiAccount); err != nil {
			return nil, err
		}
	}
	return multiAccount, nil
}

func (b *GethStatusBackend) overrideNetworks(conf *params.NodeConfig, request *requests.Login) {
	conf.Networks = setRPCs(defaultNetworks(request.WalletSecretsConfig.StatusProxyStageName), &request.WalletSecretsConfig)
}

func (b *GethStatusBackend) LoginAccount(request *requests.Login) error {
	err := b.loginAccount(request)
	if err != nil {
		// Stop node for clean up
		_ = b.StopNode()
	}
	if b.LocalPairingStateManager.IsPairing() {
		return nil
	}
	return b.LoggedIn(request.KeyUID, err)
}

func (b *GethStatusBackend) loginAccount(request *requests.Login) error {
	if err := request.Validate(); err != nil {
		return err
	}

	if request.Mnemonic != "" {
		info, err := b.generateAccountInfo(request.Mnemonic)
		if err != nil {
			return errors.Wrap(err, "failed to generate account info")
		}

		derivedAddresses, err := b.getDerivedAddresses(info.ID)
		if err != nil {
			return errors.Wrap(err, "failed to get derived addresses")
		}

		request.Password = derivedAddresses[pathEncryption].PublicKey
		request.KeycardWhisperPrivateKey = derivedAddresses[pathDefaultChat].PrivateKey
	}

	acc := multiaccounts.Account{
		KeyUID:        request.KeyUID,
		KDFIterations: request.KdfIterations,
	}

	if acc.KDFIterations == 0 {
		acc.KDFIterations = dbsetup.ReducedKDFIterationsNumber
	}

	err := b.ensureDBsOpened(acc, request.Password)
	if err != nil {
		return errors.Wrap(err, "failed to open database")
	}

	defaultCfg := &params.NodeConfig{
		// why we need this? relate PR: https://github.com/status-im/status-go/pull/4014
		KeycardPairingDataFile: DefaultKeycardPairingDataFile,
	}

	defaultCfg.WalletConfig = buildWalletConfig(&request.WalletSecretsConfig, request.StatusProxyEnabled)

	err = b.UpdateNodeConfigFleet(acc, request.Password, defaultCfg)
	if err != nil {
		return errors.Wrap(err, "failed to update node config fleet")
	}

	err = b.loadNodeConfig(defaultCfg)
	if err != nil {
		return errors.Wrap(err, "failed to load node config")
	}

	if request.RuntimeLogLevel != "" {
		b.config.LogLevel = request.RuntimeLogLevel
	}

	if b.config.WakuV2Config.Enabled && request.WakuV2Nameserver != "" {
		b.config.WakuV2Config.Nameserver = request.WakuV2Nameserver
	}

	b.config.ShhextConfig.BandwidthStatsEnabled = request.BandwidthStatsEnabled

	b.overrideNetworks(b.config, request)

	if request.APIConfig != nil {
		overrideApiConfig(b.config, request.APIConfig)
	}

	err = b.setupLogSettings()
	if err != nil {
		return errors.Wrap(err, "failed to setup log settings")
	}

	accountsDB, err := accounts.NewDB(b.appDB)
	if err != nil {
		return errors.Wrap(err, "failed to create accounts db")
	}

	multiAccount, err := b.updateAccountColorHashAndColorID(acc.KeyUID, accountsDB)
	if err != nil {
		return errors.Wrap(err, "failed to update account color hash and color id")
	}
	b.account = multiAccount

	chatAddr, err := accountsDB.GetChatAddress()
	if err != nil {
		return errors.Wrap(err, "failed to get chat address")
	}
	walletAddr, err := accountsDB.GetWalletAddress()
	if err != nil {
		return errors.Wrap(err, "failed to get wallet address")
	}
	watchAddrs, err := accountsDB.GetWalletAddresses()
	if err != nil {
		return errors.Wrap(err, "failed to get wallet addresses")
	}
	login := account.LoginParams{
		Password:       request.Password,
		ChatAddress:    chatAddr,
		WatchAddresses: watchAddrs,
		MainAccount:    walletAddr,
	}

	err = b.StartNode(b.config)
	if err != nil {
		b.logger.Info("failed to start node")
		return errors.Wrap(err, "failed to start node")
	}

	if chatKey := request.ChatPrivateKey(); chatKey == nil {
		err = b.SelectAccount(login)
		if err != nil {
			return errors.Wrap(err, "failed to select account")
		}
	} else {
		// In case of keycard, we don't have a keystore, instead we have private key loaded from the keycard
		if err := b.accountManager.SetChatAccount(chatKey); err != nil {
			return errors.Wrap(err, "failed to set chat account")
		}
		_, err = b.accountManager.SelectedChatAccount()
		if err != nil {
			return errors.Wrap(err, "failed to get selected chat account")
		}

		b.accountManager.SetAccountAddresses(walletAddr, watchAddrs...)
		err = b.injectAccountsIntoServices()
		if err != nil {
			return errors.Wrap(err, "failed to inject accounts into services")
		}
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
func (b *GethStatusBackend) UpdateNodeConfigFleet(acc multiaccounts.Account, password string, config *params.NodeConfig) error {
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
func (b *GethStatusBackend) startNodeWithAccount(acc multiaccounts.Account, password string, inputNodeCfg *params.NodeConfig, chatKey *ecdsa.PrivateKey) error {
	err := b.ensureDBsOpened(acc, password)
	if err != nil {
		return err
	}

	err = b.loadNodeConfig(inputNodeCfg)
	if err != nil {
		return err
	}

	err = b.setupLogSettings()
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

	chatAddr, err := accountsDB.GetChatAddress()
	if err != nil {
		return err
	}
	walletAddr, err := accountsDB.GetWalletAddress()
	if err != nil {
		return err
	}
	watchAddrs, err := accountsDB.GetWalletAddresses()
	if err != nil {
		return err
	}
	login := account.LoginParams{
		Password:       password,
		ChatAddress:    chatAddr,
		WatchAddresses: watchAddrs,
		MainAccount:    walletAddr,
	}

	err = b.StartNode(b.config)
	if err != nil {
		b.logger.Info("failed to start node")
		return err
	}

	if chatKey == nil {
		// Load account from keystore
		err = b.SelectAccount(login)
		if err != nil {
			return err
		}
	} else {
		// In case of keycard, we don't have keystore, but we directly have the private key
		if err := b.accountManager.SetChatAccount(chatKey); err != nil {
			return err
		}
		_, err = b.accountManager.SelectedChatAccount()
		if err != nil {
			return err
		}

		b.accountManager.SetAccountAddresses(walletAddr, watchAddrs...)
		err = b.injectAccountsIntoServices()
		if err != nil {
			return err
		}
	}

	err = b.multiaccountsDB.UpdateAccountTimestamp(acc.KeyUID, time.Now().Unix())
	if err != nil {
		b.logger.Info("failed to update account")
		return err
	}

	return nil
}

func (b *GethStatusBackend) accountsDB() (*accounts.Database, error) {
	return accounts.NewDB(b.appDB)
}

func (b *GethStatusBackend) GetSettings() (*settings.Settings, error) {
	accountsDB, err := b.accountsDB()
	if err != nil {
		return nil, err
	}

	settings, err := accountsDB.GetSettings()
	if err != nil {
		return nil, err
	}

	return &settings, nil
}

func (b *GethStatusBackend) GetEnsUsernames() ([]*ens.UsernameDetail, error) {
	db := ens.NewEnsDatabase(b.appDB)
	removed := false
	return db.GetEnsUsernames(&removed)
}

func (b *GethStatusBackend) MigrateKeyStoreDir(acc multiaccounts.Account, password, oldDir, newDir string) error {
	err := b.ensureDBsOpened(acc, password)
	if err != nil {
		return err
	}

	accountDB, err := accounts.NewDB(b.appDB)
	if err != nil {
		return err
	}
	accounts, err := accountDB.GetActiveAccounts()
	if err != nil {
		return err
	}
	settings, err := accountDB.GetSettings()
	if err != nil {
		return err
	}
	addresses := []string{settings.EIP1581Address.Hex(), settings.WalletRootAddress.Hex()}
	for _, account := range accounts {
		addresses = append(addresses, account.Address.Hex())
	}
	err = b.accountManager.MigrateKeyStoreDir(oldDir, newDir, addresses)
	if err != nil {
		return err
	}

	return nil
}

func (b *GethStatusBackend) Login(keyUID, password string) error {
	return b.startNodeWithAccount(multiaccounts.Account{KeyUID: keyUID}, password, nil, nil)
}

func (b *GethStatusBackend) StartNodeWithAccount(acc multiaccounts.Account, password string, nodecfg *params.NodeConfig, chatKey *ecdsa.PrivateKey) error {
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

func (b *GethStatusBackend) LoggedIn(keyUID string, err error) error {
	if err != nil {
		signal.SendLoggedIn(nil, nil, nil, err)
		return err
	}
	settings, err := b.GetSettings()
	if err != nil {
		return err
	}
	account, err := b.getAccountByKeyUID(keyUID)
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
	signal.SendLoggedIn(account, settings, ensUsernamesJSON, nil)
	return nil
}

func (b *GethStatusBackend) ExportUnencryptedDatabase(acc multiaccounts.Account, password, directory string) error {
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

func (b *GethStatusBackend) ImportUnencryptedDatabase(acc multiaccounts.Account, password, databasePath string) error {
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

func (b *GethStatusBackend) reEncryptKeyStoreDir(currentPassword string, newPassword string) error {
	config := b.StatusNode().Config()
	keyDir := ""
	if config == nil {
		keyDir = b.accountManager.Keydir
	} else {
		keyDir = config.KeyStoreDir
	}

	if keyDir != "" {
		err := b.accountManager.ReEncryptKeyStoreDir(keyDir, currentPassword, newPassword)
		if err != nil {
			return fmt.Errorf("ReEncryptKeyStoreDir error: %v", err)
		}
	}
	return nil
}

func (b *GethStatusBackend) ChangeDatabasePassword(keyUID string, password string, newPassword string) error {
	account, err := b.multiaccountsDB.GetAccount(keyUID)
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

	isCurrentAccount := appDBPath == internalDbPath

	restartNode := func() {
		if isCurrentAccount {
			if err != nil {
				// TODO https://github.com/status-im/status-go/issues/3906
				// Fix restarting node, as it always fails but the error is ignored
				// because UI calls Logout and Quit afterwards. It should not be UI-dependent
				// and should be handled gracefully here if it makes sense to run dummy node after
				// logout
				_ = b.startNodeWithAccount(*account, password, nil, nil)
			} else {
				_ = b.startNodeWithAccount(*account, newPassword, nil, nil)
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
	err = b.changeAppDBPassword(account, logout, password, newPassword)
	if err != nil {
		return err
	}

	// Already logged out but pass a param to decouple the logic for testing
	err = b.changeWalletDBPassword(account, noLogout, password, newPassword)
	if err != nil {
		// Revert the password to original
		err2 := b.changeAppDBPassword(account, noLogout, newPassword, password)
		if err2 != nil {
			b.logger.Error("failed to revert app db password", zap.Error(err2))
		}

		return err
	}

	return nil
}

func (b *GethStatusBackend) changeAppDBPassword(account *multiaccounts.Account, logout func(), password string, newPassword string) error {
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

func (b *GethStatusBackend) changeWalletDBPassword(account *multiaccounts.Account, logout func(), password string, newPassword string) error {
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

func (b *GethStatusBackend) createTempDBFile(pattern string) (tmpDbPath string, cleanup func(), err error) {
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

func (b *GethStatusBackend) ConvertToKeycardAccount(account multiaccounts.Account, s settings.Settings, keycardUID string, oldPassword string, newPassword string) error {
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
		if err == accounts.ErrDbKeypairNotFound {
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

	// This check is added due to mobile app cause it doesn't support a Keycard features as desktop app.
	// We should remove the following line once mobile and desktop app align.
	if len(keycardUID) > 0 {
		displayName, err := accountDB.DisplayName()
		if err != nil {
			return err
		}

		position, err := accountDB.GetPositionForNextNewKeycard()
		if err != nil {
			return err
		}

		kc := accounts.Keycard{
			KeycardUID:    keycardUID,
			KeycardName:   displayName,
			KeycardLocked: false,
			KeyUID:        account.KeyUID,
			Position:      position,
		}

		for _, acc := range keypair.Accounts {
			kc.AccountsAddresses = append(kc.AccountsAddresses, acc.Address)
		}
		err = messenger.SaveOrUpdateKeycard(context.Background(), &kc)
		if err != nil {
			return err
		}
	}

	masterAddress, err := accountDB.GetMasterAddress()
	if err != nil {
		return err
	}

	eip1581Address, err := accountDB.GetEIP1581Address()
	if err != nil {
		return err
	}

	walletRootAddress, err := accountDB.GetWalletRootAddress()
	if err != nil {
		return err
	}

	err = b.closeDBs()
	if err != nil {
		return err
	}

	err = b.ChangeDatabasePassword(account.KeyUID, oldPassword, newPassword)
	if err != nil {
		return err
	}

	// We need to delete all accounts for the Keycard which is being added
	for _, acc := range keypair.Accounts {
		err = b.accountManager.DeleteAccount(acc.Address)
		if err != nil {
			return err
		}
	}

	err = b.accountManager.DeleteAccount(masterAddress)
	if err != nil {
		return err
	}

	err = b.accountManager.DeleteAccount(eip1581Address)
	if err != nil {
		return err
	}

	err = b.accountManager.DeleteAccount(walletRootAddress)
	if err != nil {
		return err
	}

	return nil
}

func (b *GethStatusBackend) RestoreAccountAndLogin(request *requests.RestoreAccount) (*multiaccounts.Account, error) {

	if err := request.Validate(); err != nil {
		return nil, err
	}

	response, err := b.generateOrImportAccount(request.Mnemonic, 0, request.FetchBackup, &request.CreateAccount)
	if err != nil {
		return nil, err
	}

	err = b.StartNodeWithAccountAndInitialConfig(
		*response.account,
		request.Password,
		*response.settings,
		response.nodeConfig,
		response.subAccounts,
		response.chatPrivateKey,
	)

	if err != nil {
		b.logger.Error("start node", zap.Error(err))
		return nil, err
	}

	return response.account, nil
}

func (b *GethStatusBackend) RestoreKeycardAccountAndLogin(request *requests.RestoreAccount) (*multiaccounts.Account, error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}

	keyStoreDir, err := b.InitKeyStoreDirWithAccount(request.RootDataDir, request.Keycard.KeyUID)
	if err != nil {
		return nil, err
	}

	derivedAddresses := map[string]generator.AccountInfo{
		pathDefaultChat: {
			Address:    request.Keycard.WhisperAddress,
			PublicKey:  request.Keycard.WhisperPublicKey,
			PrivateKey: request.Keycard.WhisperPrivateKey,
		},
		pathWalletRoot: {
			Address: request.Keycard.WalletRootAddress,
		},
		pathDefaultWallet: {
			Address:   request.Keycard.WalletAddress,
			PublicKey: request.Keycard.WalletPublicKey,
		},
		pathEIP1581: {
			Address: request.Keycard.Eip1581Address,
		},
		pathEncryption: {
			PublicKey: request.Keycard.EncryptionPublicKey,
		},
	}

	input := &prepareAccountInput{
		customizationColorClock: 0,
		accountID:               "", // empty for keycard
		keyUID:                  request.Keycard.KeyUID,
		address:                 request.Keycard.Address,
		mnemonic:                "",
		restoringAccount:        true,
		derivedAddresses:        derivedAddresses,
		fetchBackup:             request.FetchBackup, // WARNING: Ensure this value is correct
		keyStoreDir:             keyStoreDir,
	}

	response, err := b.prepareNodeAccount(&request.CreateAccount, input)
	if err != nil {
		return nil, err
	}

	err = b.StartNodeWithAccountAndInitialConfig(
		*response.account,
		request.Password,
		*response.settings,
		response.nodeConfig,
		response.subAccounts,
		response.chatPrivateKey, //request.WhisperPrivateKey,
	)

	if err != nil {
		b.logger.Error("start node", zap.Error(err))
		return nil, errors.Wrap(err, "failed to start node")
	}

	return response.account, nil
}

func (b *GethStatusBackend) GetKeyUIDByMnemonic(mnemonic string) (string, error) {
	accountGenerator := b.accountManager.AccountsGenerator()

	info, err := accountGenerator.ImportMnemonic(mnemonic, "")
	if err != nil {
		return "", err
	}

	return info.KeyUID, nil
}

type prepareAccountInput struct {
	customizationColorClock uint64
	accountID               string
	keyUID                  string
	address                 string
	mnemonic                string
	restoringAccount        bool
	derivedAddresses        map[string]generator.AccountInfo
	fetchBackup             bool
	keyStoreDir             string
	opts                    []params.Option
}

type accountBundle struct {
	account        *multiaccounts.Account
	settings       *settings.Settings
	nodeConfig     *params.NodeConfig
	subAccounts    []*accounts.Account
	chatPrivateKey *ecdsa.PrivateKey
}

func (b *GethStatusBackend) generateOrImportAccount(mnemonic string, customizationColorClock uint64, fetchBackup bool, request *requests.CreateAccount, opts ...params.Option) (*accountBundle, error) {
	info, err := b.generateAccountInfo(mnemonic)
	if err != nil {
		return nil, err
	}

	keyStoreDir, err := b.InitKeyStoreDirWithAccount(request.RootDataDir, info.KeyUID)
	if err != nil {
		return nil, err
	}

	derivedAddresses, err := b.getDerivedAddresses(info.ID)
	if err != nil {
		return nil, err
	}

	input := &prepareAccountInput{
		customizationColorClock: customizationColorClock,
		accountID:               info.ID,
		keyUID:                  info.KeyUID,
		address:                 info.Address,
		mnemonic:                info.Mnemonic,
		restoringAccount:        mnemonic != "",
		derivedAddresses:        derivedAddresses,
		fetchBackup:             fetchBackup,
		keyStoreDir:             keyStoreDir,
		opts:                    opts,
	}

	return b.prepareNodeAccount(request, input)
}

func (b *GethStatusBackend) prepareNodeAccount(request *requests.CreateAccount, input *prepareAccountInput) (*accountBundle, error) {
	var err error
	response := &accountBundle{}

	if request.KeycardInstanceUID != "" {
		request.Password = input.derivedAddresses[pathEncryption].PublicKey
	}

	// NOTE: I intentionally left this condition separately and not an `else` branch. Technically it's an `else`,
	// 		 but the statements inside are not the opposite statement of the first statement. It's just kinda like this:
	// 		 - replace password when we're using keycard
	// 		 - store account when we're not using keycard
	if request.KeycardInstanceUID == "" {
		err = b.storeAccount(input.accountID, request.Password, paths)
		if err != nil {
			return nil, err
		}
	}

	response.account, err = b.buildAccount(request, input)
	if err != nil {
		return nil, errors.Wrap(err, "failed to build account")
	}

	response.settings, err = b.prepareSettings(request, input)
	if err != nil {
		return nil, errors.Wrap(err, "failed to prepare settings")
	}

	if response.account.Name == "" {
		response.account.Name = response.settings.Name
	}

	response.nodeConfig, err = b.prepareConfig(request, input, response.settings.InstallationID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to prepare node config")
	}

	response.subAccounts, err = b.prepareSubAccounts(request, input)
	if err != nil {
		return nil, errors.Wrap(err, "failed to prepare sub accounts")
	}

	response, err = b.prepareForKeycard(request, input, response)
	if err != nil {
		return nil, errors.Wrap(err, "failed to prepare for keycard")
	}

	return response, nil
}

func (b *GethStatusBackend) InitKeyStoreDirWithAccount(rootDataDir, keyUID string) (string, error) {
	b.UpdateRootDataDir(rootDataDir)
	keyStoreRelativePath, keystoreAbsolutePath := DefaultKeystorePath(rootDataDir, keyUID)
	// Initialize keystore dir with account
	return keyStoreRelativePath, b.accountManager.InitKeystore(keystoreAbsolutePath)
}

func (b *GethStatusBackend) generateAccountInfo(mnemonic string) (*generator.GeneratedAccountInfo, error) {
	accountGenerator := b.accountManager.AccountsGenerator()

	var info generator.GeneratedAccountInfo
	var err error
	if mnemonic == "" {
		// generate 1(n) account with default mnemonic length and no passphrase
		generatedAccountInfos, err := accountGenerator.Generate(defaultMnemonicLength, 1, "")
		info = generatedAccountInfos[0]

		if err != nil {
			return nil, err
		}
	} else {

		info, err = accountGenerator.ImportMnemonic(mnemonic, "")
		if err != nil {
			return nil, err
		}
	}

	return &info, nil
}

func (b *GethStatusBackend) storeAccount(id string, password string, paths []string) error {
	accountGenerator := b.accountManager.AccountsGenerator()

	_, err := accountGenerator.StoreAccount(id, password)
	if err != nil {
		return err
	}

	_, err = accountGenerator.StoreDerivedAccounts(id, password, paths)
	if err != nil {
		return err
	}

	return nil
}

func (b *GethStatusBackend) buildAccount(request *requests.CreateAccount, input *prepareAccountInput) (*multiaccounts.Account, error) {
	err := b.OpenAccounts()
	if err != nil {
		return nil, err
	}

	acc := &multiaccounts.Account{
		KeyUID:                  input.keyUID,
		Name:                    request.DisplayName,
		CustomizationColor:      multiacccommon.CustomizationColor(request.CustomizationColor),
		CustomizationColorClock: input.customizationColorClock,
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
			imageCropRectangle = &requests.ImageCropRectangle{
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

	return acc, nil
}

func (b *GethStatusBackend) prepareSettings(request *requests.CreateAccount, input *prepareAccountInput) (*settings.Settings, error) {
	settings, err := defaultSettings(input.keyUID, input.address, input.derivedAddresses)
	if err != nil {
		return nil, err
	}

	settings.DeviceName = request.DeviceName
	settings.DisplayName = request.DisplayName
	settings.PreviewPrivacy = request.PreviewPrivacy
	settings.CurrentNetwork = request.CurrentNetwork
	settings.TestNetworksEnabled = request.TestNetworksEnabled
	if !input.restoringAccount {
		settings.Mnemonic = &input.mnemonic
		// TODO(rasom): uncomment it as soon as address will be properly
		// marked as shown on mobile client
		//settings.MnemonicWasNotShown = true
	}

	if request.WakuV2Fleet != "" {
		settings.Fleet = &request.WakuV2Fleet
	}

	return settings, nil
}

func (b *GethStatusBackend) prepareConfig(request *requests.CreateAccount, input *prepareAccountInput, installationID string) (*params.NodeConfig, error) {
	nodeConfig, err := DefaultNodeConfig(installationID, request, input.opts...)
	if err != nil {
		return nil, err
	}
	nodeConfig.ProcessBackedupMessages = input.fetchBackup

	// when we set nodeConfig.KeyStoreDir, value of nodeConfig.KeyStoreDir should not contain the rootDataDir
	// loadNodeConfig will add rootDataDir to nodeConfig.KeyStoreDir
	nodeConfig.KeyStoreDir = input.keyStoreDir

	return nodeConfig, nil
}

func (b *GethStatusBackend) prepareSubAccounts(request *requests.CreateAccount, input *prepareAccountInput) ([]*accounts.Account, error) {
	emoji, err := randomWalletEmoji()
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate random emoji")
	}

	walletDerivedAccount := input.derivedAddresses[pathDefaultWallet]
	walletAccount := &accounts.Account{
		PublicKey:          types.Hex2Bytes(walletDerivedAccount.PublicKey),
		KeyUID:             input.keyUID,
		Address:            types.HexToAddress(walletDerivedAccount.Address),
		ColorID:            multiacccommon.CustomizationColor(request.CustomizationColor),
		Emoji:              emoji,
		Wallet:             true,
		Path:               pathDefaultWallet,
		Name:               walletAccountDefaultName,
		AddressWasNotShown: !input.restoringAccount,
	}

	chatDerivedAccount := input.derivedAddresses[pathDefaultChat]
	chatAccount := &accounts.Account{
		PublicKey: types.Hex2Bytes(chatDerivedAccount.PublicKey),
		KeyUID:    input.keyUID,
		Address:   types.HexToAddress(chatDerivedAccount.Address),
		Name:      request.DisplayName,
		Chat:      true,
		Path:      pathDefaultChat,
	}

	return []*accounts.Account{walletAccount, chatAccount}, nil
}

func (b *GethStatusBackend) prepareForKeycard(request *requests.CreateAccount, input *prepareAccountInput, response *accountBundle) (*accountBundle, error) {
	if request.KeycardInstanceUID == "" {
		return response, nil
	}

	if request.KeycardPairingKey != "" {
		// KeycardPairingKey is used only on mobile
		response.settings.KeycardPairing = request.KeycardPairingKey
		response.account.KeycardPairing = request.KeycardPairingKey
	} else {
		// KeycardPairingDataFile is used only on desktop
		kp := wallet.NewKeycardPairings()
		kp.SetKeycardPairingsFile(response.nodeConfig.KeycardPairingDataFile)
		pairings, err := kp.GetPairings()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get keycard pairings")
		}

		keycard, ok := pairings[request.KeycardInstanceUID]
		if !ok {
			return nil, errors.New("keycard not found in pairings file")
		}

		response.settings.KeycardPairing = keycard.Key
		response.account.KeycardPairing = keycard.Key
	}

	response.settings.KeycardInstanceUID = request.KeycardInstanceUID
	response.settings.KeycardPairedOn = time.Now().Unix()

	privateKeyHex := strings.TrimPrefix(input.derivedAddresses[pathDefaultChat].PrivateKey, "0x")
	var err error
	response.chatPrivateKey, err = crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse chat private key hex")
	}

	return response, nil
}

func (b *GethStatusBackend) getDerivedAddresses(id string) (map[string]generator.AccountInfo, error) {
	accountGenerator := b.accountManager.AccountsGenerator()
	return accountGenerator.DeriveAddresses(id, paths)
}

// CreateAccountAndLogin creates a new account and logs in with it.
// NOTE: requests.CreateAccount is used for public, params.Option maybe used for internal usage.
func (b *GethStatusBackend) CreateAccountAndLogin(request *requests.CreateAccount, opts ...params.Option) (*multiaccounts.Account, error) {
	validation := &requests.CreateAccountValidation{
		AllowEmptyDisplayName: false,
	}
	if err := request.Validate(validation); err != nil {
		return nil, err
	}

	response, err := b.generateOrImportAccount("", 1, false, request, opts...)
	if err != nil {
		return nil, err
	}

	err = b.StartNodeWithAccountAndInitialConfig(
		*response.account,
		request.Password,
		*response.settings,
		response.nodeConfig,
		response.subAccounts,
		response.chatPrivateKey,
	)

	if err != nil {
		b.logger.Error("start node", zap.Error(err))
		return nil, err
	}

	return response.account, nil
}

func (b *GethStatusBackend) ConvertToRegularAccount(mnemonic string, currPassword string, newPassword string) error {
	messenger := b.Messenger()
	if messenger == nil {
		return errors.New("cannot resolve messenger instance")
	}

	mnemonicNoExtraSpaces := strings.Join(strings.Fields(mnemonic), " ")
	accountInfo, err := b.accountManager.AccountsGenerator().ImportMnemonic(mnemonicNoExtraSpaces, "")
	if err != nil {
		return err
	}

	kdfIterations, err := b.multiaccountsDB.GetAccountKDFIterationsNumber(accountInfo.KeyUID)
	if err != nil {
		return err
	}

	err = b.ensureDBsOpened(multiaccounts.Account{KeyUID: accountInfo.KeyUID, KDFIterations: kdfIterations}, currPassword)
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
	const pathWalletRoot = "m/44'/60'/0'/0"
	const pathEIP1581 = "m/43'/60'/1581'"
	var paths []string
	paths = append(paths, pathWalletRoot, pathEIP1581)
	for _, acc := range knownAccounts {
		if accountInfo.KeyUID == acc.KeyUID {
			paths = append(paths, acc.Path)
		}
	}

	_, err = b.accountManager.AccountsGenerator().StoreAccount(accountInfo.ID, currPassword)
	if err != nil {
		return err
	}

	_, err = b.accountManager.AccountsGenerator().StoreDerivedAccounts(accountInfo.ID, currPassword, paths)
	if err != nil {
		return err
	}

	err = b.multiaccountsDB.UpdateAccountKeycardPairing(accountInfo.KeyUID, "")
	if err != nil {
		return err
	}

	err = messenger.DeleteAllKeycardsWithKeyUID(context.Background(), accountInfo.KeyUID)
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

	return b.ChangeDatabasePassword(accountInfo.KeyUID, currPassword, newPassword)
}

func (b *GethStatusBackend) VerifyDatabasePassword(keyUID string, password string) error {
	kdfIterations, err := b.multiaccountsDB.GetAccountKDFIterationsNumber(keyUID)
	if err != nil {
		return err
	}

	if !b.appDBExists(keyUID) || !b.walletDBExists(keyUID) {
		return errors.New("One or more databases not created")
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

func enrichMultiAccountBySubAccounts(account *multiaccounts.Account, subaccs []*accounts.Account) error {
	if account.ColorHash != nil && account.ColorID != 0 {
		return nil
	}

	for i, acc := range subaccs {
		subaccs[i].KeyUID = account.KeyUID
		if acc.Chat {
			pk := string(acc.PublicKey.Bytes())
			colorHash, err := colorhash.GenerateFor(pk)
			if err != nil {
				return err
			}
			account.ColorHash = colorHash

			colorID, err := identityutils.ToColorID(pk)
			if err != nil {
				return err
			}
			account.ColorID = colorID

			break
		}
	}

	return nil
}

func enrichMultiAccountByPublicKey(account *multiaccounts.Account, publicKey types.HexBytes) error {
	pk := string(publicKey.Bytes())
	colorHash, err := colorhash.GenerateFor(pk)
	if err != nil {
		return err
	}
	account.ColorHash = colorHash

	colorID, err := identityutils.ToColorID(pk)
	if err != nil {
		return err
	}
	account.ColorID = colorID

	return nil
}

// Deprecated: Use CreateAccountAndLogin instead
func (b *GethStatusBackend) SaveAccountAndStartNodeWithKey(
	account multiaccounts.Account,
	password string,
	settings settings.Settings,
	nodecfg *params.NodeConfig,
	subaccs []*accounts.Account,
	keyHex string,
) error {
	err := enrichMultiAccountBySubAccounts(&account, subaccs)
	if err != nil {
		return err
	}
	err = b.SaveAccount(account)
	if err != nil {
		return err
	}
	err = b.ensureDBsOpened(account, password)
	if err != nil {
		return err
	}
	err = b.saveAccountsAndSettings(settings, nodecfg, subaccs)
	if err != nil {
		return err
	}
	return b.StartNodeWithKey(account, password, keyHex, nodecfg)
}

// StartNodeWithAccountAndInitialConfig is used after account and config was generated.
// In current setup account name and config is generated on the client side. Once/if it will be generated on
// status-go side this flow can be simplified.
// TODO: Consider passing accountBundle here directly
func (b *GethStatusBackend) StartNodeWithAccountAndInitialConfig(
	account multiaccounts.Account,
	password string,
	settings settings.Settings,
	nodecfg *params.NodeConfig,
	subaccs []*accounts.Account,
	chatKey *ecdsa.PrivateKey,
) error {
	err := enrichMultiAccountBySubAccounts(&account, subaccs)
	if err != nil {
		return err
	}
	err = b.SaveAccount(account)
	if err != nil {
		return err
	}
	err = b.ensureDBsOpened(account, password)
	if err != nil {
		return err
	}
	err = b.saveAccountsAndSettings(settings, nodecfg, subaccs)
	if err != nil {
		return err
	}
	return b.StartNodeWithAccount(account, password, nodecfg, chatKey)
}

// TODO: change in `saveAccountsAndSettings` function param `subaccs []*accounts.Account` parameter to `profileKeypair *accounts.Keypair` parameter
func (b *GethStatusBackend) saveAccountsAndSettings(settings settings.Settings, nodecfg *params.NodeConfig, subaccs []*accounts.Account) error {
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

	// In case of setting up new account either way (creating new, importing seed phrase, keycard account...) we should not
	// back up any data after login, as it was the case before, that's the reason why we're setting last backup time to the time
	// when an account was created.
	now := time.Now().Unix()
	err = accdb.SetLastBackup(uint64(now))
	if err != nil {
		return err
	}

	keypair := &accounts.Keypair{
		KeyUID:                  settings.KeyUID,
		Name:                    settings.DisplayName,
		Type:                    accounts.KeypairTypeProfile,
		DerivedFrom:             settings.Address.String(),
		LastUsedDerivationIndex: 0,
	}

	// When creating a new account, the chat account should have position -1, cause it doesn't participate
	// in the wallet view and default wallet account should be at position 0.
	for _, acc := range subaccs {
		if acc.Chat {
			acc.Position = -1
		}
		if acc.Wallet {
			acc.Position = 0
		}
		acc.Operable = accounts.AccountFullyOperable
		keypair.Accounts = append(keypair.Accounts, acc)
	}

	return accdb.SaveOrUpdateKeypair(keypair)
}

func (b *GethStatusBackend) loadNodeConfig(inputNodeCfg *params.NodeConfig) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	conf, err := nodecfg.GetNodeConfigFromDB(b.appDB)
	if err != nil {
		return err
	}

	if inputNodeCfg != nil {
		// If an installationID is provided, we override it
		if conf != nil && conf.ShhextConfig.InstallationID != "" {
			inputNodeCfg.ShhextConfig.InstallationID = conf.ShhextConfig.InstallationID
		}

		conf, err = b.OverwriteNodeConfigValues(conf, inputNodeCfg)
		if err != nil {
			return err
		}
	}

	// Start WakuV1 if WakuV2 is not enabled
	conf.WakuConfig.Enabled = !conf.WakuV2Config.Enabled
	// NodeConfig.Version should be taken from version.Version
	// which is set at the compile time.
	// What's cached is usually outdated so we overwrite it here.
	conf.Version = version.Version()
	conf.RootDataDir = b.rootDataDir
	conf.DataDir = filepath.Join(b.rootDataDir, conf.DataDir)
	conf.KeyStoreDir = filepath.Join(b.rootDataDir, conf.KeyStoreDir)

	if _, err = os.Stat(conf.RootDataDir); os.IsNotExist(err) {
		if err := os.MkdirAll(conf.RootDataDir, os.ModePerm); err != nil {
			b.logger.Warn("failed to create data directory", zap.Error(err))
			return err
		}
	}

	if len(conf.LogDir) == 0 {
		conf.LogFile = filepath.Join(b.rootDataDir, conf.LogFile)
	} else {
		conf.LogFile = filepath.Join(conf.LogDir, conf.LogFile)
	}

	b.config = conf

	if inputNodeCfg != nil && inputNodeCfg.RuntimeLogLevel != "" {
		b.config.LogLevel = inputNodeCfg.RuntimeLogLevel
	}

	return nil
}

func (b *GethStatusBackend) saveNodeConfig(n *params.NodeConfig) error {
	err := nodecfg.SaveNodeConfig(b.appDB, n)
	if err != nil {
		return err
	}
	return nil
}

func (b *GethStatusBackend) GetNodeConfig() (*params.NodeConfig, error) {
	return nodecfg.GetNodeConfigFromDB(b.appDB)
}

func (b *GethStatusBackend) startNode(config *params.NodeConfig) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("node crashed on start: %v", err)
			sentry.RecoverError(err)
		}
	}()

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

	if b.accountManager.GetManager() == nil {
		err = b.accountManager.InitKeystore(config.KeyStoreDir)
		if err != nil {
			return err
		}
	}

	manager := b.accountManager.GetManager()
	if manager == nil {
		return errors.New("ethereum accounts.Manager is nil")
	}

	if err = b.statusNode.StartWithOptions(config, node.StartOptions{
		// The peers discovery protocols are started manually after
		// `node.ready` signal is sent.
		// It was discussed in https://github.com/status-im/status-go/pull/1333.
		StartDiscovery:  false,
		AccountsManager: manager,
	}); err != nil {
		return
	}
	b.accountManager.SetRPCClient(b.statusNode.RPCClient(), rpc.DefaultCallTimeout)
	signal.SendNodeStarted()

	b.transactor.SetNetworkID(config.NetworkID)
	b.transactor.SetRPC(b.statusNode.RPCClient(), rpc.DefaultCallTimeout)
	b.personalAPI.SetRPC(b.statusNode.RPCClient(), rpc.DefaultCallTimeout)

	if err = b.registerHandlers(); err != nil {
		b.logger.Error("Handler registration failed", zap.Error(err))
		return
	}
	b.logger.Info("Handlers registered")

	// Handle a case when a node is stopped and resumed.
	// If there is no account selected, an error is returned.
	if _, err := b.accountManager.SelectedChatAccount(); err == nil {
		if err := b.injectAccountsIntoServices(); err != nil {
			return err
		}
	} else if err != account.ErrNoAccountSelected {
		return err
	}

	if b.statusNode.WalletService() != nil {
		b.statusNode.WalletService().KeycardPairings().SetKeycardPairingsFile(config.KeycardPairingDataFile)
	}

	signal.SendNodeReady()

	if err := b.statusNode.StartDiscovery(); err != nil {
		return err
	}

	return nil
}

// StopNode stop Status node. Stopped node cannot be resumed.
func (b *GethStatusBackend) StopNode() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.stopNode()
}

func (b *GethStatusBackend) stopNode() error {
	if b.statusNode == nil || !b.IsNodeRunning() {
		return nil
	}
	if !b.LocalPairingStateManager.IsPairing() {
		defer signal.SendNodeStopped()
	}

	return b.statusNode.Stop()
}

// RestartNode restart running Status node, fails if node is not running
func (b *GethStatusBackend) RestartNode() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if !b.IsNodeRunning() {
		return node.ErrNoRunningNode
	}

	if err := b.stopNode(); err != nil {
		return err
	}

	return b.startNode(b.config)
}

// ResetChainData remove chain data from data directory.
// Node is stopped, and new node is started, with clean data directory.
func (b *GethStatusBackend) ResetChainData() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if err := b.stopNode(); err != nil {
		return err
	}
	// config is cleaned when node is stopped
	if err := b.statusNode.ResetChainData(b.config); err != nil {
		return err
	}
	signal.SendChainDataRemoved()
	return b.startNode(b.config)
}

// CallRPC executes public RPC requests on node's in-proc RPC server.
func (b *GethStatusBackend) CallRPC(inputJSON string) (string, error) {
	client := b.statusNode.RPCClient()
	if client == nil {
		return "", ErrRPCClientUnavailable
	}
	return client.CallRaw(inputJSON), nil
}

// CallPrivateRPC executes public and private RPC requests on node's in-proc RPC server.
func (b *GethStatusBackend) CallPrivateRPC(inputJSON string) (string, error) {
	client := b.statusNode.RPCClient()
	if client == nil {
		return "", ErrRPCClientUnavailable
	}
	return client.CallRaw(inputJSON), nil
}

// SendTransaction creates a new transaction and waits until it's complete.
func (b *GethStatusBackend) SendTransaction(sendArgs wallettypes.SendTxArgs, password string) (hash types.Hash, err error) {
	verifiedAccount, err := b.getVerifiedWalletAccount(sendArgs.From.String(), password)
	if err != nil {
		return hash, err
	}

	hash, _, err = b.transactor.SendTransaction(sendArgs, verifiedAccount, -1)
	return hash, err
}

func (b *GethStatusBackend) SendTransactionWithChainID(chainID uint64, sendArgs wallettypes.SendTxArgs, password string) (hash types.Hash, err error) {
	verifiedAccount, err := b.getVerifiedWalletAccount(sendArgs.From.String(), password)
	if err != nil {
		return hash, err
	}

	hash, _, err = b.transactor.SendTransactionWithChainID(chainID, sendArgs, -1, verifiedAccount)
	return hash, err
}

func (b *GethStatusBackend) SendTransactionWithSignature(sendArgs wallettypes.SendTxArgs, sig []byte) (hash types.Hash, err error) {
	txWithSignature, err := b.transactor.BuildTransactionWithSignature(b.transactor.NetworkID(), sendArgs, sig)
	if err != nil {
		return hash, err
	}

	return b.transactor.SendTransactionWithSignature(common.Address(sendArgs.From), sendArgs.Symbol, sendArgs.MultiTransactionID, txWithSignature)
}

// HashTransaction validate the transaction and returns new sendArgs and the transaction hash.
func (b *GethStatusBackend) HashTransaction(sendArgs wallettypes.SendTxArgs) (wallettypes.SendTxArgs, types.Hash, error) {
	return b.transactor.HashTransaction(sendArgs)
}

// SignMessage checks the pwd vs the selected account and passes on the signParams
// to personalAPI for message signature
func (b *GethStatusBackend) SignMessage(rpcParams personal.SignParams) (types.HexBytes, error) {
	verifiedAccount, err := b.getVerifiedWalletAccount(rpcParams.Address, rpcParams.Password)
	if err != nil {
		return types.HexBytes{}, err
	}
	return b.personalAPI.Sign(rpcParams, verifiedAccount)
}

// Recover calls the personalAPI to return address associated with the private
// key that was used to calculate the signature in the message
func (b *GethStatusBackend) Recover(rpcParams personal.RecoverParams) (types.Address, error) {
	return b.personalAPI.Recover(rpcParams)
}

// SignTypedData accepts data and password. Gets verified account and signs typed data.
func (b *GethStatusBackend) SignTypedData(typed typeddata.TypedData, address string, password string) (types.HexBytes, error) {
	account, err := b.getVerifiedWalletAccount(address, password)
	if err != nil {
		return types.HexBytes{}, err
	}
	chain := new(big.Int).SetUint64(b.StatusNode().Config().NetworkID)
	sig, err := typeddata.Sign(typed, account.AccountKey.PrivateKey, chain)
	if err != nil {
		return types.HexBytes{}, err
	}
	return types.HexBytes(sig), err
}

// SignTypedDataV4 accepts data and password. Gets verified account and signs typed data.
func (b *GethStatusBackend) SignTypedDataV4(typed signercore.TypedData, address string, password string) (types.HexBytes, error) {
	account, err := b.getVerifiedWalletAccount(address, password)
	if err != nil {
		return types.HexBytes{}, err
	}
	chain := new(big.Int).SetUint64(b.StatusNode().Config().NetworkID)
	sig, err := typeddata.SignTypedDataV4(typed, account.AccountKey.PrivateKey, chain)
	if err != nil {
		return types.HexBytes{}, err
	}
	return types.HexBytes(sig), err
}

// HashTypedData generates the hash of TypedData.
func (b *GethStatusBackend) HashTypedData(typed typeddata.TypedData) (types.Hash, error) {
	chain := new(big.Int).SetUint64(b.StatusNode().Config().NetworkID)
	hash, err := typeddata.ValidateAndHash(typed, chain)
	if err != nil {
		return types.Hash{}, err
	}
	return types.Hash(hash), err
}

// HashTypedDataV4 generates the hash of TypedData.
func (b *GethStatusBackend) HashTypedDataV4(typed signercore.TypedData) (types.Hash, error) {
	chain := new(big.Int).SetUint64(b.StatusNode().Config().NetworkID)
	hash, err := typeddata.HashTypedDataV4(typed, chain)
	if err != nil {
		return types.Hash{}, err
	}
	return types.Hash(hash), err
}

func (b *GethStatusBackend) getVerifiedWalletAccount(address, password string) (*account.SelectedExtKey, error) {
	config := b.StatusNode().Config()
	db, err := accounts.NewDB(b.appDB)
	if err != nil {
		b.logger.Error("failed to create new *Database instance", zap.Error(err))
		return nil, err
	}
	exists, err := db.AddressExists(types.HexToAddress(address))
	if err != nil {
		b.logger.Error("failed to query db for a given address", zap.String("address", address), zap.Error(err))
		return nil, err
	}

	if !exists {
		b.logger.Error("failed to get a selected account", zap.Error(wallettypes.ErrInvalidTxSender))
		return nil, wallettypes.ErrAccountDoesntExist
	}

	key, err := b.accountManager.VerifyAccountPassword(config.KeyStoreDir, address, password)
	if _, ok := err.(*account.ErrCannotLocateKeyFile); ok {
		key, err = b.generatePartialAccountKey(db, address, password)
		if err != nil {
			return nil, err
		}
	}

	if err != nil {
		b.logger.Error("failed to verify account", zap.String("account", address), zap.Error(err))
		return nil, err
	}

	return &account.SelectedExtKey{
		Address:    key.Address,
		AccountKey: key,
	}, nil
}

func (b *GethStatusBackend) generatePartialAccountKey(db *accounts.Database, address string, password string) (*types.Key, error) {
	dbPath, err := db.GetPath(types.HexToAddress(address))
	path := "m/" + dbPath[strings.LastIndex(dbPath, "/")+1:]
	if err != nil {
		b.logger.Error("failed to get path for given account address", zap.String("account", address), zap.Error(err))
		return nil, err
	}

	rootAddress, err := db.GetWalletRootAddress()
	if err != nil {
		return nil, err
	}
	info, err := b.accountManager.AccountsGenerator().LoadAccount(rootAddress.Hex(), password)
	if err != nil {
		return nil, err
	}
	masterID := info.ID

	accInfosMap, err := b.accountManager.AccountsGenerator().StoreDerivedAccounts(masterID, password, []string{path})
	if err != nil {
		return nil, err
	}

	_, key, err := b.accountManager.AddressToDecryptedAccount(accInfosMap[path].Address, password)
	if err != nil {
		return nil, err
	}

	return key, nil
}

// registerHandlers attaches Status callback handlers to running node
func (b *GethStatusBackend) registerHandlers() error {
	var clients []*rpc.Client

	if c := b.StatusNode().RPCClient(); c != nil {
		clients = append(clients, c)
	} else {
		return errors.New("RPC client unavailable")
	}

	for _, client := range clients {
		client.RegisterHandler(
			params.AccountsMethodName,
			func(context.Context, uint64, ...interface{}) (interface{}, error) {
				return b.accountManager.Accounts()
			},
		)

		if b.allowAllRPC {
			// this should only happen in unit-tests, this variable is not available outside this package
			continue
		}
		client.RegisterHandler(params.SendTransactionMethodName, unsupportedMethodHandler)
		client.RegisterHandler(params.PersonalSignMethodName, unsupportedMethodHandler)
		client.RegisterHandler(params.PersonalRecoverMethodName, unsupportedMethodHandler)
	}

	return nil
}

func unsupportedMethodHandler(ctx context.Context, chainID uint64, rpcParams ...interface{}) (interface{}, error) {
	return nil, ErrUnsupportedRPCMethod
}

// ConnectionChange handles network state changes logic.
func (b *GethStatusBackend) ConnectionChange(typ string, expensive bool) {
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
	b.statusNode.ConnectionChanged(state)

	// logic of handling state changes here
	// restart node? force peers reconnect? etc
}

// AppStateChange handles app state changes (background/foreground).
// state values: see https://facebook.github.io/react-native/docs/appstate.html
func (b *GethStatusBackend) AppStateChange(state AppState) {
	if !state.IsValid() {
		b.logger.Warn("invalid app state, not reporting app state change", zap.Any("state", state))
		return
	}

	var messenger *protocol.Messenger

	b.appState = state

	if b.statusNode == nil {
		b.logger.Warn("statusNode nil, not reporting app state change")
		return
	}

	if b.statusNode.WakuExtService() != nil {
		messenger = b.statusNode.WakuExtService().Messenger()
	}

	if b.statusNode.WakuV2ExtService() != nil {
		messenger = b.statusNode.WakuV2ExtService().Messenger()
	}

	if messenger == nil {
		b.logger.Warn("messenger nil, not reporting app state change")
		return
	}

	if state == AppStateForeground {
		messenger.ToForeground()
	} else {
		messenger.ToBackground()
	}

	// TODO: put node in low-power mode if the app is in background (or inactive)
	// and normal mode if the app is in foreground.
}

func (b *GethStatusBackend) StopLocalNotifications() error {
	if b.statusNode == nil {
		return nil
	}
	return b.statusNode.StopLocalNotifications()
}

func (b *GethStatusBackend) StartLocalNotifications() error {
	if b.statusNode == nil {
		return nil
	}
	return b.statusNode.StartLocalNotifications()

}

// Logout clears whisper identities.
func (b *GethStatusBackend) Logout() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.logger.Debug("logging out")
	err := b.cleanupServices()
	if err != nil {
		return err
	}
	err = b.closeDBs()
	if err != nil {
		return err
	}

	b.AccountManager().Logout()
	b.account = nil

	if b.statusNode != nil {
		if err := b.statusNode.Stop(); err != nil {
			return err
		}
		b.statusNode = nil
	}

	if !b.LocalPairingStateManager.IsPairing() {
		signal.SendNodeStopped()
	}

	// re-initialize the node, at some point we should better manage the lifecycle
	b.initialize()

	err = b.statusNode.StartMediaServerWithoutDB()
	if err != nil {
		b.logger.Error("failed to start media server without app db", zap.Error(err))
		return err
	}
	return nil
}

// cleanupServices stops parts of services that doesn't managed by a node and removes injected data from services.
func (b *GethStatusBackend) cleanupServices() error {
	b.selectedAccountKeyID = ""
	if b.statusNode == nil {
		return nil
	}
	return b.statusNode.Cleanup()
}

func (b *GethStatusBackend) closeDBs() error {
	err := b.closeWalletDB()
	if err != nil {
		return err
	}
	return b.closeAppDB()
}

func (b *GethStatusBackend) closeAppDB() error {
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

func (b *GethStatusBackend) closeWalletDB() error {
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
func (b *GethStatusBackend) SelectAccount(loginParams account.LoginParams) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.AccountManager().RemoveOnboarding()

	err := b.accountManager.SelectAccount(loginParams)
	if err != nil {
		return err
	}

	if loginParams.MultiAccount != nil {
		b.account = loginParams.MultiAccount
	}

	if err := b.injectAccountsIntoServices(); err != nil {
		return err
	}

	return nil
}

func (b *GethStatusBackend) GetActiveAccount() (*multiaccounts.Account, error) {
	if b.account == nil {
		return nil, errors.New("master key account is nil in the GethStatusBackend")
	}

	return b.account, nil
}

func (b *GethStatusBackend) LocalPairingStarted() error {
	if b.account == nil {
		return errors.New("master key account is nil in the GethStatusBackend")
	}

	accountDB, err := accounts.NewDB(b.appDB)
	if err != nil {
		return err
	}

	return accountDB.MnemonicWasShown()
}

func (b *GethStatusBackend) injectAccountsIntoWakuService(w types.WakuKeyManager, st *ext.Service) error {
	chatAccount, err := b.accountManager.SelectedChatAccount()
	if err != nil {
		return err
	}

	identity := chatAccount.AccountKey.PrivateKey

	acc, err := b.GetActiveAccount()
	if err != nil {
		return err
	}

	if err := w.DeleteKeyPairs(); err != nil { // err is not possible; method return value is incorrect
		return err
	}
	b.selectedAccountKeyID, err = w.AddKeyPair(identity)
	if err != nil {
		return ErrWakuIdentityInjectionFailure
	}

	if st != nil {
		if err := st.InitProtocol(b.statusNode.GethNode().Config().Name, identity, b.appDB, b.walletDB, b.statusNode.HTTPServer(), b.multiaccountsDB, acc, b.accountManager, b.statusNode.RPCClient(), b.statusNode.WalletService(), b.statusNode.CommunityTokensService(), b.statusNode.WakuV2Service(), logutils.ZapLogger(), b.statusNode.AccountsFeed()); err != nil {
			return err
		}
		// Set initial connection state
		st.ConnectionChanged(b.connectionState)

		messenger := st.Messenger()
		// Init public status api
		b.statusNode.StatusPublicService().Init(messenger)
		b.statusNode.AccountService().Init(messenger)
		// Init chat service
		accDB, err := accounts.NewDB(b.appDB)
		if err != nil {
			return err
		}
		b.statusNode.ChatService(accDB).Init(messenger)
		b.statusNode.EnsService().Init(messenger.SyncEnsNamesWithDispatchMessage)
		b.statusNode.CommunityTokensService().Init(messenger)
	}

	return nil
}

func (b *GethStatusBackend) InstallationID() string {
	m := b.Messenger()
	if m != nil {
		return m.InstallationID()
	}
	return ""
}

func (b *GethStatusBackend) KeyUID() string {
	m := b.Messenger()
	if m != nil {
		return m.KeyUID()
	}
	return ""
}

func (b *GethStatusBackend) injectAccountsIntoServices() error {
	if b.statusNode.WakuService() != nil {
		return b.injectAccountsIntoWakuService(b.statusNode.WakuService(), func() *ext.Service {
			if b.statusNode.WakuExtService() == nil {
				return nil
			}
			return b.statusNode.WakuExtService().Service
		}())
	}

	if b.statusNode.WakuV2Service() != nil {
		return b.injectAccountsIntoWakuService(b.statusNode.WakuV2Service(), func() *ext.Service {
			if b.statusNode.WakuV2ExtService() == nil {
				return nil
			}
			return b.statusNode.WakuV2ExtService().Service
		}())
	}

	return nil
}

// ExtractGroupMembershipSignatures extract signatures from tuples of content/signature
func (b *GethStatusBackend) ExtractGroupMembershipSignatures(signaturePairs [][2]string) ([]string, error) {
	return crypto.ExtractSignatures(signaturePairs)
}

// SignGroupMembership signs a piece of data containing membership information
func (b *GethStatusBackend) SignGroupMembership(content string) (string, error) {
	selectedChatAccount, err := b.accountManager.SelectedChatAccount()
	if err != nil {
		return "", err
	}

	return crypto.SignStringAsHex(content, selectedChatAccount.AccountKey.PrivateKey)
}

func (b *GethStatusBackend) Messenger() *protocol.Messenger {
	node := b.StatusNode()
	if node != nil {
		accountService := node.AccountService()
		if accountService != nil {
			return accountService.GetMessenger()
		}
	}
	return nil
}

// SignHash exposes vanilla ECDSA signing for signing a message for Swarm
func (b *GethStatusBackend) SignHash(hexEncodedHash string) (string, error) {
	hash, err := hexutil.Decode(hexEncodedHash)
	if err != nil {
		return "", fmt.Errorf("SignHash: could not unmarshal the input: %v", err)
	}

	chatAccount, err := b.accountManager.SelectedChatAccount()
	if err != nil {
		return "", fmt.Errorf("SignHash: could not select account: %v", err.Error())
	}

	signature, err := ethcrypto.Sign(hash, chatAccount.AccountKey.PrivateKey)
	if err != nil {
		return "", fmt.Errorf("SignHash: could not sign the hash: %v", err)
	}

	hexEncodedSignature := types.EncodeHex(signature)
	return hexEncodedSignature, nil
}

func (b *GethStatusBackend) SwitchFleet(fleet string, conf *params.NodeConfig) error {
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

func (b *GethStatusBackend) getAppDBPath(keyUID string) (string, error) {
	if len(b.rootDataDir) == 0 {
		return "", errors.New("root datadir wasn't provided")
	}

	return filepath.Join(b.rootDataDir, fmt.Sprintf("%s-v4.db", keyUID)), nil
}

func (b *GethStatusBackend) getWalletDBPath(keyUID string) (string, error) {
	if len(b.rootDataDir) == 0 {
		return "", errors.New("root datadir wasn't provided")
	}

	return filepath.Join(b.rootDataDir, fmt.Sprintf("%s-wallet.db", keyUID)), nil
}

func (b *GethStatusBackend) SetSentryDSN(dsn string) {
	b.sentryDSN = dsn
}

func (b *GethStatusBackend) EnablePanicReporting() error {
	return sentry.Init(
		sentry.WithDSN(b.sentryDSN),
		sentry.WithDefaultContext(),
	)
}

func (b *GethStatusBackend) DisablePanicReporting() error {
	return sentry.Close()
}

func (b *GethStatusBackend) TogglePanicReporting(enabled bool) error {
	if enabled {
		return b.EnablePanicReporting()
	}
	return b.DisablePanicReporting()
}
