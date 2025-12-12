package node

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

	errorspkg "github.com/pkg/errors"

	"go.uber.org/zap"

	"github.com/ethereum/go-ethereum/event"
	gethrpc "github.com/ethereum/go-ethereum/rpc"

	accsmanagement "github.com/status-im/status-go/accounts-management"
	common2 "github.com/status-im/status-go/common"
	"github.com/status-im/status-go/crypto"
	"github.com/status-im/status-go/internal/db/multiaccounts"
	"github.com/status-im/status-go/internal/db/multiaccounts/accounts"
	"github.com/status-im/status-go/internal/timesource"
	"github.com/status-im/status-go/internal/transactions"
	"github.com/status-im/status-go/ipfs"
	"github.com/status-im/status-go/params"
	rpc2 "github.com/status-im/status-go/pkg/backend/node/rpc"
	"github.com/status-im/status-go/pkg/pubsub"
	"github.com/status-im/status-go/rpc"
	"github.com/status-im/status-go/server"
	accountssvc "github.com/status-im/status-go/services/accounts"
	appgeneral "github.com/status-im/status-go/services/app-general"
	"github.com/status-im/status-go/services/backup"
	"github.com/status-im/status-go/services/browsers"
	"github.com/status-im/status-go/services/chat"
	"github.com/status-im/status-go/services/communitytokens"
	"github.com/status-im/status-go/services/connector"
	"github.com/status-im/status-go/services/ens"
	"github.com/status-im/status-go/services/eth"
	"github.com/status-im/status-go/services/gif"
	"github.com/status-im/status-go/services/linkpreview"
	localnotifications "github.com/status-im/status-go/services/local-notifications"
	"github.com/status-im/status-go/services/newsfeed"
	"github.com/status-im/status-go/services/permissions"
	"github.com/status-im/status-go/services/personal"
	"github.com/status-im/status-go/services/rpcstats"
	"github.com/status-im/status-go/services/sharedurls"
	"github.com/status-im/status-go/services/stickers"
	"github.com/status-im/status-go/services/updates"
	"github.com/status-im/status-go/services/utils"
	"github.com/status-im/status-go/services/wakuv2ext"
	"github.com/status-im/status-go/services/wallet"
	"github.com/status-im/status-go/services/wallet/community"
	"github.com/status-im/status-go/services/wallet/pendingtxtracker"
	"github.com/status-im/status-go/services/wallet/token"
	"github.com/status-im/status-go/services/wallet/tokenbalances"
)

// errors
var (
	ErrNodeRunning   = errors.New("node is already running")
	ErrNoRunningNode = errors.New("there is no running node")
)

// FIXME: This is temporal and will be replaced with the actual setting
const privateMode = false

// StatusNode abstracts contained geth node and provides helper methods to
// interact with it.
type StatusNode struct {
	mu sync.RWMutex

	appDB           *sql.DB
	multiaccountsDB *multiaccounts.Database
	walletDB        *sql.DB

	running atomic.Bool

	config    *params.NodeConfig // Status node configuration
	rpcClient *rpc.Client        // reference to an RPC client

	services  []common2.StatusService
	rpcServer *gethrpc.Server

	downloader *ipfs.Downloader

	mediaServerAddress       *string
	mediaServerAdvertizeHost string
	mediaServerAdvertizePort int
	mediaServerEnableTLS     *bool
	mediaServer              *server.MediaServer

	tokenManager *token.Manager

	logger *zap.Logger

	gethAccountsManager *accsmanagement.AccountsManager
	transactor          *transactions.Transactor

	publicMethods map[string]bool
	// we explicitly list every service, we could use interfaces
	// and store them in a nicer way and user reflection, but for now stupid is good
	rpcStatsSrvc           *rpcstats.Service
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
	linkPreviewSrvc        *linkpreview.Service

	walletFeed        event.Feed
	accountsPublisher *pubsub.Publisher

	localBackup *backup.Controller
}

// New makes new instance of StatusNode.
func New(transactor *transactions.Transactor, gethAccountsManager *accsmanagement.AccountsManager, logger *zap.Logger) *StatusNode {
	logger = logger.Named("StatusNode")
	return &StatusNode{
		transactor:          transactor,
		gethAccountsManager: gethAccountsManager,
		logger:              logger,
		publicMethods:       make(map[string]bool),
		accountsPublisher:   pubsub.NewPublisher(),
		rpcServer:           gethrpc.NewServer(),
	}
}

// Config exposes reference to running node's configuration
func (n *StatusNode) Config() *params.NodeConfig {
	n.mu.RLock()
	defer n.mu.RUnlock()

	return n.config
}

func (n *StatusNode) MediaServer() *server.MediaServer {
	n.mu.RLock()
	defer n.mu.RUnlock()

	return n.mediaServer
}

func (n *StatusNode) startMediaServer() error {
	if n.mediaServer != nil {
		if err := n.mediaServer.Stop(); err != nil {
			return err
		}
	}

	var opts []server.MediaServerOption
	if n.mediaServerEnableTLS != nil {
		opts = append(opts, server.WithMediaServerDisableTLS(!*n.mediaServerEnableTLS))
	}
	if n.mediaServerAddress != nil {
		opts = append(opts, server.WithMediaServerAddress(*n.mediaServerAddress))
	}
	opts = append(opts, server.WithMediaServerAdvertizeAddress(n.mediaServerAdvertizeHost, n.mediaServerAdvertizePort))
	mediaServer, err := server.NewMediaServer(nil, nil, n.multiaccountsDB, nil, opts...)
	if err != nil {
		return err
	}

	n.mediaServer = mediaServer

	if err := n.mediaServer.Start(); err != nil {
		return err
	}

	return nil
}

// StartMediaServerWithoutDB starts media server without starting the node
// The server can only handle requests that don't require appdb or IPFS downloader
func (n *StatusNode) StartMediaServerWithoutDB() error {
	if n.IsRunning() {
		n.logger.Debug("node is already running, no need to StartMediaServerWithoutDB")
		return nil
	}

	return n.startMediaServer()
}

// StartWithOptions starts current StatusNode, failing if it's already started.
// It takes some options that allows to further configure starting process.
func (n *StatusNode) Start(config *params.NodeConfig) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if !n.running.CompareAndSwap(false, true) {
		n.logger.Debug("node is already running")
		return ErrNodeRunning
	}

	n.logger.Debug("starting with options", zap.Stringer("ClusterConfig", &config.ClusterConfig))

	return n.startWithDB(config)
}

func (n *StatusNode) StartLocalBackup() error {
	if n.localBackup != nil {
		return errors.New("local backup already started")
	}

	backupPath, err := n.accountsSrvc.GetBackupPath()
	if err != nil {
		return err
	}
	if backupPath == "" {
		// No path set yet, set it to the user's config directory
		dir, err := os.UserConfigDir()
		// We do not return the error as it's not a major issue
		if err != nil {
			n.logger.Error("failed to get user config dir", zap.Error(err))
		} else {
			err = n.accountsSrvc.SetBackupPath(filepath.Join(dir, "Status", "backups"))
			if err != nil {
				n.logger.Error("failed to set backup path", zap.Error(err))
			}
		}
	}

	chatAccount, err := n.gethAccountsManager.SelectedChatAccount()
	if err != nil {
		return err
	}

	privateKey := chatAccount.PrivateKey()

	n.localBackup, err = backup.NewController(backup.Config{
		PrivateKey:       crypto.Keccak256(crypto.FromECDSA(privateKey)),
		FileNameProvider: n,
		BackupEnabled:    true,
		Interval:         time.Minute * 30,
	}, n.logger.Named("LocalBackup"))
	if err != nil {
		return err
	}

	if n.accountsSrvc != nil {
		n.localBackup.Register("settings", n.accountsSrvc)
	}

	if n.walletSrvc != nil {
		n.localBackup.Register("wallet", n.walletSrvc)
	}

	if n.wakuV2ExtSrvc != nil {
		n.localBackup.Register("messenger", n.wakuV2ExtSrvc.Messenger())
	}

	n.localBackup.Start()

	return nil
}

func (n *StatusNode) GetBackupFilename() (string, error) {
	chatAccount, err := n.gethAccountsManager.SelectedChatAccount()
	if err != nil {
		return "", err
	}

	privateKey := chatAccount.PrivateKey()

	backupPath, err := n.accountsSrvc.GetBackupPath()
	if err != nil {
		return "", err
	}

	compressedPubKey, err := utils.SerializePublicKey(crypto.CompressPubkey(&privateKey.PublicKey))
	if err != nil {
		return "", err
	}

	if backupPath == "" {
		return "", errors.New("backup path is not set")
	}

	fullPath := filepath.Join(backupPath, fmt.Sprintf("%s_user_data.bkp", compressedPubKey[len(compressedPubKey)-6:]))

	return fullPath, nil
}

func (n *StatusNode) PerformLocalBackup() (string, error) {
	return n.localBackup.PerformBackup()
}

func (n *StatusNode) LoadLocalBackup(filePath string) error {
	return n.localBackup.LoadBackup(filePath)
}

func (n *StatusNode) SetMediaServerOptions(address *string, enableTLS *bool, advertizeHost string, advertizePort int) {
	n.mediaServerAddress = address
	n.mediaServerEnableTLS = enableTLS
	n.mediaServerAdvertizeHost = advertizeHost
	n.mediaServerAdvertizePort = advertizePort
}

func (n *StatusNode) startWithDB(config *params.NodeConfig) error {
	n.config = config

	if err := n.setupRPCClient(); err != nil {
		return err
	}

	n.downloader = ipfs.NewDownloader(config.RootDataDir)

	if n.mediaServer == nil {
		if err := n.startMediaServer(); err != nil {
			return err
		}
	}
	n.mediaServer.SetDataProviders(n.appDB, n.walletDB, n.downloader)

	if err := n.createAndStartTokenManager(); err != nil {
		return err
	}

	if err := n.initServices(config, n.mediaServer); err != nil {
		return err
	}

	// Run migrations
	err := n.runServicesMigrations()
	if err != nil {
		return errorspkg.Wrap(err, "failed to run services migrations")
	}

	// Register services

	for _, service := range n.services {
		err := n.registerService(service)
		if err != nil {
			name := reflect.TypeOf(service).Name()
			text := fmt.Sprintf("failed to register service '%s'", name)
			return errorspkg.Wrap(err, text)
		}
	}

	// Start services

	err = n.timeSourceSrvc.Start(context.Background())
	if err != nil {
		return errorspkg.Wrap(err, "failed to start time source")
	}

	for _, service := range n.services {
		err := service.Start()
		if err != nil {
			name := reflect.TypeOf(service).Name()
			text := fmt.Sprintf("failed to start service '%s'", name)
			return errorspkg.Wrap(err, text)
		}
	}

	return nil
}

func (n *StatusNode) createAndStartTokenManager() error {
	const (
		defaultAutoRefreshInterval      = 30 * time.Minute // interval after which we should fetch the token lists from the remote source (or use the default one if remote source is not set)
		defaultAutoRefreshCheckInterval = 3 * time.Minute  // interval after which we should check if we should trigger the auto-refresh
	)

	autoRefreshInterval := defaultAutoRefreshInterval
	autoRefreshCheckInterval := defaultAutoRefreshCheckInterval
	if n.config.WalletConfig.TokensListsAutoRefreshInterval > 0 &&
		n.config.WalletConfig.TokensListsAutoRefreshCheckInterval > 0 &&
		n.config.WalletConfig.TokensListsAutoRefreshInterval > n.config.WalletConfig.TokensListsAutoRefreshCheckInterval {
		autoRefreshInterval = time.Duration(n.config.WalletConfig.TokensListsAutoRefreshInterval) * time.Second
		autoRefreshCheckInterval = time.Duration(n.config.WalletConfig.TokensListsAutoRefreshCheckInterval) * time.Second
	}

	accDB, err := accounts.NewDB(n.appDB)
	if err != nil {
		return err
	}

	n.tokenManager, err = token.NewTokenManager(n.walletDB, n.rpcClient, community.NewManager(n.appDB, n.mediaServer, nil),
		n.rpcClient.GetNetworkManager(), n.appDB, n.mediaServer, &n.walletFeed, n.accountsPublisher, accDB,
		autoRefreshInterval, autoRefreshCheckInterval)
	if err != nil {
		return err
	}

	return n.tokenManager.Start(context.Background())
}

func (n *StatusNode) setupRPCClient() (err error) {
	config := rpc.ClientConfig{
		Networks:          n.config.Networks,
		DB:                n.appDB,
		AccountsPublisher: n.accountsPublisher,
	}
	n.rpcClient, err = rpc.NewClient(config)
	if err != nil {
		return
	}
	n.rpcClient.Start(context.Background())
	return
}

// Stop will stop current StatusNode. A stopped node cannot be resumed.
func (n *StatusNode) Stop() error {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.logger.Debug("stopping")

	if !n.running.CompareAndSwap(true, false) {
		return ErrNoRunningNode
	}

	var errs []error
	n.timeSourceSrvc.Stop()

	for _, service := range n.services {
		err := service.Stop()
		errs = append(errs, err)
	}

	if n.localBackup != nil {
		n.localBackup.Stop()
		n.localBackup = nil
	}

	n.accountsPublisher.Close()

	n.rpcClient.Stop()
	n.rpcClient = nil
	n.config = nil

	n.mediaServer.SetDataProviders(nil, nil, nil)

	n.downloader.Stop()
	n.downloader = nil

	n.rpcStatsSrvc = nil
	n.accountsSrvc = nil
	n.browsersSrvc = nil
	n.permissionsSrvc = nil
	n.walletSrvc = nil
	n.localNotificationsSrvc = nil
	n.personalSrvc = nil
	n.timeSourceSrvc = nil
	n.wakuV2ExtSrvc = nil
	n.ensSrvc = nil
	n.communityTokensSrvc = nil
	n.stickersSrvc = nil
	n.connectorSrvc = nil
	n.publicMethods = make(map[string]bool)
	n.pendingTracker = nil
	n.appGeneralSrvc = nil
	n.newsfeedSrvc = nil

	n.logger.Debug("status node stopped")
	return errors.Join(errs...)
}

// IsRunning confirm that node is running.
func (n *StatusNode) IsRunning() bool {
	return n.running.Load()
}

func (n *StatusNode) CallInProcessRPC(inputJSON string) string {
	codec := rpc2.NewSingleRequestCodec(inputJSON)
	n.rpcServer.ServeCodec(codec.GethCodec(), 0)
	return codec.Output()
}

// RPCClient exposes reference to RPC client connected to the running node.
func (n *StatusNode) RPCClient() *rpc.Client {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.rpcClient
}

func (n *StatusNode) SetAppDB(db *sql.DB) {
	n.appDB = db
}

func (n *StatusNode) GetAppDB() *sql.DB {
	return n.appDB
}

func (n *StatusNode) SetMultiaccountsDB(db *multiaccounts.Database) {
	n.multiaccountsDB = db
}

func (n *StatusNode) SetWalletDB(db *sql.DB) {
	n.walletDB = db
}

func (n *StatusNode) GetWalletDB() *sql.DB {
	return n.walletDB
}

func (n *StatusNode) TokenManager() *token.Manager {
	return n.tokenManager
}

func (n *StatusNode) TokenBalancesFetcher() tokenbalances.FetcherIface {
	if n.walletSrvc != nil {
		return n.walletSrvc.GetTokenBalancesFetcher()
	}
	return nil
}

func (n *StatusNode) TokenBalancesStorage() tokenbalances.Storage {
	if n.walletSrvc != nil {
		return n.walletSrvc.GetTokenBalancesStorage()
	}
	return nil
}
