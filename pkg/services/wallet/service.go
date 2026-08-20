package wallet

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"

	accsmanagement "github.com/status-im/status-go/internal/accounts-management"
	accsmanagementtypes "github.com/status-im/status-go/internal/accounts-management/types"
	"github.com/status-im/status-go/internal/db/multiaccounts/accounts"
	"github.com/status-im/status-go/internal/logutils"
	"github.com/status-im/status-go/internal/rpc"
	"github.com/status-im/status-go/internal/transactions"
	"github.com/status-im/status-go/pkg/services/wallet/common"
	"github.com/status-im/status-go/pkg/services/wallet/multistandardbalance"
	"github.com/status-im/status-go/pkg/services/wallet/pendingtxtracker"
	"github.com/status-im/status-go/pkg/services/wallet/tokenbalances"
	"github.com/status-im/status-go/pkg/services/wallet/transferdetector"

	ethCommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/event"
	gethrpc "github.com/ethereum/go-ethereum/rpc"

	"github.com/status-im/status-go/internal/protocol/backupsync"
	protocolCommon "github.com/status-im/status-go/internal/protocol/common"
	"github.com/status-im/status-go/internal/protocol/protobuf"
	"github.com/status-im/status-go/internal/protocol/syncing"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/pkg/pubsub"
	"github.com/status-im/status-go/pkg/security"
	"github.com/status-im/status-go/pkg/services/ens/ensresolver"
	"github.com/status-im/status-go/pkg/services/media"
	"github.com/status-im/status-go/pkg/services/wallet/activity"
	"github.com/status-im/status-go/pkg/services/wallet/activityfetcher"
	alchemymanager "github.com/status-im/status-go/pkg/services/wallet/activityfetcher/alchemy"
	"github.com/status-im/status-go/pkg/services/wallet/blockchainstate"
	"github.com/status-im/status-go/pkg/services/wallet/collectibles"
	"github.com/status-im/status-go/pkg/services/wallet/collectibles/ownership"
	collectibles_ownership "github.com/status-im/status-go/pkg/services/wallet/collectibles/ownership"
	"github.com/status-im/status-go/pkg/services/wallet/community"
	"github.com/status-im/status-go/pkg/services/wallet/currency"
	"github.com/status-im/status-go/pkg/services/wallet/following"
	"github.com/status-im/status-go/pkg/services/wallet/leaderboard"
	"github.com/status-im/status-go/pkg/services/wallet/market"
	"github.com/status-im/status-go/pkg/services/wallet/onramp"
	"github.com/status-im/status-go/pkg/services/wallet/puzzleauth"
	"github.com/status-im/status-go/pkg/services/wallet/routeexecution"
	"github.com/status-im/status-go/pkg/services/wallet/router"
	"github.com/status-im/status-go/pkg/services/wallet/router/pathprocessor"
	"github.com/status-im/status-go/pkg/services/wallet/thirdparty"
	activityfetcher_alchemy "github.com/status-im/status-go/pkg/services/wallet/thirdparty/activity/alchemy"
	"github.com/status-im/status-go/pkg/services/wallet/thirdparty/collectibles/alchemy"
	"github.com/status-im/status-go/pkg/services/wallet/thirdparty/collectibles/rarible"
	"github.com/status-im/status-go/pkg/services/wallet/thirdparty/efp"
	"github.com/status-im/status-go/pkg/services/wallet/thirdparty/market/coingecko"
	"github.com/status-im/status-go/pkg/services/wallet/token"
	"github.com/status-im/status-go/pkg/services/wallet/transfer"
	"github.com/status-im/status-go/pkg/services/wallet/walletevent"
)

const (
	EventWatchOnlyAccountRetrieved walletevent.EventType = "wallet-watch-only-account-retrieved"
)

func createCoingeckoProxyClient(config params.MarketDataProxyConfig) *coingecko.Client {
	baseURL := leaderboard.GetMarketProxyUrl(config.UrlOverride.Reveal(), config.StageName)

	return coingecko.NewClientWithParams(coingecko.Params{
		URL:      baseURL,
		User:     config.User,
		Password: config.Password,
	})
}

func createAlchemyProxyClient(config params.NftProxyConfig) *alchemy.Client {
	var creds *thirdparty.BasicCreds
	if !config.User.Empty() {
		creds = &thirdparty.BasicCreds{
			User:     config.User,
			Password: config.Password,
		}
	}

	var alchemyHTTP *http.Client
	if config.UsePuzzleAuth {
		origin := alchemy.GetNftProxyHost(config.UrlOverride.Reveal(), config.StageName)
		alchemyHTTP = &http.Client{
			Timeout:   time.Minute,
			Transport: puzzleauth.NewTransport(origin, http.DefaultTransport),
		}
	}

	return alchemy.NewClientWithParams(alchemy.Params{
		IsProxy:        true,
		ProxyCustomURL: config.UrlOverride.Reveal(),
		ProxyStageName: config.StageName,
		APIKey:         security.SensitiveString{},
		Creds:          creds,
		HttpClient:     alchemyHTTP,
	})
}

func ThirdpartyServicesEnabled(accountsDB *accounts.Database) bool {
	enabled, err := accountsDB.ThirdpartyServicesEnabled()
	if err != nil {
		return true
	}
	return enabled
}

// NewService initializes service instance.
func NewService(
	db *sql.DB,
	accountsDB *accounts.Database,
	appDB *sql.DB,
	rpcClient *rpc.Client,
	accountsPublisher *pubsub.Publisher,
	gethManager *accsmanagement.AccountsManager,
	transactor *transactions.Transactor,
	config *params.NodeConfig,
	ensResolver *ensresolver.EnsResolver,
	pendingTxManager *pendingtxtracker.PendingTxTracker,
	feed *event.Feed,
	mediaServer *media.Server,
	tokenManager *token.Manager,
) (*Service, error) {
	signals := &walletevent.SignalsTransmitter{
		Publisher: feed,
	}
	communityManager := community.NewManager(db, mediaServer, feed)

	featureFlags := &protocolCommon.FeatureFlags{}

	savedAddressesManager := &SavedAddressesManager{db: db}
	transactionManager := transfer.NewTransactionManager(gethManager, transactor, config, accountsDB, pendingTxManager, feed)
	blockChainState := blockchainstate.NewBlockChainState(rpcClient)

	thirdpartyServicesEnabled := ThirdpartyServicesEnabled(accountsDB)

	var cryptoOnRampProviders []onramp.Provider = []onramp.Provider{}
	var marketProviders []thirdparty.MarketDataProvider = []thirdparty.MarketDataProvider{}
	var collectibleProviders thirdparty.CollectibleProviders = thirdparty.CollectibleProviders{}
	var pathProcessors []pathprocessor.PathProcessor = []pathprocessor.PathProcessor{}
	var leaderboardConfig leaderboard.ServiceConfig = leaderboard.NewDefaultServiceConfig()
	var followingManager *following.Manager

	if thirdpartyServicesEnabled {

		cryptoOnRampProviders = []onramp.Provider{
			onramp.NewMoonPayProvider(),
		}

		coingeckoClient := coingecko.NewClientWithParams(coingecko.Params{
			CoingeckoAPIKey:     config.WalletConfig.CoingeckoAPIKey,
			CoingeckoDemoAPIKey: config.WalletConfig.CoingeckoDemoAPIKey,
		})
		coingeckoProxy := createCoingeckoProxyClient(config.WalletConfig.MarketDataProxyConfig)
		marketProviders = []thirdparty.MarketDataProvider{
			coingeckoProxy, coingeckoClient,
		}

		raribleClient := rarible.NewClient(config.WalletConfig.RaribleMainnetAPIKey, config.WalletConfig.RaribleTestnetAPIKey)
		alchemyClient := alchemy.NewClient(config.WalletConfig.AlchemyAPIKey)
		alchemyProxy := createAlchemyProxyClient(config.WalletConfig.NftProxyConfig)

		// Collectible providers in priority order (i.e. provider N+1 will be tried only if provider N fails)
		contractOwnershipProviders := []thirdparty.CollectibleContractOwnershipProvider{
			alchemyProxy,
			raribleClient,
			alchemyClient,
		}

		accountOwnershipProviders := []thirdparty.CollectibleAccountOwnershipProvider{
			alchemyProxy,
			raribleClient,
			alchemyClient,
		}

		collectibleDataProviders := []thirdparty.CollectibleDataProvider{
			alchemyProxy,
			raribleClient,
			alchemyClient,
		}

		collectionDataProviders := []thirdparty.CollectionDataProvider{
			alchemyProxy,
			raribleClient,
			alchemyClient,
		}

		collectibleSearchProviders := []thirdparty.CollectibleSearchProvider{
			raribleClient,
		}

		collectibleProviders = thirdparty.CollectibleProviders{
			ContractOwnershipProviders: contractOwnershipProviders,
			AccountOwnershipProviders:  accountOwnershipProviders,
			CollectibleDataProviders:   collectibleDataProviders,
			CollectionDataProviders:    collectionDataProviders,
			SearchProviders:            collectibleSearchProviders,
		}

		pathProcessors = buildPathProcessors(rpcClient, transactor, tokenManager, ensResolver, featureFlags, config.WalletConfig.CommunityTokenDeployerOverrides)

		leaderboardConfig = leaderboard.NewLeaderboardConfig(config.WalletConfig.MarketDataProxyConfig)

		// EFP (Ethereum Follow Protocol) provider
		efpHTTPClient := thirdparty.NewHTTPClient(
			thirdparty.WithDetailedTimeouts(
				5*time.Second,  // dialTimeout
				5*time.Second,  // tlsHandshakeTimeout
				5*time.Second,  // responseHeaderTimeout
				20*time.Second, // requestTimeout
			),
			thirdparty.WithMaxRetries(5),
		)
		efpClient := efp.NewClient(efpHTTPClient)
		followingManager = following.NewManager(efpClient, logutils.ZapLogger().Named("FollowingManager"))
	}

	// Initialize followingManager with nil provider if third-party services are disabled
	if followingManager == nil {
		followingManager = following.NewManager(nil, logutils.ZapLogger().Named("FollowingManager"))
	}

	cryptoOnRampManager := onramp.NewManager(cryptoOnRampProviders)

	marketManager := market.NewManager(marketProviders, tokenManager, feed)
	currency := currency.NewService(db, feed, tokenManager, marketManager)

	multistandardBalanceFetcher := multistandardbalance.NewFetcher(rpcClient, multistandardbalance.DefaultBatchSize, config.WalletConfig.MulticallOverrides)
	multistandardBalanceStorage := multistandardbalance.NewStorageMemory()
	multistandardBalanceController := multistandardbalance.NewController(
		multistandardbalance.DefaultControllerConfig(),
		multistandardBalanceStorage,
		multistandardBalanceFetcher,
		accountsDB,
		accountsPublisher,
		rpcClient.GetNetworkManager(),
		NewMultistandardBalanceTokenListProvider(tokenManager),
		NewMultistandardBalanceCollectiblesListProvider(ownership.NewOwnershipDB(db), collectibles.NewContractTypeDB(db)),
		blockChainState,
		feed,
		logutils.ZapLogger().Named("MultistandardBalanceController"),
	)

	tokenBalancesFetcher := tokenbalances.NewFetcher(multistandardBalanceFetcher)
	tokenBalancesStorage := tokenbalances.NewStorageMultistandardBalance(multistandardBalanceStorage)

	transferDetectorController := transferdetector.NewController(
		transferdetector.DefaultControllerConfig(),
		transferdetector.NewFetcher(rpcClient),
		accountsDB,
		accountsPublisher,
		rpcClient.GetNetworkManager(),
		blockChainState,
		logutils.ZapLogger().Named("TransferDetectorController"),
	)

	reader := NewReader(
		tokenManager,
		marketManager,
		feed,
		multistandardBalanceController.GetPublisher(),
		tokenBalancesStorage,
		transferDetectorController.GetPublisher(),
	)

	collectiblesPublisher := pubsub.NewPublisher()
	collectiblesManager := collectibles.NewManager(
		db,
		rpcClient,
		communityManager,
		collectibleProviders,
		mediaServer,
		feed,
	)
	collectiblesOwnershipController := collectibles_ownership.NewController(
		ownership.NewOwnershipDB(db), accountsDB, accountsPublisher, rpcClient.GetNetworkManager(),
		multistandardBalanceController.GetPublisher(),
		transferDetectorController.GetPublisher(),
		blockChainState,
		collectiblesManager,
		collectiblesPublisher,
		logutils.ZapLogger().Named("CollectiblesOwnershipController"),
	)
	collectiblesOwnershipController.SetChainSupportedCheck(func(chainID common.ChainID) bool {
		return !collectibles.IsUnsupportedCollectibleChain(uint64(chainID))
	})
	collectibles := collectibles.NewService(
		db,
		feed,
		communityManager,
		collectiblesManager,
		collectiblesOwnershipController,
		collectiblesPublisher)

	activity := activity.NewService(db, accountsDB, tokenManager, collectiblesManager, feed)

	router := router.NewRouter(rpcClient, transactor, tokenManager, tokenBalancesFetcher, marketManager, collectibles,
		collectiblesManager)
	for _, processor := range pathProcessors {
		router.AddPathProcessor(processor)
	}

	routeExecutionManager := routeexecution.NewManager(db, feed, router, tokenManager, transactionManager)

	leaderboardService := leaderboard.NewMarketDataService(leaderboardConfig, db, feed)

	alchemyEthClientGetter := rpc.NewProviderChainClientGetter(common.SmartProxyAlchemy, rpcClient)
	alchemyFetcherDb := activityfetcher_alchemy.NewPersistence(db)
	alchemyFetcherClient := activityfetcher_alchemy.NewClient(alchemyEthClientGetter)
	alchemyFetcherManager := alchemymanager.NewManager(alchemyFetcherClient, alchemyFetcherDb)
	activityFetcherManager := activityfetcher.NewManager(alchemyFetcherManager)
	activityFetcherService := activityfetcher.NewService(activityFetcherManager, rpcClient.GetNetworkManager(), accountsDB, accountsPublisher, rpcClient, feed, multistandardBalanceController.GetPublisher())

	return &Service{
		db:                             db,
		accountsDB:                     accountsDB,
		rpcClient:                      rpcClient,
		tokenManager:                   tokenManager,
		communityManager:               communityManager,
		savedAddressesManager:          savedAddressesManager,
		transactionManager:             transactionManager,
		pendingTxManager:               pendingTxManager,
		multistandardBalanceController: multistandardBalanceController,
		transferDetectorController:     transferDetectorController,
		tokenBalancesFetcher:           tokenBalancesFetcher,
		tokenBalancesStorage:           tokenBalancesStorage,
		cryptoOnRampManager:            cryptoOnRampManager,
		collectiblesManager:            collectiblesManager,
		collectibles:                   collectibles,
		followingManager:               followingManager,
		gethManager:                    gethManager,
		marketManager:                  marketManager,
		transactor:                     transactor,
		feed:                           feed,
		signals:                        signals,
		reader:                         reader,
		currency:                       currency,
		activity:                       activity,
		decoder:                        NewDecoder(),
		blockChainState:                blockChainState,
		keycardPairings:                NewKeycardPairings(),
		config:                         config,
		featureFlags:                   featureFlags,
		router:                         router,
		routeExecutionManager:          routeExecutionManager,
		leaderboardService:             leaderboardService,
		activityFetcherService:         activityFetcherService,
		started:                        false,
	}, nil
}

func buildPathProcessors(
	rpcClient *rpc.Client,
	transactor *transactions.Transactor,
	tokenManager *token.Manager,
	ensResolver *ensresolver.EnsResolver,
	featureFlags *protocolCommon.FeatureFlags,
	deployerOverrides map[uint64]ethCommon.Address,
) []pathprocessor.PathProcessor {
	ret := make([]pathprocessor.PathProcessor, 0)

	transfer := pathprocessor.NewTransferProcessor(rpcClient, transactor)
	ret = append(ret, transfer)

	erc721Transfer := pathprocessor.NewNFTProcessor(rpcClient, transactor)
	ret = append(ret, erc721Transfer)

	erc1155Transfer := pathprocessor.NewERC1155Processor(rpcClient, transactor)
	ret = append(ret, erc1155Transfer)

	hop := pathprocessor.NewHopBridgeProcessor(rpcClient, transactor, tokenManager, rpcClient.GetNetworkManager())
	ret = append(ret, hop)

	// disable paraswap, todo: put it back after testing
	// paraswap := pathprocessor.NewSwapParaswapProcessor(rpcClient, transactor, tokenManager)
	// ret = append(ret, paraswap)

	lifi := pathprocessor.NewLiFiProcessor(rpcClient, transactor, tokenManager)
	ret = append(ret, lifi)

	ensRegister := pathprocessor.NewENSRegisterProcessor(rpcClient, transactor, ensResolver)
	ret = append(ret, ensRegister)

	ensRelease := pathprocessor.NewENSReleaseProcessor(rpcClient, transactor, ensResolver)
	ret = append(ret, ensRelease)

	ensPublicKey := pathprocessor.NewENSPublicKeyProcessor(rpcClient, transactor, ensResolver)
	ret = append(ret, ensPublicKey)

	buyStickers := pathprocessor.NewStickersBuyProcessor(rpcClient, transactor)
	ret = append(ret, buyStickers)

	communityBurn := pathprocessor.NewCommunityBurnProcessor(rpcClient, transactor)
	ret = append(ret, communityBurn)

	communityDeployAssets := pathprocessor.NewCommunityDeployAssetsProcessor(rpcClient, transactor, deployerOverrides)
	ret = append(ret, communityDeployAssets)

	communityDeployCollectibles := pathprocessor.NewCommunityDeployCollectiblesProcessor(rpcClient, transactor, deployerOverrides)
	ret = append(ret, communityDeployCollectibles)

	communityDeployOwnerToken := pathprocessor.NewCommunityDeployOwnerTokenProcessor(rpcClient, transactor, deployerOverrides)
	ret = append(ret, communityDeployOwnerToken)

	communityMintTokens := pathprocessor.NewCommunityMintTokensProcessor(rpcClient, transactor)
	ret = append(ret, communityMintTokens)

	communityRemoteBurn := pathprocessor.NewCommunityRemoteBurnProcessor(rpcClient, transactor)
	ret = append(ret, communityRemoteBurn)

	communitySetSignerPubKey := pathprocessor.NewCommunitySetSignerPubKeyProcessor(rpcClient, transactor)
	ret = append(ret, communitySetSignerPubKey)

	return ret
}

// Service is a wallet service.
type Service struct {
	db                             *sql.DB
	accountsDB                     *accounts.Database
	rpcClient                      *rpc.Client
	tokenManager                   *token.Manager
	communityManager               *community.Manager
	savedAddressesManager          *SavedAddressesManager
	transactionManager             *transfer.TransactionManager
	pendingTxManager               *pendingtxtracker.PendingTxTracker
	multistandardBalanceController *multistandardbalance.Controller
	tokenBalancesFetcher           tokenbalances.FetcherIface
	tokenBalancesStorage           tokenbalances.Storage
	transferDetectorController     *transferdetector.Controller
	cryptoOnRampManager            *onramp.Manager
	collectiblesManager            *collectibles.Manager
	collectibles                   *collectibles.Service
	followingManager               *following.Manager
	gethManager                    *accsmanagement.AccountsManager
	marketManager                  *market.Manager
	transactor                     *transactions.Transactor
	feed                           *event.Feed
	signals                        *walletevent.SignalsTransmitter
	reader                         *Reader
	currency                       *currency.Service
	activity                       *activity.Service
	decoder                        *Decoder
	blockChainState                *blockchainstate.BlockChainState
	keycardPairings                *KeycardPairings
	config                         *params.NodeConfig
	featureFlags                   *protocolCommon.FeatureFlags
	router                         *router.Router
	routeExecutionManager          *routeexecution.Manager
	leaderboardService             *leaderboard.MarketDataService
	activityFetcherService         *activityfetcher.Service
	started                        bool
	paused                         bool
	mu                             sync.Mutex
	cancelWalletServiceCtx         context.CancelFunc
}

// Start signals transmitter.
func (s *Service) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !ThirdpartyServicesEnabled(s.accountsDB) {
		return nil
	}
	if s.started && !s.paused {
		return nil
	}
	err := s.startBackgroundWorkers()
	if err == nil {
		s.started = true
		s.paused = false
	}
	return err
}

// Set external Collectibles community info provider
func (s *Service) SetWalletCommunityInfoProvider(provider thirdparty.CommunityInfoProvider) {
	s.communityManager.SetCommunityInfoProvider(provider)
}

// Stop reactor and close db.
func (s *Service) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	logutils.ZapLogger().Info("wallet will be stopped")
	s.router.Stop()
	s.signals.Stop()
	s.multistandardBalanceController.Stop()
	s.transferDetectorController.Stop()
	s.reader.Stop()
	s.activity.Stop()
	s.collectibles.Stop()
	s.tokenManager.Stop()
	s.leaderboardService.Stop()
	s.started = false
	s.paused = false
	logutils.ZapLogger().Info("wallet stopped")

	// Cancel wallet service context
	if s.cancelWalletServiceCtx != nil {
		s.cancelWalletServiceCtx()
		s.cancelWalletServiceCtx = nil
	}

	return nil
}

func (s *Service) startBackgroundWorkers() error {
	if s.cancelWalletServiceCtx != nil {
		return nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.cancelWalletServiceCtx = cancel

	s.multistandardBalanceController.Start()
	s.transferDetectorController.Start()
	s.currency.Start(ctx)
	err := s.signals.Start(ctx)
	s.collectibles.Start(ctx)
	s.leaderboardService.Start(ctx)
	s.activityFetcherService.Start(ctx)
	if s.reader != nil && !s.reader.IsRunning() {
		_ = s.reader.Start()
	}
	return err
}

func (s *Service) stopBackgroundWorkers() {
	if s.reader != nil && s.reader.IsRunning() {
		s.reader.Stop()
	}
	s.signals.Stop()
	s.multistandardBalanceController.Stop()
	s.transferDetectorController.Stop()
	s.collectibles.Stop()
	s.leaderboardService.Stop()
	if s.cancelWalletServiceCtx != nil {
		s.cancelWalletServiceCtx()
		s.cancelWalletServiceCtx = nil
	}
}

// APIs returns list of available RPC APIs.
func (s *Service) APIs() []gethrpc.API {
	return []gethrpc.API{
		{
			Namespace: "wallet",
			Version:   "0.1.0",
			Service:   NewAPI(s),
			Public:    true,
		},
	}
}

func (s *Service) IsStarted() bool {
	return s.started
}

func (s *Service) KeycardPairings() *KeycardPairings {
	return s.keycardPairings
}

func (s *Service) Config() *params.NodeConfig {
	return s.config
}

func (s *Service) FeatureFlags() *protocolCommon.FeatureFlags {
	return s.featureFlags
}

func (s *Service) GetRPCClient() *rpc.Client {
	return s.rpcClient
}

func (s *Service) GetTransactor() *transactions.Transactor {
	return s.transactor
}

func (s *Service) GetTokenManager() *token.Manager {
	return s.tokenManager
}

func (s *Service) GetTokenBalancesFetcher() tokenbalances.FetcherIface {
	return s.tokenBalancesFetcher
}

func (s *Service) GetTokenBalancesStorage() tokenbalances.Storage {
	return s.tokenBalancesStorage
}

func (s *Service) GetMarketManager() *market.Manager {
	return s.marketManager
}

func (s *Service) GetCollectiblesService() *collectibles.Service {
	return s.collectibles
}

func (s *Service) GetCollectiblesManager() *collectibles.Manager {
	return s.collectiblesManager
}

// LocalBackup Code
func (s *Service) prepareSyncAccountMessage(acc *accsmanagementtypes.Account) *protobuf.SyncAccount {
	return &protobuf.SyncAccount{
		Clock:                 acc.Clock,
		Address:               acc.Address.Bytes(),
		KeyUid:                acc.KeyUID,
		PublicKey:             acc.PublicKey,
		Path:                  acc.Path,
		Name:                  acc.Name,
		ColorId:               string(acc.ColorID),
		Emoji:                 acc.Emoji,
		Wallet:                acc.Wallet,
		Chat:                  acc.Chat,
		Hidden:                acc.Hidden,
		Removed:               acc.Removed,
		Operable:              string(acc.Operable),
		Position:              acc.Position,
		ProdPreferredChainIDs: acc.ProdPreferredChainIDs,
		TestPreferredChainIDs: acc.TestPreferredChainIDs,
	}
}

func (s *Service) backupWatchOnlyAccounts() ([]*protobuf.Backup, error) {
	accounts, err := s.accountsDB.GetAllWatchOnlyAccounts()
	if err != nil {
		return nil, err
	}

	var backupMessages []*protobuf.Backup
	for _, acc := range accounts {

		backupMessage := &protobuf.Backup{}
		backupMessage.WatchOnlyAccount = s.prepareSyncAccountMessage(acc)

		backupMessages = append(backupMessages, backupMessage)
	}

	return backupMessages, nil
}

func (s *Service) ExportBackup() ([]byte, error) {
	backup := &protobuf.WalletLocalBackup{}

	woAccountsToBackup, err := s.backupWatchOnlyAccounts()
	if err != nil {
		return nil, err
	}
	for _, d := range woAccountsToBackup {
		backup.WatchOnlyAccounts = append(backup.WatchOnlyAccounts, d.WatchOnlyAccount)
	}

	return proto.Marshal(backup)
}

func (s *Service) handleWatchOnlyAccount(message *protobuf.SyncAccount) error {
	if message == nil {
		return nil
	}

	acc, err := syncing.HandleSyncWatchOnlyAccount(s.accountsDB, message, nil)
	if err != nil && !errors.Is(err, syncing.ErrTryingToStoreOldWalletAccount) {
		return err
	}
	response := backupsync.BackedUpDataResponse{
		WatchOnlyAccount: acc,
	}
	encodedmessage, err := json.Marshal(response)
	if err != nil {
		return err
	}
	event := walletevent.Event{
		Type:    EventWatchOnlyAccountRetrieved,
		Message: string(encodedmessage),
	}
	s.feed.Send(event)

	return nil
}

func (s *Service) ImportBackup(data []byte) error {
	var backup protobuf.WalletLocalBackup
	err := proto.Unmarshal(data, &backup)
	if err != nil {
		return err
	}
	var errs []error

	for _, watchOnlyAccount := range backup.WatchOnlyAccounts {
		err = s.handleWatchOnlyAccount(watchOnlyAccount)
		if err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}
