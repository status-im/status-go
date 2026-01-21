package wallet

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/golang/protobuf/proto"
	"go.uber.org/zap"

	accsmanagement "github.com/status-im/status-go/internal/accounts-management"
	accsmanagementtypes "github.com/status-im/status-go/internal/accounts-management/types"
	"github.com/status-im/status-go/internal/db/multiaccounts/accounts"
	"github.com/status-im/status-go/internal/logutils"
	"github.com/status-im/status-go/internal/rpc"
	rpc2 "github.com/status-im/status-go/internal/rpc"
	"github.com/status-im/status-go/internal/rpc/network"
	"github.com/status-im/status-go/internal/transactions"
	"github.com/status-im/status-go/services/communitytokens/communitytokensdatabase"
	"github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/multistandardbalance"
	"github.com/status-im/status-go/services/wallet/pendingtxtracker"
	"github.com/status-im/status-go/services/wallet/thirdparty/market/cryptocompare"
	"github.com/status-im/status-go/services/wallet/tokenbalances"
	"github.com/status-im/status-go/services/wallet/transferdetector"

	"github.com/ethereum/go-ethereum/event"
	gethrpc "github.com/ethereum/go-ethereum/rpc"

	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/pkg/pubsub"
	"github.com/status-im/status-go/protocol/backupsync"
	protocolCommon "github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/syncing"
	"github.com/status-im/status-go/server"
	"github.com/status-im/status-go/services/ens/ensresolver"
	"github.com/status-im/status-go/services/wallet/activity"
	"github.com/status-im/status-go/services/wallet/activityfetcher"
	alchemymanager "github.com/status-im/status-go/services/wallet/activityfetcher/alchemy"
	"github.com/status-im/status-go/services/wallet/blockchainstate"
	"github.com/status-im/status-go/services/wallet/collectibles"
	"github.com/status-im/status-go/services/wallet/collectibles/ownership"
	collectibles_ownership "github.com/status-im/status-go/services/wallet/collectibles/ownership"
	"github.com/status-im/status-go/services/wallet/community"
	"github.com/status-im/status-go/services/wallet/currency"
	"github.com/status-im/status-go/services/wallet/following"
	"github.com/status-im/status-go/services/wallet/leaderboard"
	"github.com/status-im/status-go/services/wallet/market"
	"github.com/status-im/status-go/services/wallet/onramp"
	"github.com/status-im/status-go/services/wallet/routeexecution"
	"github.com/status-im/status-go/services/wallet/router"
	"github.com/status-im/status-go/services/wallet/router/pathprocessor"
	"github.com/status-im/status-go/services/wallet/thirdparty"
	activityfetcher_alchemy "github.com/status-im/status-go/services/wallet/thirdparty/activity/alchemy"
	"github.com/status-im/status-go/services/wallet/thirdparty/collectibles/alchemy"
	"github.com/status-im/status-go/services/wallet/thirdparty/collectibles/rarible"
	"github.com/status-im/status-go/services/wallet/thirdparty/efp"
	"github.com/status-im/status-go/services/wallet/thirdparty/market/coingecko"
	"github.com/status-im/status-go/services/wallet/token"
	"github.com/status-im/status-go/services/wallet/tokenhistoricalownership"
	"github.com/status-im/status-go/services/wallet/transfer"
	"github.com/status-im/status-go/services/wallet/walletevent"
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
	rpcClient *rpc2.Client,
	accountsPublisher *pubsub.Publisher,
	gethManager *accsmanagement.AccountsManager,
	transactor *transactions.Transactor,
	config *params.NodeConfig,
	ensResolver *ensresolver.EnsResolver,
	pendingTxManager *pendingtxtracker.PendingTxTracker,
	feed *event.Feed,
	mediaServer *server.MediaServer,
	statusProxyStageName string,
) (*Service, error) {
	signals := &walletevent.SignalsTransmitter{
		Publisher: feed,
	}

	walletPublisher := pubsub.NewPublisher()

	communityManager := community.NewManager(db, mediaServer, feed)

	tokenManager := createTokenManager(
		config,
		db,
		rpcClient,
		rpcClient.GetNetworkManager(),
		appDB,
		accountsPublisher,
		accountsDB,
		communityManager,
		communitytokensdatabase.NewCommunityTokensDatabase(appDB),
		mediaServer,
		walletPublisher,
	)

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

		cryptoCompare := cryptocompare.NewClient()
		coingeckoClient := coingecko.NewClient()
		coingeckoProxy := createCoingeckoProxyClient(config.WalletConfig.MarketDataProxyConfig)
		cryptoCompareProxy := cryptocompare.NewClientWithParams(cryptocompare.Params{
			ID:       fmt.Sprintf("%s-proxy", cryptoCompare.ID()),
			URL:      fmt.Sprintf("https://%s.api.status.im/cryptocompare/", statusProxyStageName),
			User:     config.WalletConfig.StatusProxyMarketUser,
			Password: config.WalletConfig.StatusProxyMarketPassword,
		})
		marketProviders = []thirdparty.MarketDataProvider{
			coingeckoProxy, coingeckoClient, cryptoCompare, cryptoCompareProxy,
		}

		raribleClient := rarible.NewClient(config.WalletConfig.RaribleMainnetAPIKey, config.WalletConfig.RaribleTestnetAPIKey)
		alchemyClient := alchemy.NewClient(config.WalletConfig.AlchemyAPIKey)

		// Collectible providers in priority order (i.e. provider N+1 will be tried only if provider N fails)
		contractOwnershipProviders := []thirdparty.CollectibleContractOwnershipProvider{
			raribleClient,
			alchemyClient,
		}

		accountOwnershipProviders := []thirdparty.CollectibleAccountOwnershipProvider{
			raribleClient,
			alchemyClient,
		}

		collectibleDataProviders := []thirdparty.CollectibleDataProvider{
			raribleClient,
			alchemyClient,
		}

		collectionDataProviders := []thirdparty.CollectionDataProvider{
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

		pathProcessors = buildPathProcessors(rpcClient, transactor, tokenManager, ensResolver, featureFlags)

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

	tokenBalancesService := tokenbalances.NewService(
		tokenBalancesStorage,
		multistandardBalanceController.GetPublisher(),
		multistandardBalanceController,
		tokenManager,
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

	alchemyEthClientGetter := rpc2.NewProviderChainClientGetter(common.SmartProxyAlchemy, rpcClient)
	alchemyFetcherDb := activityfetcher_alchemy.NewPersistence(db)
	alchemyFetcherClient := activityfetcher_alchemy.NewClient(alchemyEthClientGetter)
	alchemyFetcherManager := alchemymanager.NewManager(alchemyFetcherClient, alchemyFetcherDb)
	activityFetcherManager := activityfetcher.NewManager(alchemyFetcherManager)
	activityFetcherService := activityfetcher.NewService(activityFetcherManager, rpcClient.GetNetworkManager(), accountsDB, accountsPublisher, rpcClient, feed)

	tokenHistoricalOwnershipService := tokenhistoricalownership.NewService(
		db,
		tokenhistoricalownership.NewStorage(db),
		activityFetcherService.GetPublisher(),
		activityFetcherService,
		tokenBalancesStorage,
		tokenBalancesService.GetPublisher(),
		transferDetectorController.GetPublisher(),
		accountsPublisher,
	)

	tokenManager.SetHistoricallyOwnedTokensProvider(tokenHistoricalOwnershipService)

	return &Service{
		db:                              db,
		accountsDB:                      accountsDB,
		rpcClient:                       rpcClient,
		tokenManager:                    tokenManager,
		communityManager:                communityManager,
		savedAddressesManager:           savedAddressesManager,
		transactionManager:              transactionManager,
		pendingTxManager:                pendingTxManager,
		multistandardBalanceController:  multistandardBalanceController,
		transferDetectorController:      transferDetectorController,
		tokenBalancesFetcher:            tokenBalancesFetcher,
		tokenBalancesStorage:            tokenBalancesStorage,
		cryptoOnRampManager:             cryptoOnRampManager,
		collectiblesManager:             collectiblesManager,
		collectibles:                    collectibles,
		followingManager:                followingManager,
		gethManager:                     gethManager,
		marketManager:                   marketManager,
		transactor:                      transactor,
		feed:                            feed,
		signals:                         signals,
		tokenBalancesService:            tokenBalancesService,
		tokenHistoricalOwnershipService: tokenHistoricalOwnershipService,
		currency:                        currency,
		activity:                        activity,
		decoder:                         NewDecoder(),
		blockChainState:                 blockChainState,
		keycardPairings:                 NewKeycardPairings(),
		config:                          config,
		featureFlags:                    featureFlags,
		router:                          router,
		routeExecutionManager:           routeExecutionManager,
		leaderboardService:              leaderboardService,
		activityFetcherService:          activityFetcherService,
		started:                         false,
	}, nil
}

func buildPathProcessors(
	rpcClient *rpc2.Client,
	transactor *transactions.Transactor,
	tokenManager *token.Manager,
	ensResolver *ensresolver.EnsResolver,
	featureFlags *protocolCommon.FeatureFlags,
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

	paraswap := pathprocessor.NewSwapParaswapProcessor(rpcClient, transactor, tokenManager)
	ret = append(ret, paraswap)

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

	communityDeployAssets := pathprocessor.NewCommunityDeployAssetsProcessor(rpcClient, transactor)
	ret = append(ret, communityDeployAssets)

	communityDeployCollectibles := pathprocessor.NewCommunityDeployCollectiblesProcessor(rpcClient, transactor)
	ret = append(ret, communityDeployCollectibles)

	communityDeployOwnerToken := pathprocessor.NewCommunityDeployOwnerTokenProcessor(rpcClient, transactor)
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
	db                              *sql.DB
	accountsDB                      *accounts.Database
	rpcClient                       *rpc2.Client
	tokenManager                    *token.Manager
	communityManager                *community.Manager
	savedAddressesManager           *SavedAddressesManager
	transactionManager              *transfer.TransactionManager
	pendingTxManager                *pendingtxtracker.PendingTxTracker
	multistandardBalanceController  *multistandardbalance.Controller
	tokenBalancesFetcher            tokenbalances.FetcherIface
	tokenBalancesStorage            tokenbalances.Storage
	transferDetectorController      *transferdetector.Controller
	cryptoOnRampManager             *onramp.Manager
	collectiblesManager             *collectibles.Manager
	collectibles                    *collectibles.Service
	followingManager                *following.Manager
	gethManager                     *accsmanagement.AccountsManager
	marketManager                   *market.Manager
	transactor                      *transactions.Transactor
	feed                            *event.Feed
	signals                         *walletevent.SignalsTransmitter
	tokenBalancesService            *tokenbalances.Service
	tokenHistoricalOwnershipService *tokenhistoricalownership.Service
	currency                        *currency.Service
	activity                        *activity.Service
	decoder                         *Decoder
	blockChainState                 *blockchainstate.BlockChainState
	keycardPairings                 *KeycardPairings
	config                          *params.NodeConfig
	featureFlags                    *protocolCommon.FeatureFlags
	router                          *router.Router
	routeExecutionManager           *routeexecution.Manager
	leaderboardService              *leaderboard.MarketDataService
	activityFetcherService          *activityfetcher.Service
	walletPublisher                 *pubsub.Publisher
	started                         bool

	cancelWalletServiceCtx context.CancelFunc
}

// Start signals transmitter.
func (s *Service) Start() error {
	if ThirdpartyServicesEnabled(s.accountsDB) {
		ctx, cancel := context.WithCancel(context.Background())
		s.cancelWalletServiceCtx = cancel

		s.tokenManager.Start(ctx)
		s.multistandardBalanceController.Start()
		s.transferDetectorController.Start()
		s.tokenBalancesService.Start()
		s.tokenHistoricalOwnershipService.Start()
		s.currency.Start(ctx)
		err := s.signals.Start(ctx)
		s.collectibles.Start(ctx)
		s.leaderboardService.Start(ctx)
		s.activityFetcherService.Start(ctx)
		s.started = true
		return err
	}
	return nil
}

// Set external Collectibles community info provider
func (s *Service) SetWalletCommunityInfoProvider(provider thirdparty.CommunityInfoProvider) {
	s.communityManager.SetCommunityInfoProvider(provider)
}

// Stop reactor and close db.
func (s *Service) Stop() error {
	logutils.ZapLogger().Info("wallet will be stopped")
	s.walletPublisher.Close()

	s.router.Stop()
	s.signals.Stop()
	s.multistandardBalanceController.Stop()
	s.transferDetectorController.Stop()
	s.tokenBalancesService.Stop()
	s.tokenHistoricalOwnershipService.Stop()
	s.activity.Stop()
	s.collectibles.Stop()
	s.tokenManager.Stop()
	s.leaderboardService.Stop()
	s.started = false
	logutils.ZapLogger().Info("wallet stopped")

	// Cancel wallet service context
	if s.cancelWalletServiceCtx != nil {
		s.cancelWalletServiceCtx()
		s.cancelWalletServiceCtx = nil
	}

	return nil
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

func (s *Service) GetRPCClient() *rpc2.Client {
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

func createTokenManager(
	config *params.NodeConfig,
	walletDB *sql.DB,
	ethClientGetter rpc.EthClientGetter,
	networkManager network.ManagerInterface,
	appDB *sql.DB,
	accountsPublisher *pubsub.Publisher,
	accountsDB *accounts.Database,
	communityManager *community.Manager,
	communityTokensDB *communitytokensdatabase.Database,
	communityTokenImageBuilder token.CommunityTokenImageBuilder,
	walletPublisher *pubsub.Publisher,
) *token.Manager {
	const (
		defaultAutoRefreshInterval      = 30 * time.Minute // interval after which we should fetch the token lists from the remote source (or use the default one if remote source is not set)
		defaultAutoRefreshCheckInterval = 3 * time.Minute  // interval after which we should check if we should trigger the auto-refresh
	)

	autoRefreshInterval := defaultAutoRefreshInterval
	autoRefreshCheckInterval := defaultAutoRefreshCheckInterval
	if config.WalletConfig.TokensListsAutoRefreshInterval > 0 &&
		config.WalletConfig.TokensListsAutoRefreshCheckInterval > 0 &&
		config.WalletConfig.TokensListsAutoRefreshInterval > config.WalletConfig.TokensListsAutoRefreshCheckInterval {
		autoRefreshInterval = time.Duration(config.WalletConfig.TokensListsAutoRefreshInterval) * time.Second
		autoRefreshCheckInterval = time.Duration(config.WalletConfig.TokensListsAutoRefreshCheckInterval) * time.Second
	}

	tokenManager, err := token.NewTokenManager(
		walletDB,
		ethClientGetter,
		networkManager,
		appDB,
		accountsPublisher,
		accountsDB,
		communityManager,
		communityTokensDB,
		communityTokenImageBuilder,
		autoRefreshInterval,
		autoRefreshCheckInterval,
		walletPublisher)
	if err != nil {
		logutils.ZapLogger().Error("failed to create token manager", zap.Error(err))
		return nil
	}

	// check for possible custom tokens in the config
	if len(config.WalletConfig.CustomTokens) > 0 {
		for _, token := range config.WalletConfig.CustomTokens {
			err := tokenManager.UpsertCustom(*token)
			if err != nil {
				logutils.ZapLogger().Error("failed to upsert custom token", zap.Error(err))
				return nil
			}
		}
	}

	return tokenManager
}
