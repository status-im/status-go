package backend

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"sync/atomic"
	"time"

	accounts2 "github.com/ethereum/go-ethereum/accounts"
	errorspkg "github.com/pkg/errors"

	"go.uber.org/zap"

	"github.com/ethereum/go-ethereum/event"
	gethrpc "github.com/ethereum/go-ethereum/rpc"

	accsmanagement "github.com/status-im/status-go/accounts-management"
	common2 "github.com/status-im/status-go/common"
	"github.com/status-im/status-go/connection"
	"github.com/status-im/status-go/crypto"
	"github.com/status-im/status-go/internal/timesource"
	"github.com/status-im/status-go/ipfs"
	"github.com/status-im/status-go/multiaccounts"
	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/node/backup"
	"github.com/status-im/status-go/params"
	rpc2 "github.com/status-im/status-go/pkg/backend/rpc"
	"github.com/status-im/status-go/pkg/pubsub"
	"github.com/status-im/status-go/rpc"
	"github.com/status-im/status-go/server"
	accountssvc "github.com/status-im/status-go/services/accounts"
	appgeneral "github.com/status-im/status-go/services/app-general"
	"github.com/status-im/status-go/services/browsers"
	"github.com/status-im/status-go/services/chat"
	"github.com/status-im/status-go/services/communitytokens"
	"github.com/status-im/status-go/services/connector"
	"github.com/status-im/status-go/services/ens"
	"github.com/status-im/status-go/services/eth"
	"github.com/status-im/status-go/services/gif"
	localnotifications "github.com/status-im/status-go/services/local-notifications"
	"github.com/status-im/status-go/services/newsfeed"
	"github.com/status-im/status-go/services/permissions"
	"github.com/status-im/status-go/services/personal"
	"github.com/status-im/status-go/services/rpcstats"
	"github.com/status-im/status-go/services/sharedurls"
	"github.com/status-im/status-go/services/status"
	"github.com/status-im/status-go/services/stickers"
	"github.com/status-im/status-go/services/updates"
	"github.com/status-im/status-go/services/utils"
	"github.com/status-im/status-go/services/wakuv2ext"
	"github.com/status-im/status-go/services/wallet"
	"github.com/status-im/status-go/services/wallet/community"
	"github.com/status-im/status-go/services/wallet/pendingtxtracker"
	"github.com/status-im/status-go/services/wallet/token"
	"github.com/status-im/status-go/services/wallet/tokenbalances"
	"github.com/status-im/status-go/transactions"
)

// FIXME: This is temporal and will be replaced with the actual setting
const privateMode = false

// Services abstracts contained geth node and provides helper methods to
// interact with it.
type Services struct {
	mu sync.RWMutex

	backend   *StatusBackend
	rpcServer *gethrpc.Server

	// FIXME: Old implementation below:

	//appDB           *sql.DB
	multiaccountsDB *multiaccounts.Database
	//walletDB        *sql.DB

	running atomic.Bool

	config    *params.NodeConfig // Status node configuration
	rpcClient *rpc.Client        // reference to an RPC client

	services []common2.StatusService

	tokenManager *token.Manager

	logger *zap.Logger

	gethAccountsManager *accsmanagement.AccountsManager
	transactor          *transactions.Transactor

	publicMethods map[string]bool
	// we explicitly list every service, we could use interfaces
	// and store them in a nicer way and user reflection, but for now stupid is good
	rpcStatsSrvc           *rpcstats.Service
	statusPublicSrvc       *status.Service
	accountsSrvc           *accountssvc.Service
	browsersSrvc           *browsers.Service
	permissionsSrvc        *permissions.Service
	walletSrvc             *wallet.Service
	localNotificationsSrvc *localnotifications.Service
	personalSrvc           *personal.Service
	timeSourceSrvc         timesource.Service
	wakuV2ExtSrvc          *wakuv2ext.Service
	ensSrvc                *ens.Service
	communityTokensSrvc    *communitytokens.Service
	gifSrvc                *gif.Service
	stickersSrvc           *stickers.Service
	chatSrvc               *chat.Service
	updatesSrvc            *updates.Service
	pendingTracker         *pendingtxtracker.PendingTxTracker
	connectorSrvc          *connector.Service
	appGeneralSrvc         *appgeneral.Service
	ethSrvc                *eth.Service
	newsfeedSrvc           *newsfeed.Service
	sharedUrlsSrvc         *sharedurls.Service

	walletFeed        event.Feed
	accountsPublisher *pubsub.Publisher

	localBackup *backup.Controller
}

func NewServices(backend *StatusBackend, logger *zap.Logger) *Services {
	return &Services{
		backend:   backend,
		logger:    logger,
		rpcServer: gethrpc.NewServer(),
	}
}

//// NewServices makes new instance of Services.
//func NewServices(transactor *transactions.Transactor, gethAccountsManager *accsmanagement.AccountsManager, logger *zap.Logger) *Services {
//	logger = logger.Named("Services")
//	return &Services{
//		transactor:          transactor,
//		gethAccountsManager: gethAccountsManager,
//		logger:              logger,
//		publicMethods:       make(map[string]bool),
//		accountsPublisher:   pubsub.NewPublisher(),
//		rpcServer:           gethrpc.NewServer(),
//	}
//}

func (b *Services) RegisterName(name string, receiver interface{}) error {
	return b.rpcServer.RegisterName(name, receiver)
}

// Config exposes reference to running node's configuration
func (b *Services) Config() *params.NodeConfig {
	b.mu.RLock()
	defer b.mu.RUnlock()

	return b.config
}

// StartWithOptions starts current Services, failing if it's already started.
// It takes some options that allows to further configure starting process.
func (b *Services) Start(config *params.NodeConfig) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if !b.running.CompareAndSwap(false, true) {
		b.logger.Debug("node is already running")
		return ErrNodeRunning
	}

	b.logger.Debug("starting with options", zap.Stringer("ClusterConfig", &config.ClusterConfig))

	return b.startWithDB(config)
}

func (b *Services) StartLocalBackup() error {
	if b.localBackup != nil {
		return errors.New("local backup already started")
	}

	backupPath, err := b.accountsSrvc.GetBackupPath()
	if err != nil {
		return err
	}
	if backupPath == "" {
		// No path set yet, set it to the user's config directory
		dir, err := os.UserConfigDir()
		// We do not return the error as it's not a major issue
		if err != nil {
			b.logger.Error("failed to get user config dir", zap.Error(err))
		} else {
			err = b.accountsSrvc.SetBackupPath(filepath.Join(dir, "Status", "backups"))
			if err != nil {
				b.logger.Error("failed to set backup path", zap.Error(err))
			}
		}
	}

	chatAccount, err := b.gethAccountsManager.SelectedChatAccount()
	if err != nil {
		return err
	}

	privateKey := chatAccount.PrivateKey()

	filenameGetter := func() (string, error) {
		backupPath, err := b.accountsSrvc.GetBackupPath()
		if err != nil {
			return "", err
		}

		compressedPubKey, err := utils.SerializePublicKey(crypto.CompressPubkey(&privateKey.PublicKey))
		if err != nil {
			return "", err
		}

		var backupDir string
		if backupPath != "" {
			backupDir = backupPath
		} else {
			backupDir = filepath.Join(b.config.RootDataDir, "backups")
		}

		fullPath := filepath.Join(backupDir, fmt.Sprintf("%s_user_data.bkp", compressedPubKey[len(compressedPubKey)-6:]))

		return fullPath, nil
	}

	b.localBackup, err = backup.NewController(backup.BackupConfig{
		PrivateKey:     crypto.Keccak256(crypto.FromECDSA(privateKey)),
		FileNameGetter: filenameGetter,
		BackupEnabled:  true,
		Interval:       time.Minute * 30,
	}, b.logger.Named("LocalBackup"))
	if err != nil {
		return err
	}

	if b.accountsSrvc != nil {
		b.localBackup.Register("settings", b.accountsSrvc)
	}

	if b.walletSrvc != nil {
		b.localBackup.Register("wallet", b.walletSrvc)
	}

	if b.statusPublicSrvc != nil {
		b.localBackup.Register("messenger", b.statusPublicSrvc.Messenger())
	}

	b.localBackup.Start()

	return nil
}

func (b *Services) PerformLocalBackup() (string, error) {
	return b.localBackup.PerformBackup()
}

func (b *Services) LoadLocalBackup(filePath string) error {
	return b.localBackup.LoadBackup(filePath)
}

func (b *Services) startWithDB(config *params.NodeConfig) error {
	b.config = config

	if err := b.setupRPCClient(); err != nil {
		return err
	}

	if err := b.createAndStartTokenManager(); err != nil {
		return err
	}

	if err := b.initServices(config, b.backend.mediaServer); err != nil {
		return err
	}

	// Run migrations
	err := b.runServicesMigrations()
	if err != nil {
		return errorspkg.Wrap(err, "failed to run services migrations")
	}

	// Register services

	for _, service := range b.services {
		err := b.registerService(service)
		if err != nil {
			name := reflect.TypeOf(service).Name()
			text := fmt.Sprintf("failed to register service '%s'", name)
			return errorspkg.Wrap(err, text)
		}
	}

	// Start services

	err = b.timeSourceSrvc.Start(context.Background())
	if err != nil {
		return errorspkg.Wrap(err, "failed to start time source")
	}

	for _, service := range b.services {
		err := service.Start()
		if err != nil {
			name := reflect.TypeOf(service).Name()
			text := fmt.Sprintf("failed to start service '%s'", name)
			return errorspkg.Wrap(err, text)
		}
	}

	return nil
}

func (b *Services) createAndStartTokenManager() error {
	accDB, err := accounts.NewDB(b.appDB)
	if err != nil {
		return err
	}

	b.tokenManager = token.NewTokenManager(b.walletDB, b.rpcClient, community.NewManager(b.appDB, b.mediaServer, nil),
		b.rpcClient.GetNetworkManager(), b.appDB, b.mediaServer, &b.walletFeed, b.accountsPublisher, accDB,
		token.NewPersistence(b.walletDB))

	const (
		defaultAutoRefreshInterval      = 30 * time.Minute // interval after which we should fetch the token lists from the remote source (or use the default one if remote source is not set)
		defaultAutoRefreshCheckInterval = 3 * time.Minute  // interval after which we should check if we should trigger the auto-refresh
	)

	autoRefreshInterval := defaultAutoRefreshInterval
	autoRefreshCheckInterval := defaultAutoRefreshCheckInterval
	if b.config.WalletConfig.TokensListsAutoRefreshInterval > 0 &&
		b.config.WalletConfig.TokensListsAutoRefreshCheckInterval > 0 &&
		b.config.WalletConfig.TokensListsAutoRefreshInterval > b.config.WalletConfig.TokensListsAutoRefreshCheckInterval {
		autoRefreshInterval = time.Duration(b.config.WalletConfig.TokensListsAutoRefreshInterval) * time.Second
		autoRefreshCheckInterval = time.Duration(b.config.WalletConfig.TokensListsAutoRefreshCheckInterval) * time.Second
	}

	b.tokenManager.Start(context.Background(), autoRefreshInterval, autoRefreshCheckInterval)
	return nil
}

func (b *Services) setupRPCClient() (err error) {
	config := rpc.ClientConfig{
		Networks:          b.config.Networks,
		DB:                b.appDB,
		AccountsPublisher: b.accountsPublisher,
	}
	b.rpcClient, err = rpc.NewClient(config)
	if err != nil {
		return
	}
	b.rpcClient.Start(context.Background())
	return
}

// Stop will stop current Services. A stopped node cannot be resumed.
func (b *Services) Stop() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.logger.Debug("stopping")

	if !b.running.CompareAndSwap(true, false) {
		return ErrNoRunningNode
	}

	var errs []error
	b.timeSourceSrvc.Stop()

	for _, service := range b.services {
		err := service.Stop()
		errs = append(errs, err)
	}

	if b.localBackup != nil {
		b.localBackup.Stop()
		b.localBackup = nil
	}

	b.accountsPublisher.Close()

	b.rpcClient.Stop()
	b.rpcClient = nil
	b.config = nil

	b.mediaServer.SetDataProviders(nil, nil, nil)

	b.downloader.Stop()
	b.downloader = nil

	b.rpcStatsSrvc = nil
	b.accountsSrvc = nil
	b.browsersSrvc = nil
	b.permissionsSrvc = nil
	b.walletSrvc = nil
	b.localNotificationsSrvc = nil
	b.personalSrvc = nil
	b.timeSourceSrvc = nil
	b.wakuV2ExtSrvc = nil
	b.ensSrvc = nil
	b.communityTokensSrvc = nil
	b.stickersSrvc = nil
	b.connectorSrvc = nil
	b.publicMethods = make(map[string]bool)
	b.pendingTracker = nil
	b.appGeneralSrvc = nil
	b.newsfeedSrvc = nil

	b.logger.Debug("status node stopped")
	return errors.Join(errs...)
}

// IsRunning confirm that node is running.
func (b *Services) IsRunning() bool {
	return b.running.Load()
}

func (b *Services) ConnectionChanged(state connection.State) {
	if b.wakuV2ExtSrvc != nil {
		b.wakuV2ExtSrvc.ConnectionChanged(state)
	}
}

func (b *Services) CallInProcessRPC(inputJSON string) string {
	codec := rpc2.NewSingleRequestCodec(inputJSON)
	b.rpcServer.ServeCodec(codec.GethCodec(), 0)
	return codec.Output()
}

// RPCClient exposes reference to RPC client connected to the running node.
func (b *Services) RPCClient() *rpc.Client {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.rpcClient
}

func (b *Services) SetAppDB(db *sql.DB) {
	b.appDB = db
}

func (b *Services) GetAppDB() *sql.DB {
	return b.appDB
}

func (b *Services) SetMultiaccountsDB(db *multiaccounts.Database) {
	b.multiaccountsDB = db
}

func (b *Services) SetWalletDB(db *sql.DB) {
	b.walletDB = db
}

func (b *Services) GetWalletDB() *sql.DB {
	return b.walletDB
}

func (b *Services) TokenManager() *token.Manager {
	return b.tokenManager
}

func (b *Services) TokenBalancesFetcher() *tokenbalances.Fetcher {
	if b.walletSrvc != nil {
		b.walletSrvc.GetTokenBalancesFetcher()
	}
	return nil
}

func (b *Services) TokenBalancesStorage() tokenbalances.Storage {
	if b.walletSrvc != nil {
		return b.walletSrvc.GetTokenBalancesStorage()
	}
	return nil
}
