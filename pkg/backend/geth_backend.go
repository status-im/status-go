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
	"path/filepath"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/afex/hystrix-go/hystrix"
	"github.com/imdario/mergo"
	"github.com/pkg/errors"
	"github.com/status-im/extkeys"
	"go.uber.org/zap"

	"github.com/ethereum/go-ethereum/common/hexutil"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	signercore "github.com/ethereum/go-ethereum/signer/core/apitypes"

	accsmanagement "github.com/status-im/status-go/internal/accounts-management"
	"github.com/status-im/status-go/internal/accounts-management/common"
	generator "github.com/status-im/status-go/internal/accounts-management/generator"
	"github.com/status-im/status-go/internal/accounts-management/keystore/envelope"
	accsmanagementtypes "github.com/status-im/status-go/internal/accounts-management/types"
	"github.com/status-im/status-go/internal/connection"
	"github.com/status-im/status-go/internal/crypto"
	types "github.com/status-im/status-go/internal/crypto/types"
	"github.com/status-im/status-go/internal/db/appdatabase"
	"github.com/status-im/status-go/internal/db/dbsetup"
	"github.com/status-im/status-go/internal/db/multiaccounts"
	"github.com/status-im/status-go/internal/db/multiaccounts/accounts"
	multiacccommon "github.com/status-im/status-go/internal/db/multiaccounts/common"
	settings "github.com/status-im/status-go/internal/db/multiaccounts/settings"
	"github.com/status-im/status-go/internal/db/sqlite"
	"github.com/status-im/status-go/internal/db/walletdb"
	"github.com/status-im/status-go/internal/images"
	"github.com/status-im/status-go/internal/instrumentation/trace"
	"github.com/status-im/status-go/internal/ipfs"
	logutils "github.com/status-im/status-go/internal/logutils"
	"github.com/status-im/status-go/internal/metrics"
	"github.com/status-im/status-go/internal/nodecfg"
	"github.com/status-im/status-go/internal/panics"
	"github.com/status-im/status-go/internal/platform"
	"github.com/status-im/status-go/internal/protocol"
	"github.com/status-im/status-go/internal/protocol/communities"
	identityutils "github.com/status-im/status-go/internal/protocol/identity"
	"github.com/status-im/status-go/internal/protocol/identity/colorhash"
	"github.com/status-im/status-go/internal/protocol/requests"
	"github.com/status-im/status-go/internal/rpc"
	"github.com/status-im/status-go/internal/signal"
	"github.com/status-im/status-go/internal/transactions"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/params/networkdefaults"
	"github.com/status-im/status-go/pkg/backend/node"
	nodeadapters "github.com/status-im/status-go/pkg/backend/node/adapters"
	"github.com/status-im/status-go/pkg/sentry"
	"github.com/status-im/status-go/pkg/services/ens"
	"github.com/status-im/status-go/pkg/services/ext"
	"github.com/status-im/status-go/pkg/services/pairing/statecontrol"
	"github.com/status-im/status-go/pkg/services/personal"
	"github.com/status-im/status-go/pkg/services/typeddata"
	"github.com/status-im/status-go/pkg/services/wallet"
	"github.com/status-im/status-go/pkg/services/wallet/wallettypes"
	"github.com/status-im/status-go/pkg/version"
)

var (
	// ErrDBNotAvailable is returned if a method is called before the DB is available for usage
	ErrDBNotAvailable = errors.New("DB is unavailable")
	newAccountsDB     = accounts.NewDB
)

type LoginParams struct {
	ChatAddress    types.Address                `json:"chatAddress"`
	Password       string                       `json:"password"`
	MainAccount    types.Address                `json:"mainAccount"` // TODO: remove this field
	MultiAccount   *multiaccounts.Account       `json:"multiAccount"`
	ProfileKeypair *accsmanagementtypes.Keypair `json:"-"`
}

// StatusBackend implements the Status.im service over go-ethereum
type StatusBackend struct {
	mu sync.Mutex
	// rootDataDir is the same for all networks.
	rootDataDir string
	appDB       *sql.DB
	walletDB    *sql.DB
	config      *params.NodeConfig

	statusNode               *node.StatusNode
	signer                   communities.MessageSigner
	multiaccountsDB          *multiaccounts.Database
	account                  *multiaccounts.Account
	accountsManager          *accsmanagement.AccountsManager
	connectionState          connection.State
	transactor               *transactions.Transactor
	LocalPairingStateManager *statecontrol.ProcessStateManager
	prometheusMetrics        *metrics.Server
	sentryDSN                string

	logger            *zap.Logger
	preLoginLogConfig *logutils.PreLoginLogConfig

	secretCache profileSecretCache

	shutdownTasks []func() error
}

// dbCredentials carries the resolved sqlcipher credentials for opening the profile databases.
type dbCredentials struct {
	secret          string
	kdfIter         int
	fallbackSecret  string
	fallbackKdfIter int
}

// openDBWithCredsFallback opens a database with the resolved secret, retrying with the fallback credentials (when present)
// so a crash mid-migration never locks the user out. The primary error is reported if both fail.
func (b *StatusBackend) openDBWithCredsFallback(dbName string, creds dbCredentials, open func(secret string, kdfIter int) (*sql.DB, error)) (*sql.DB, error) {
	db, err := open(creds.secret, creds.kdfIter)
	if err == nil || creds.fallbackSecret == "" || creds.fallbackSecret == creds.secret {
		return db, err
	}

	fallbackDB, fallbackErr := open(creds.fallbackSecret, creds.fallbackKdfIter)
	if fallbackErr != nil {
		return nil, err
	}
	b.logger.Warn("database opened with legacy credentials after failed DEK open; encryption-scheme migration was likely interrupted",
		zap.String("db", dbName))
	return fallbackDB, nil
}

// NewStatusBackend create a new StatusBackend instance
func NewStatusBackend(logger *zap.Logger) *StatusBackend {
	logger = logger.Named("StatusBackend")
	backend := &StatusBackend{
		logger:            logger,
		preLoginLogConfig: logutils.NewPreLoginLogConfig(),
		shutdownTasks:     []func() error{},
	}
	if err := backend.initialize(); err != nil {
		logger.Error("failed to initialize backend", zap.Error(err))
		panic(err)
	}

	logger.Info("Status backend initialized",
		zap.String("backend geth version", version.Version()),
		zap.String("commit", version.GitCommit()),
		zap.String("IpfsGatewayURL", ipfs.GatewayURL))

	if platform.IsMobilePlatform() {
		// A limit below the working set (~350-620MB while syncing a large
		// community) forces the GC into permanent maximum-aggression mode;
		// 500MB keeps an OOM guard while leaving the GC headroom for the
		// common case.
		debug.SetMemoryLimit(500 * 1024 * 1024) // 500MB
	}

	return backend
}

func (b *StatusBackend) PreLoginLog() *logutils.PreLoginLogConfig {
	return b.preLoginLogConfig
}

func (b *StatusBackend) initialize() (err error) {
	accountsManager, err := accsmanagement.NewAccountsManager(b.logger)
	if err != nil {
		b.logger.Error("failed to create new *AccountsManager instance", zap.Error(err))
		return
	}

	transactor := transactions.NewTransactor()
	personalService := personal.New()
	statusNode := node.New(transactor, accountsManager, b.logger)

	b.statusNode = statusNode
	b.accountsManager = accountsManager
	b.transactor = transactor
	b.signer = personalService
	b.statusNode.SetMultiaccountsDB(b.multiaccountsDB)
	b.LocalPairingStateManager = new(statecontrol.ProcessStateManager)
	b.LocalPairingStateManager.SetPairing(false)
	b.LocalPairingStateManager.SetMessageSyncEnabled(false)

	return
}

// StatusNode returns reference to node manager
func (b *StatusBackend) StatusNode() *node.StatusNode {
	return b.statusNode
}

// AccountsManager returns reference to accounts manager
func (b *StatusBackend) AccountsManager() *accsmanagement.AccountsManager {
	return b.accountsManager
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
	return b.statusNode.IsRunning()
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
	b.statusNode.ConnectionChanged(b.connectionState)

	return nil
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
	b.statusNode.ConnectionChanged(state)
}

func (b *StatusBackend) UpdateRootDataDir(datadir string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.rootDataDir = datadir
	b.accountsManager.SetRootDataDir(datadir)
}

func (b *StatusBackend) GetMultiaccountDB() *multiaccounts.Database {
	return b.multiaccountsDB
}

func (b *StatusBackend) OpenAccounts() error {
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

	// Probably we should iron out a bit better how to create/dispose of the status-service
	b.statusNode.SetMultiaccountsDB(db)

	err = b.statusNode.StartMediaServerWithoutDB()
	if err != nil {
		b.logger.Error("failed to start media server without app db", zap.Error(err))
		return err
	}

	return nil
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
				acc.Images[k].LocalURL = b.statusNode.MediaServer().MakeAccountImageURL(acc.KeyUID, v.Name, v.Clock)
			}
			return &acc, nil
		}
	}
	return nil, fmt.Errorf("account with keyUID %s not found", logutils.TruncateWithDot(keyUID))
}

func (b *StatusBackend) SaveAccount(account multiaccounts.Account) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.multiaccountsDB == nil {
		return errors.New("accounts db wasn't initialized")
	}
	return b.multiaccountsDB.SaveAccount(account)
}

func (b *StatusBackend) DeleteMultiaccount(keyUID string, keyStoreDir string) error {
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

	if err := envelope.Remove(b.rootDataDir, keyUID); err != nil {
		return err
	}

	if b.account != nil && b.account.KeyUID == keyUID {
		// reset active account
		b.account = nil
		b.clearProfileSecretCache()
	}

	return os.RemoveAll(keyStoreDir)
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
	resolved, err := b.resolveProfileSecret(account.KeyUID, password, account.KDFIterations)
	if err != nil {
		return err
	}
	creds := dbCredentials{secret: resolved.secret, kdfIter: resolved.dbKdfIter}
	if resolved.migrated {
		creds.fallbackSecret = resolved.legacySecret
		creds.fallbackKdfIter = resolved.legacyKdfIter
	}

	b.mu.Lock()
	dbsUnset := b.walletDB == nil && b.appDB == nil
	b.mu.Unlock()
	if dbsUnset {
		b.setSecretResolverForProfile(account.KeyUID, account.KDFIterations)

		appDBPath, pathErr := b.getAppDBPath(account.KeyUID)
		if pathErr == nil {
			walletDBPath, walletPathErr := b.getWalletDBPath(account.KeyUID)
			legacyAppDBPath := filepath.Join(b.rootDataDir, fmt.Sprintf("%s.db", account.KeyUID))
			unsupportedAppDBPath := filepath.Join(b.rootDataDir, fmt.Sprintf("app-%x.sql", account.KeyUID))
			if walletPathErr == nil && fileExists(walletDBPath) && fileExists(appDBPath) &&
				!fileExists(legacyAppDBPath) && !fileExists(unsupportedAppDBPath) {
				return b.ensureEstablishedDBsOpened(account, creds, appDBPath, walletDBPath)
			}
		}
	}

	return b.ensureDBsOpenedSequential(account, creds)
}

func (b *StatusBackend) ensureEstablishedDBsOpened(account multiaccounts.Account, creds dbCredentials, appDBPath, walletDBPath string) (err error) {
	b.mu.Lock()
	if b.walletDB != nil || b.appDB != nil {
		b.mu.Unlock()
		return b.ensureDBsOpenedSequential(account, creds)
	}
	defer b.mu.Unlock()

	walletResultCh := make(chan struct {
		db  *sql.DB
		err error
	}, 1)
	appResultCh := make(chan struct {
		db  *sql.DB
		err error
	}, 1)
	go func() {
		defer panics.LogOnPanic()
		db, openErr := b.openDBWithCredsFallback("wallet", creds, func(secret string, kdfIter int) (*sql.DB, error) {
			return walletdb.OpenDB(walletDBPath, secret, kdfIter)
		})
		walletResultCh <- struct {
			db  *sql.DB
			err error
		}{db: db, err: openErr}
	}()
	go func() {
		defer panics.LogOnPanic()
		db, openErr := b.openDBWithCredsFallback("app", creds, func(secret string, kdfIter int) (*sql.DB, error) {
			return appdatabase.OpenDB(appDBPath, secret, kdfIter)
		})
		appResultCh <- struct {
			db  *sql.DB
			err error
		}{db: db, err: openErr}
	}()

	walletResult := <-walletResultCh
	appResult := <-appResultCh
	if walletResult.err != nil {
		if appResult.db != nil {
			_ = appResult.db.Close()
		}
		return walletResult.err
	}
	if appResult.err != nil {
		_ = walletResult.db.Close()
		return appResult.err
	}

	if err = walletdb.MigrateDB(walletResult.db); err != nil {
		_ = walletResult.db.Close()
		_ = appResult.db.Close()
		return err
	}
	appdatabase.CurrentAppDBKeyUID = account.KeyUID
	if err = appdatabase.MigrateDB(appResult.db); err != nil {
		_ = walletResult.db.Close()
		_ = appResult.db.Close()
		return err
	}
	accountsDB, err := newAccountsDB(appResult.db)
	if err != nil {
		_ = walletResult.db.Close()
		_ = appResult.db.Close()
		return err
	}
	b.walletDB = walletResult.db
	b.appDB = appResult.db
	b.statusNode.SetWalletDB(b.walletDB)
	b.statusNode.SetAppDB(b.appDB)
	b.accountsManager.SetPersistence(accountsDB)

	return nil
}

func (b *StatusBackend) ensureDBsOpenedSequential(account multiaccounts.Account, creds dbCredentials) (err error) {
	// After wallet DB initial migration, the tables moved to wallet DB are removed from appDB
	// so better migrate wallet DB first to avoid removal if wallet DB migration fails
	if err = b.ensureWalletDBOpened(account, creds); err != nil {
		return err
	}

	if err = b.ensureAppDBOpened(account, creds); err != nil {
		return err
	}

	return nil
}

func (b *StatusBackend) ensureAppDBOpened(account multiaccounts.Account, creds dbCredentials) (err error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.appDB != nil {
		return nil
	}
	if len(b.rootDataDir) == 0 {
		return errors.New("root datadir wasn't provided")
	}

	account.KDFIterations = creds.kdfIter
	dbFilePath, err := b.runDBFileMigrations(account, creds.secret)
	if err != nil {
		return errors.New("Failed to migrate db file: " + err.Error())
	}

	appdatabase.CurrentAppDBKeyUID = account.KeyUID
	b.appDB, err = b.openDBWithCredsFallback("app", creds, func(secret string, kdfIter int) (*sql.DB, error) {
		return appdatabase.InitializeDB(dbFilePath, secret, kdfIter)
	})
	if err != nil {
		b.logger.Error("failed to initialize db", zap.Error(err))
		return err
	}

	b.statusNode.SetAppDB(b.appDB)

	accountsDB, err := accounts.NewDB(b.appDB)
	if err != nil {
		b.logger.Error("failed to create new *Database instance", zap.Error(err))
		return
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

func (b *StatusBackend) ensureWalletDBOpened(account multiaccounts.Account, creds dbCredentials) (err error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.walletDB != nil {
		return nil
	}

	dbWalletPath, err := b.getWalletDBPath(account.KeyUID)
	if err != nil {
		return err
	}

	b.walletDB, err = b.openDBWithCredsFallback("wallet", creds, func(secret string, kdfIter int) (*sql.DB, error) {
		return walletdb.InitializeDB(dbWalletPath, secret, kdfIter)
	})
	if err != nil {
		b.logger.Error("failed to initialize wallet db", zap.Error(err))
		return err
	}
	b.statusNode.SetWalletDB(b.walletDB)
	return nil
}

func (b *StatusBackend) SetupLogSettings() error {
	_ = logutils.ZapLogger().Sync()
	return logutils.OverrideRootLoggerWithConfig(b.config.ProfileLogSettings())
}

// Deprecated: Use StartNodeWithAccount instead.
func (b *StatusBackend) StartNodeWithKey(acc multiaccounts.Account, password string, keyHex string, nodecfg *params.NodeConfig) error {
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
		if err == nil {
			b.statusNode.StartTokenManager()
		}
		return nil
	}
	return b.LoggedIn(acc.KeyUID, err)
}

func (b *StatusBackend) OverwriteNodeConfigValues(conf *params.NodeConfig, n *params.NodeConfig) (*params.NodeConfig, error) {
	if err := mergo.Merge(conf, n, mergo.WithOverride); err != nil {
		return nil, err
	}

	conf.Networks = n.Networks

	if err := b.saveNodeConfig(conf); err != nil {
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

func (b *StatusBackend) overrideNetworks(conf *params.NodeConfig, request *requests.Login, thirdpartyServicesEnabled bool) {
	conf.Networks = networkdefaults.BuildDefaultNetworks(&request.WalletSecretsConfig, thirdpartyServicesEnabled)
}

func (b *StatusBackend) LoginAccount(request *requests.Login) error {
	err := b.loginAccount(request)
	if err != nil {
		// Stop node for clean up
		_ = b.StopNode()
	}
	if b.LocalPairingStateManager.IsPairing() {
		if err == nil {
			b.statusNode.StartTokenManager()
		}
		return nil
	}
	err = b.LoggedIn(request.KeyUID, err)
	if err != nil {
		return errors.Wrap(err, "failed to send LoggedIn signal")
	}

	return nil
}

func (b *StatusBackend) loginAccount(request *requests.Login) error {
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

		_, generatedDerivedAccountsInfo, err := b.generateDerivedAddresses(generatedAccount, paths)
		if err != nil {
			return errors.Wrap(err, "failed to derive children accounts")
		}

		request.Password = generatedDerivedAccountsInfo[common.PathEIP1581Encryption].PublicKey
		request.KeycardWhisperPrivateKey = generatedDerivedAccountsInfo[common.PathEIP1581Chat].PrivateKey
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

	defaultCfg := &params.NodeConfig{
		// why we need this? relate PR: https://github.com/status-im/status-go/pull/4014
		KeycardPairingDataFile: filepath.Join(b.rootDataDir, DefaultKeycardPairingDataFileRelativePath),
	}

	defaultCfg.WalletConfig = buildWalletConfig(&request.WalletConfig, &request.WalletSecretsConfig)

	if request.LogFilePath != "" {
		defaultCfg.LogDir = request.LogFilePath
	}

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

	accountSettings, err := b.GetSettings()
	if err != nil {
		return errors.Wrap(err, "failed to load accountSettings")
	}

	b.overrideNetworks(b.config, request, accountSettings.ThirdpartyServicesEnabled)

	if request.APIConfig != nil {
		overrideApiConfig(b.config, request.APIConfig)
	}

	b.config.WalletConnectProjectID = request.WalletConnectProjectID

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

	profileKeypair, err := accountsDB.GetProfileKeypair()
	if err != nil {
		return errors.Wrap(err, "failed to get profile keypair")
	}

	err = b.StartNode(b.config)
	if err != nil {
		b.logger.Info("failed to start node")
		return errors.Wrap(err, "failed to start node")
	}

	login := LoginParams{
		Password:       request.Password,
		ChatAddress:    chatAddr,
		MainAccount:    walletAddr,
		ProfileKeypair: profileKeypair,
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

	err = b.backfillKeypairsXPubOnLogin(request, accountsDB)
	if err != nil {
		return errors.Wrap(err, "failed to backfill keypairs xpub")
	}

	return nil
}

func (b *StatusBackend) backfillKeypairsXPubOnLogin(request *requests.Login, accountsDB *accounts.Database) error {
	if request.WalletXPub != "" {
		kp, err := accountsDB.GetKeypairByKeyUID(request.KeyUID)
		if err != nil {
			return errors.Wrap(err, "failed to get the profile keypair")
		}
		if kp != nil && kp.XPub == "" {
			extKey, err := extkeys.NewKeyFromString(request.WalletXPub)
			if err != nil {
				return errors.Wrap(err, "invalid wallet xpub provided in the login request")
			}
			if extKey.IsPrivate {
				return errors.New("private extended key provided as the wallet xpub in the login request")
			}
			err = accountsDB.UpdateKeypairXPub(kp.KeyUID, request.WalletXPub, kp.ColdWallet, kp.Clock)
			if err != nil {
				return errors.Wrap(err, "failed to store the profile keypair xpub")
			}
		}
	}

	return b.accountsManager.BackfillKeypairsXPub(request.Password)
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
		effectiveDefault := ResolveDefaultFleet()
		b.logger.Warn("fleet is not supported, overriding with default value",
			zap.String("fleet", fleet),
			zap.String("defaultFleet", effectiveDefault))
		fleet = effectiveDefault
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

func (b *StatusBackend) Login(keyUID, password string) error {
	err := b.startNodeWithAccount(multiaccounts.Account{KeyUID: keyUID}, password, nil, nil)
	if err == nil {
		b.statusNode.StartTokenManager()
	}
	return err
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
	if err == nil {
		b.statusNode.StartTokenManager()
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
	b.statusNode.StartTokenManager()
	return nil
}

func (b *StatusBackend) ExportUnencryptedDatabase(acc multiaccounts.Account, password, directory string) error {
	resolved, err := b.resolveProfileSecret(acc.KeyUID, password, acc.KDFIterations)
	if err != nil {
		return err
	}

	b.mu.Lock()
	defer b.mu.Unlock()
	if b.appDB != nil {
		return nil
	}
	if len(b.rootDataDir) == 0 {
		return errors.New("root datadir wasn't provided")
	}

	acc.KDFIterations = resolved.dbKdfIter
	dbPath, err := b.runDBFileMigrations(acc, resolved.secret)
	if err != nil {
		return err
	}

	err = sqlite.DecryptDB(dbPath, directory, resolved.secret, resolved.dbKdfIter)
	if err != nil {
		b.logger.Error("failed to initialize db", zap.Error(err))
		return err
	}
	return nil
}

func (b *StatusBackend) ImportUnencryptedDatabase(acc multiaccounts.Account, password, databasePath string) error {
	resolved, err := b.resolveProfileSecret(acc.KeyUID, password, acc.KDFIterations)
	if err != nil {
		return err
	}

	b.mu.Lock()
	defer b.mu.Unlock()
	if b.appDB != nil {
		return nil
	}

	path, err := b.getAppDBPath(acc.KeyUID)
	if err != nil {
		return err
	}

	err = sqlite.EncryptDB(databasePath, path, resolved.secret, resolved.dbKdfIter, signal.SendReEncryptionStarted, signal.SendReEncryptionFinished)
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
			b.statusNode.StartTokenManager()
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

	err = accountDB.SaveSettingField(settings.Mnemonic, nil)
	if err != nil {
		return err
	}

	err = accountDB.SaveSettingField(settings.ProfileMigrationNeeded, false)
	if err != nil {
		return err
	}

	err = messenger.MigrateKeypairToColdWallet(context.Background(), account.KeyUID, oldPassword, accsmanagementtypes.ColdWalletTypeStatusKeycard)
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
func (b *StatusBackend) CreateAccountAndLogin(request *requests.CreateAccount) (*multiaccounts.Account, error) {
	validation := &requests.CreateAccountValidation{
		AllowEmptyDisplayName: true,
	}
	if err := request.Validate(validation); err != nil {
		return nil, err
	}

	mnemonic, err := common.CreateRandomMnemonicWithDefaultLength()
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
		finalMnemonic, err = common.CreateRandomMnemonicWithDefaultLength()
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

func (b *StatusBackend) generateDerivedAddresses(genAcc *generator.Account, paths []string) (genDerivedAccounts map[string]*generator.Account, genDerivedAccountsInfo map[string]generator.AccountInfo, err error) {
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

func (b *StatusBackend) buildAccount(request *requests.CreateAccount, keyUID string, customizationColorClock uint64) (*multiaccounts.Account, error) {
	err := b.OpenAccounts()
	if err != nil {
		return nil, err
	}

	acc := &multiaccounts.Account{
		KeyUID:                  keyUID,
		Name:                    request.DisplayName,
		CustomizationColor:      multiacccommon.CustomizationColor(request.CustomizationColor),
		CustomizationColorClock: customizationColorClock,
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

func (b *StatusBackend) prepareSettings(request *requests.CreateAccount, mnemonic string, keyUID string, masterAddress string,
	derivedAddresses map[string]generator.AccountInfo, restoreAccount bool) (*settings.Settings, error) {
	s, err := defaultSettings(keyUID, masterAddress, derivedAddresses)
	if err != nil {
		return nil, err
	}

	s.DeviceName = request.DeviceName
	s.DisplayName = request.DisplayName
	s.PreviewPrivacy = request.PreviewPrivacy
	s.CurrentNetwork = request.CurrentNetwork
	s.TestNetworksEnabled = request.TestNetworksEnabled
	s.AutoRefreshTokensEnabled = request.AutoRefreshTokensEnabled
	if !restoreAccount {
		s.Mnemonic = &mnemonic
		s.MnemonicWasNotShown = true
	}

	if request.WakuV2Fleet != "" {
		s.Fleet = &request.WakuV2Fleet
	}

	s.ThirdpartyServicesEnabled = request.ThirdpartyServicesEnabled

	return s, nil
}

func (b *StatusBackend) prepareConfig(request *requests.CreateAccount, keyUID string, installationID string) (*params.NodeConfig, error) {
	nodeConfig, err := DefaultNodeConfig(installationID, keyUID, request)
	if err != nil {
		return nil, err
	}

	return nodeConfig, nil
}

func (b *StatusBackend) prepareWalletAccount(request *requests.CreateAccount) *accsmanagementtypes.AccountCreationDetails {
	return &accsmanagementtypes.AccountCreationDetails{
		Path:    common.PathDefaultWalletAccount,
		Name:    walletAccountDefaultName,
		Emoji:   walletAccountDefaultEmoji,
		ColorID: request.CustomizationColor,
	}
}

func (b *StatusBackend) prepareKeypair(request *requests.CreateAccount, keyUID string, masterAddress string,
	derivedAddresses map[string]generator.AccountInfo, restoreAccount bool, walletXPub string,
	coldWallet accsmanagementtypes.ColdWalletType) (keypair *accsmanagementtypes.Keypair, err error) {
	// set up keypair
	keypair = &accsmanagementtypes.Keypair{
		Name:                    request.DisplayName,
		KeyUID:                  keyUID,
		Type:                    accsmanagementtypes.KeypairTypeProfile,
		DerivedFrom:             masterAddress,
		LastUsedDerivationIndex: 0,
		XPub:                    walletXPub,
		ColdWallet:              coldWallet,
	}

	// add chat account
	chatDerivedAccount := derivedAddresses[common.PathEIP1581Chat]
	keypair.Accounts = append(keypair.Accounts, &accsmanagementtypes.Account{
		PublicKey: types.Hex2Bytes(chatDerivedAccount.PublicKey),
		KeyUID:    keypair.KeyUID,
		Address:   types.HexToAddress(chatDerivedAccount.Address),
		Chat:      true,
		Path:      common.PathEIP1581Chat,
		Position:  -1, // When creating a new account, the chat account should have position -1, cause it doesn't participate
		Operable:  accsmanagementtypes.AccountFullyOperable,
	})

	// add wallet account
	walletDerivedAccount := derivedAddresses[common.PathDefaultWalletAccount]
	keypair.Accounts = append(keypair.Accounts, &accsmanagementtypes.Account{
		PublicKey:          types.Hex2Bytes(walletDerivedAccount.PublicKey),
		KeyUID:             keypair.KeyUID,
		Address:            types.HexToAddress(walletDerivedAccount.Address),
		ColorID:            multiacccommon.CustomizationColor(request.CustomizationColor),
		Emoji:              walletAccountDefaultEmoji,
		Wallet:             true,
		Path:               common.PathDefaultWalletAccount,
		Name:               walletAccountDefaultName,
		AddressWasNotShown: !restoreAccount,
		Position:           0, // When creating a new account, the wallet account should have position 0, cause it's the default wallet account
		Operable:           accsmanagementtypes.AccountFullyOperable,
	})

	return
}

func (b *StatusBackend) prepareForKeycard(request *requests.CreateAccount, multiAccount *multiaccounts.Account,
	nodeConfig *params.NodeConfig) error {
	if request.KeycardInstanceUID == "" {
		return nil
	}

	if request.KeycardPairingKey != "" {
		// KeycardPairingKey is used only on mobile
		multiAccount.KeycardPairing = request.KeycardPairingKey
	} else {
		// KeycardPairingDataFile is used only on desktop
		kp := wallet.NewKeycardPairings()
		kp.SetKeycardPairingsFile(nodeConfig.KeycardPairingDataFile)
		pairings, err := kp.GetPairings()
		if err != nil {
			return errors.Wrap(err, "failed to get keycard pairings")
		}

		keycard, ok := pairings[request.KeycardInstanceUID]
		if !ok {
			return errors.New("keycard not found in pairings file")
		}

		multiAccount.KeycardPairing = keycard.Key
	}

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

	err = b.multiaccountsDB.UpdateAccountKeycardPairing(generatedAccountInfo.KeyUID, "")
	if err != nil {
		return err
	}

	err = messenger.MigrateColdWalletKeypairToApp(context.Background(), mnemonicNoExtraSpaces, currPassword)
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
	return b.statusNode.AccountService().VerifyPassword(password)
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

func EnrichMultiAccountByPublicKey(account *multiaccounts.Account, chatPublicKey types.HexBytes) error {
	pk := string(chatPublicKey.Bytes())
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

func (b *StatusBackend) StartNodeWithChatKeyOrMnemonic(
	request *requests.CreateAccount,
	mnemonic string, // empty mnemonic is used for keycard account, not empty for regular account
	keycardData *requests.KeycardData, // nil for regular account, not nil for account with already set keycard
	restoreAccount bool,
) (*multiaccounts.Account, error) {
	// very important to update root data dir here
	b.UpdateRootDataDir(request.RootDataDir)

	var (
		isKeycard               = request.KeycardInstanceUID != ""
		keyUID                  string
		masterAddress           string
		chatPrivateKey          *ecdsa.PrivateKey // set only for keycard account
		chatPublicKey           types.HexBytes
		customizationColorClock uint64 // not sure if we need this customizationColorClock at all since the desktop app doesn't use it
		derivedAddresses        = map[string]generator.AccountInfo{
			common.PathWalletRoot:           {},
			common.PathEIP1581Root:          {},
			common.PathEIP1581Chat:          {},
			common.PathDefaultWalletAccount: {},
		}
		keypairToStoreDirectly *accsmanagementtypes.Keypair
		walletXPub             string
		coldWallet             accsmanagementtypes.ColdWalletType
	)

	if keycardData != nil { // means that the keycard is already set, details already on it
		keyUID = keycardData.KeyUID
		masterAddress = keycardData.Address
		walletXPub = keycardData.WalletXPub
		coldWallet = keycardData.ColdWallet

		derivedAddresses[common.PathWalletRoot] = generator.AccountInfo{
			AccountPublicInfo: generator.AccountPublicInfo{
				Address: keycardData.WalletRootAddress,
			},
		}
		derivedAddresses[common.PathEIP1581Root] = generator.AccountInfo{
			AccountPublicInfo: generator.AccountPublicInfo{
				Address: keycardData.Eip1581Address,
			},
		}
		derivedAddresses[common.PathEIP1581Chat] = generator.AccountInfo{
			AccountPublicInfo: generator.AccountPublicInfo{
				Address:   keycardData.WhisperAddress,
				PublicKey: keycardData.WhisperPublicKey,
			},
			PrivateKey: keycardData.WhisperPrivateKey,
		}
		derivedAddresses[common.PathDefaultWalletAccount] = generator.AccountInfo{
			AccountPublicInfo: generator.AccountPublicInfo{
				Address:   keycardData.WalletAddress,
				PublicKey: keycardData.WalletPublicKey,
			},
		}
		derivedAddresses[common.PathEIP1581Encryption] = generator.AccountInfo{
			AccountPublicInfo: generator.AccountPublicInfo{
				PublicKey: keycardData.EncryptionPublicKey,
			},
		}
	} else {
		genMasterAcc, err := generator.CreateAccountFromMnemonic(mnemonic, "")
		if err != nil {
			return nil, err
		}

		keyUID = genMasterAcc.KeyUID()
		masterAddress = genMasterAcc.Address().Hex()

		if !restoreAccount {
			customizationColorClock = 1
		}

		derivationPaths := []string{
			common.PathWalletRoot,
			common.PathEIP1581Root,
			common.PathEIP1581Chat,
			common.PathDefaultWalletAccount,
			common.PathEIP1581Encryption,
		}
		_, generatedDerivedAddresses, err := b.generateDerivedAddresses(genMasterAcc, derivationPaths)
		if err != nil {
			return nil, err
		}
		derivedAddresses = generatedDerivedAddresses

		walletXPub, err = generator.DeriveExtendedPublicKeyAtPath(mnemonic, "", common.PathWalletXPub)
		if err != nil {
			return nil, err
		}
	}

	if isKeycard {
		genChatAccount, err := generator.CreateAccountFromPrivateKey(derivedAddresses[common.PathEIP1581Chat].PrivateKey)
		if err != nil {
			return nil, err
		}

		coldWallet = accsmanagementtypes.ColdWalletTypeStatusKeycard
		chatPrivateKey = genChatAccount.PrivateKey()
		chatPublicKey = types.Hex2Bytes(genChatAccount.PublicKeyHex())

		request.Password = derivedAddresses[common.PathEIP1581Encryption].PublicKey
	} else {
		chatPublicKey = types.Hex2Bytes(derivedAddresses[common.PathEIP1581Chat].PublicKey)
	}

	settings, err := b.prepareSettings(request, mnemonic, keyUID, masterAddress, derivedAddresses, restoreAccount)
	if err != nil {
		return nil, errors.Wrap(err, "failed to prepare settings")
	}

	multiAccount, err := b.buildAccount(request, keyUID, customizationColorClock)
	if err != nil {
		return nil, err
	}

	if multiAccount.Name == "" {
		multiAccount.Name = settings.Name
	}

	err = EnrichMultiAccountByPublicKey(multiAccount, chatPublicKey)
	if err != nil {
		return nil, err
	}

	nodeConfig, err := b.prepareConfig(request, keyUID, settings.InstallationID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to prepare node config")
	}

	err = b.ensureDBsOpened(*multiAccount, request.Password)
	if err != nil {
		return nil, err
	}

	if isKeycard {
		err = b.prepareForKeycard(request, multiAccount, nodeConfig)
		if err != nil {
			return nil, errors.Wrap(err, "failed to prepare for keycard")
		}

		keypairToStoreDirectly, err = b.prepareKeypair(request, keyUID, masterAddress, derivedAddresses, restoreAccount,
			walletXPub, coldWallet)
		if err != nil {
			return nil, errors.Wrap(err, "failed to prepare keypair")
		}
	} else {
		walletAccount := b.prepareWalletAccount(request)
		_, err := b.accountsManager.CreateKeypairFromMnemonicAndStore(mnemonic, request.Password, request.DisplayName,
			coldWallet, walletAccount, true, 0)
		if err != nil {
			return nil, err
		}
	}

	err = b.StartNodeWithAccountAndInitialConfig(
		multiAccount,
		request.Password,
		*settings,
		nodeConfig,
		keypairToStoreDirectly,
		chatPrivateKey,
	)

	return multiAccount, err
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

func (b *StatusBackend) loadNodeConfig(inputNodeCfg *params.NodeConfig) error {
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

	conf.RootDataDir = b.rootDataDir

	if _, err = os.Stat(conf.RootDataDir); os.IsNotExist(err) {
		if err := os.MkdirAll(conf.RootDataDir, os.ModePerm); err != nil {
			b.logger.Warn("failed to create data directory", zap.Error(err))
			return err
		}
	}

	b.config = conf

	if inputNodeCfg != nil && inputNodeCfg.RuntimeLogLevel != "" {
		b.config.LogLevel = inputNodeCfg.RuntimeLogLevel
	}

	return nil
}

func (b *StatusBackend) saveNodeConfig(n *params.NodeConfig) error {
	err := nodecfg.SaveNodeConfig(b.appDB, n)
	if err != nil {
		return err
	}
	return nil
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

	if err = b.statusNode.Start(config); err != nil {
		return
	}

	b.transactor.SetEthClientGetter(b.statusNode.RPCClient(), rpc.DefaultCallTimeout)

	signal.SendNodeStarted()

	if b.statusNode.WalletService() != nil {
		b.statusNode.WalletService().KeycardPairings().SetKeycardPairingsFile(config.KeycardPairingDataFile)
	}

	if b.prometheusMetrics != nil {
		b.prometheusMetrics.RegisterHandler("waku", b.wakuMetricsHandler())
	}

	if config.OTELConfig.Enabled {
		b.logger.Info("initializing OpenTelemetry tracer provider",
			zap.String("endpoint", config.OTELConfig.Endpoint),
			zap.Bool("insecure", config.OTELConfig.Insecure),
		)

		shutdownTracer, err := trace.InitProvider(context.Background(), trace.Config{
			ServiceName:  "status-go",
			OTLPEndpoint: config.OTELConfig.Endpoint,
			Insecure:     config.OTELConfig.Insecure,
		})
		if err != nil {
			return err
		}

		b.shutdownTasks = append(b.shutdownTasks, func() error { return shutdownTracer(context.Background()) })
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
	if b.statusNode == nil || !b.IsNodeRunning() {
		return nil
	}
	if !b.LocalPairingStateManager.IsPairing() {
		defer signal.SendNodeStopped()
	}

	for _, task := range b.shutdownTasks {
		if err := task(); err != nil {
			b.logger.Error("shutdown task failed", zap.Error(err))
		}
	}
	b.shutdownTasks = []func() error{}

	return b.statusNode.Stop()
}

// RestartNode restart running Status node, fails if node is not running
func (b *StatusBackend) RestartNode() error {
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

// CallRPC executes public RPC requests on node's in-proc RPC server.
func (b *StatusBackend) CallInProcessRPC(inputJSON string) string {
	return b.statusNode.CallInProcessRPC(inputJSON)
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

// SignTypedData accepts data and password. Gets verified account and signs typed data.
func (b *StatusBackend) SignTypedData(typed typeddata.TypedData, address string, password string) (types.HexBytes, error) {
	acc, err := b.getVerifiedWalletAccount(address, password)
	if err != nil {
		return types.HexBytes{}, err
	}
	chain := new(big.Int).SetUint64(b.StatusNode().Config().NetworkID)
	sig, err := typeddata.Sign(typed, acc.PrivateKey(), chain)
	if err != nil {
		return types.HexBytes{}, err
	}
	return sig, err
}

// SignTypedDataV4 accepts data and password. Gets verified account and signs typed data.
func (b *StatusBackend) SignTypedDataV4(typed signercore.TypedData, address string, password string) (types.HexBytes, error) {
	acc, err := b.getVerifiedWalletAccount(address, password)
	if err != nil {
		return types.HexBytes{}, err
	}
	chain := new(big.Int).SetUint64(b.StatusNode().Config().NetworkID)
	sig, err := typeddata.SignTypedDataV4(typed, acc.PrivateKey(), chain)
	if err != nil {
		return types.HexBytes{}, err
	}
	return types.HexBytes(sig), err
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

// PausableServices returns the list of services that can be individually paused/resumed.
func (b *StatusBackend) PausableServices() []node.PausableServiceInfo {
	if b.statusNode == nil {
		return nil
	}
	return b.statusNode.ServiceRegistry().ListPausable()
}

// PauseService pauses a single named service.
func (b *StatusBackend) PauseService(name string) error {
	if b.statusNode == nil {
		return nil
	}
	return b.statusNode.ServiceRegistry().Pause(name)
}

// ResumeService resumes a single named service.
func (b *StatusBackend) ResumeService(name string) error {
	if b.statusNode == nil {
		return nil
	}
	return b.statusNode.ServiceRegistry().Resume(name)
}

// PauseServices pauses a list of named services, collecting any errors.
func (b *StatusBackend) PauseServices(names []string) error {
	if b.statusNode == nil {
		return nil
	}

	if rpcClient := b.statusNode.RPCClient(); rpcClient != nil {
		rpcClient.SetPaused(true)
	}

	return b.statusNode.ServiceRegistry().PauseMultiple(names)
}

// ResumeServices resumes a list of named services, collecting any errors.
func (b *StatusBackend) ResumeServices(names []string) error {
	if b.statusNode == nil {
		return nil
	}

	if rpcClient := b.statusNode.RPCClient(); rpcClient != nil {
		rpcClient.SetPaused(false)
	}

	return b.statusNode.ServiceRegistry().ResumeMultiple(names)
}

func (b *StatusBackend) StopLocalNotifications() error {
	if b.statusNode == nil {
		return nil
	}
	return b.statusNode.StopLocalNotifications()
}

func (b *StatusBackend) StartLocalNotifications() error {
	if b.statusNode == nil {
		return nil
	}
	return b.statusNode.StartLocalNotifications()

}

// Logout clears whisper identities.
func (b *StatusBackend) Logout() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.logger.Debug("logging out")

	b.AccountsManager().Logout()
	b.account = nil
	b.clearProfileSecretCache()

	if b.statusNode != nil {
		if b.statusNode.IsRunning() {
			if err := b.statusNode.Stop(); err != nil {
				return err
			}
		}
		if err := b.statusNode.StopMediaServer(); err != nil {
			return err
		}
		b.statusNode = nil
	}

	err := b.closeDBs()
	if err != nil {
		return err
	}

	if !b.LocalPairingStateManager.IsPairing() {
		signal.SendNodeStopped()
	}

	if err = b.switchToPreLoginLog(); err != nil {
		return err
	}

	// re-initialize the node, at some point we should better manage the lifecycle
	if err = b.initialize(); err != nil {
		return err
	}

	err = b.statusNode.StartMediaServerWithoutDB()
	if err != nil {
		b.logger.Error("failed to start media server without app db", zap.Error(err))
		return err
	}

	return nil
}

// switchToPreLoginLog switches to global pre-login logging settings.
// This log is profile-independent and should be enabled by default,
// including in release builds, to help diagnose login issues.
// related issue: https://github.com/status-im/status-mobile/issues/21501
func (b *StatusBackend) switchToPreLoginLog() error {
	_ = logutils.ZapLogger().Sync()
	return logutils.OverrideRootLoggerWithConfig(b.preLoginLogConfig.ConvertToLogSettings())
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

	err := b.accountsManager.SetChatAccountWithProfileKeypair(loginParams.ChatAddress, loginParams.Password, privateKey, loginParams.ProfileKeypair)
	if err != nil {
		return err
	}

	if loginParams.MultiAccount != nil {
		b.account = loginParams.MultiAccount
	}

	if err := b.initProtocol(); err != nil {
		return err
	}

	if err = b.statusNode.StartLocalBackup(); err != nil {
		return err
	}

	return nil
}

func (b *StatusBackend) GetActiveAccount() (*multiaccounts.Account, error) {
	if b.account == nil {
		return nil, errors.New("master key account is nil in the StatusBackend")
	}

	return b.account, nil
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

func (b *StatusBackend) initProtocol() error {
	st := b.statusNode.WakuV2ExtService()
	if st == nil {
		return nil
	}
	chatAccount, err := b.accountsManager.SelectedChatAccount()
	if err != nil {
		return err
	}
	identity := chatAccount.PrivateKey()
	acc, err := b.GetActiveAccount()
	if err != nil {
		return err
	}
	params := ext.InitProtocolParams{
		Identity:               identity,
		AppDB:                  b.appDB,
		WalletDB:               b.walletDB,
		HTTPServer:             b.statusNode.MediaServer(),
		MultiAccountDB:         b.multiaccountsDB,
		Account:                acc,
		AccountsManager:        b.accountsManager,
		RPCClient:              b.statusNode.RPCClient(),
		WalletService:          b.statusNode.WalletService(),
		CommunityTokensService: b.statusNode.CommunityTokensService(),
		AccountsPublisher:      b.statusNode.AccountsPublisher(),
		TimeSource:             b.statusNode.TimeSource(),
		MetricsEnabled:         b.prometheusMetrics != nil,
		TokenManager:           NewCommunitiesTokenManager(b.statusNode.TokenManager()),
		TokenBalanceManager:    NewCommunitiesTokenBalanceManager(b.statusNode.TokenBalancesFetcher(), b.statusNode.TokenBalancesStorage()),
		NetworkManager:         NewCommunitiesNetworkManager(b.statusNode.RPCClient().GetNetworkManager()),
	}
	err = st.InitProtocol(params)
	if err != nil {
		return err
	}

	messenger := st.Messenger()
	b.statusNode.AccountService().Init(messenger, acc)
	// Init chat service
	accDB, err := accounts.NewDB(b.appDB)
	if err != nil {
		return err
	}
	b.statusNode.ChatService(accDB).Init(messenger)
	b.statusNode.EnsService().Init(messenger.SyncEnsNamesWithDispatchMessage)
	b.statusNode.CommunityTokensService().Init(messenger)
	b.statusNode.SharedUrlsService().SetDataProvider(nodeadapters.NewSharedUrlsMessengerAdapter(messenger))
	if b.statusNode.NewsFeedService() != nil {
		b.statusNode.NewsFeedService().SetActivityCenter(nodeadapters.NewNewsFeedActivityCenterAdapter(messenger))
	}
	b.statusNode.LinkPreviewService().SetStatusDataProvider(nodeadapters.NewLinkPreviewMessengerAdapter(messenger))

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

func (b *StatusBackend) getAppDBPath(keyUID string) (string, error) {
	if len(b.rootDataDir) == 0 {
		return "", errors.New("root datadir wasn't provided")
	}

	return filepath.Join(b.rootDataDir, fmt.Sprintf("%s-v4.db", keyUID)), nil
}

func (b *StatusBackend) getWalletDBPath(keyUID string) (string, error) {
	if len(b.rootDataDir) == 0 {
		return "", errors.New("root datadir wasn't provided")
	}

	return filepath.Join(b.rootDataDir, fmt.Sprintf("%s-wallet.db", keyUID)), nil
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

func (b *StatusBackend) SetProfileLogMaxBackups(count uint) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	err := nodecfg.SetMaxLogBackups(b.appDB, count)
	if err != nil {
		return err
	}
	b.config.LogMaxBackups = int(count)

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
			b.logger.Error("failed to get waku metrics: StatusNode is nil")
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
