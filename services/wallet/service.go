package wallet

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/p2p"
	gethrpc "github.com/ethereum/go-ethereum/rpc"

	"github.com/status-im/status-go/account"
	"github.com/status-im/status-go/logutils"
	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/params"
	protocolCommon "github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/rpc"
	"github.com/status-im/status-go/server"
	"github.com/status-im/status-go/services/ens"
	"github.com/status-im/status-go/services/stickers"
	"github.com/status-im/status-go/services/wallet/activity"
	"github.com/status-im/status-go/services/wallet/balance"
	"github.com/status-im/status-go/services/wallet/blockchainstate"
	"github.com/status-im/status-go/services/wallet/collectibles"
	"github.com/status-im/status-go/services/wallet/community"
	"github.com/status-im/status-go/services/wallet/currency"
	"github.com/status-im/status-go/services/wallet/history"
	"github.com/status-im/status-go/services/wallet/market"
	"github.com/status-im/status-go/services/wallet/onramp"
	"github.com/status-im/status-go/services/wallet/routeexecution"
	"github.com/status-im/status-go/services/wallet/router"
	"github.com/status-im/status-go/services/wallet/router/pathprocessor"
	"github.com/status-im/status-go/services/wallet/thirdparty"
	"github.com/status-im/status-go/services/wallet/thirdparty/alchemy"
	"github.com/status-im/status-go/services/wallet/thirdparty/coingecko"
	"github.com/status-im/status-go/services/wallet/thirdparty/cryptocompare"
	"github.com/status-im/status-go/services/wallet/thirdparty/opensea"
	"github.com/status-im/status-go/services/wallet/thirdparty/rarible"
	"github.com/status-im/status-go/services/wallet/token"
	"github.com/status-im/status-go/services/wallet/transfer"
	"github.com/status-im/status-go/services/wallet/walletevent"
	"github.com/status-im/status-go/transactions"
)

const (
	EventBlockchainStatusChanged walletevent.EventType = "wallet-blockchain-status-changed"
)

// NewService initializes service instance.
func NewService(
	db *sql.DB,
	accountsDB *accounts.Database,
	appDB *sql.DB,
	rpcClient *rpc.Client,
	accountFeed *event.Feed,
	settingsFeed *event.Feed,
	gethManager *account.GethManager,
	transactor *transactions.Transactor,
	config *params.NodeConfig,
	ens *ens.Service,
	stickers *stickers.Service,
	pendingTxManager *transactions.PendingTxTracker,
	feed *event.Feed,
	mediaServer *server.MediaServer,
	statusProxyStageName string,
) *Service {
	signals := &walletevent.SignalsTransmitter{
		Publisher: feed,
	}
	blockchainStatus := make(map[uint64]string)
	mutex := sync.Mutex{}
	rpcClient.SetWalletNotifier(func(chainID uint64, message string) {
		mutex.Lock()
		defer mutex.Unlock()

		if len(blockchainStatus) == 0 {
			networks, err := rpcClient.NetworkManager.Get(false)
			if err != nil {
				return
			}

			for _, network := range networks {
				blockchainStatus[network.ChainID] = "up"
			}
		}

		blockchainStatus[chainID] = message
		encodedmessage, err := json.Marshal(blockchainStatus)
		if err != nil {
			return
		}

		feed.Send(walletevent.Event{
			Type:     EventBlockchainStatusChanged,
			Accounts: []common.Address{},
			Message:  string(encodedmessage),
			At:       time.Now().Unix(),
			ChainID:  chainID,
		})
	})

	communityManager := community.NewManager(db, mediaServer, feed)
	balanceCacher := balance.NewCacherWithTTL(5 * time.Minute)
	tokenManager := token.NewTokenManager(db, rpcClient, communityManager, rpcClient.NetworkManager, appDB, mediaServer, feed, accountFeed, accountsDB, token.NewPersistence(db))
	tokenManager.Start()

	cryptoOnRampProviders := []onramp.Provider{
		onramp.NewMercuryoProvider(tokenManager),
		onramp.NewRampProvider(),
		onramp.NewMoonPayProvider(),
	}
	cryptoOnRampManager := onramp.NewManager(cryptoOnRampProviders)

	savedAddressesManager := &SavedAddressesManager{db: db}
	transactionManager := transfer.NewTransactionManager(transfer.NewMultiTransactionDB(db), gethManager, transactor, config, accountsDB, pendingTxManager, feed)
	blockChainState := blockchainstate.NewBlockChainState()
	transferController := transfer.NewTransferController(db, accountsDB, rpcClient, accountFeed, feed, transactionManager, pendingTxManager,
		tokenManager, balanceCacher, blockChainState)
	transferController.Start()
	cryptoCompare := cryptocompare.NewClient()
	coingecko := coingecko.NewClient()
	cryptoCompareProxy := cryptocompare.NewClientWithParams(cryptocompare.Params{
		ID:       fmt.Sprintf("%s-proxy", cryptoCompare.ID()),
		URL:      fmt.Sprintf("https://%s.api.status.im/cryptocompare/", statusProxyStageName),
		User:     config.WalletConfig.StatusProxyMarketUser,
		Password: config.WalletConfig.StatusProxyMarketPassword,
	})
	marketManager := market.NewManager([]thirdparty.MarketDataProvider{cryptoCompare, coingecko, cryptoCompareProxy}, feed)
	reader := NewReader(tokenManager, marketManager, token.NewPersistence(db), feed)
	history := history.NewService(db, accountsDB, accountFeed, feed, rpcClient, tokenManager, marketManager, balanceCacher.Cache())
	currency := currency.NewService(db, feed, tokenManager, marketManager)

	openseaHTTPClient := opensea.NewHTTPClient()
	openseaV2Client := opensea.NewClientV2(config.WalletConfig.OpenseaAPIKey, openseaHTTPClient)
	raribleClient := rarible.NewClient(config.WalletConfig.RaribleMainnetAPIKey, config.WalletConfig.RaribleTestnetAPIKey)
	alchemyClient := alchemy.NewClient(config.WalletConfig.AlchemyAPIKeys)

	// Collectible providers in priority order (i.e. provider N+1 will be tried only if provider N fails)
	contractOwnershipProviders := []thirdparty.CollectibleContractOwnershipProvider{
		raribleClient,
		alchemyClient,
	}

	accountOwnershipProviders := []thirdparty.CollectibleAccountOwnershipProvider{
		raribleClient,
		alchemyClient,
		openseaV2Client,
	}

	collectibleDataProviders := []thirdparty.CollectibleDataProvider{
		raribleClient,
		alchemyClient,
		openseaV2Client,
	}

	collectionDataProviders := []thirdparty.CollectionDataProvider{
		raribleClient,
		alchemyClient,
		openseaV2Client,
	}

	collectibleSearchProviders := []thirdparty.CollectibleSearchProvider{
		raribleClient,
	}

	collectibleProviders := thirdparty.CollectibleProviders{
		ContractOwnershipProviders: contractOwnershipProviders,
		AccountOwnershipProviders:  accountOwnershipProviders,
		CollectibleDataProviders:   collectibleDataProviders,
		CollectionDataProviders:    collectionDataProviders,
		SearchProviders:            collectibleSearchProviders,
	}

	collectiblesManager := collectibles.NewManager(
		db,
		rpcClient,
		communityManager,
		collectibleProviders,
		mediaServer,
		feed,
	)
	collectibles := collectibles.NewService(db, feed, accountsDB, accountFeed, settingsFeed, communityManager, rpcClient.NetworkManager, collectiblesManager)

	activity := activity.NewService(db, accountsDB, tokenManager, collectiblesManager, feed, pendingTxManager)

	featureFlags := &protocolCommon.FeatureFlags{}
	if config.WalletConfig.EnableCelerBridge {
		featureFlags.EnableCelerBridge = true
	}

	router := router.NewRouter(rpcClient, transactor, tokenManager, marketManager, collectibles,
		collectiblesManager, ens, stickers)
	pathProcessors := buildPathProcessors(rpcClient, transactor, tokenManager, ens, stickers, featureFlags)
	for _, processor := range pathProcessors {
		router.AddPathProcessor(processor)
	}

	routeExecutionManager := routeexecution.NewManager(router, transactionManager, transferController)

	return &Service{
		db:                    db,
		accountsDB:            accountsDB,
		rpcClient:             rpcClient,
		tokenManager:          tokenManager,
		communityManager:      communityManager,
		savedAddressesManager: savedAddressesManager,
		transactionManager:    transactionManager,
		pendingTxManager:      pendingTxManager,
		transferController:    transferController,
		cryptoOnRampManager:   cryptoOnRampManager,
		collectiblesManager:   collectiblesManager,
		collectibles:          collectibles,
		gethManager:           gethManager,
		marketManager:         marketManager,
		transactor:            transactor,
		ens:                   ens,
		stickers:              stickers,
		feed:                  feed,
		signals:               signals,
		reader:                reader,
		history:               history,
		currency:              currency,
		activity:              activity,
		decoder:               NewDecoder(),
		blockChainState:       blockChainState,
		keycardPairings:       NewKeycardPairings(),
		config:                config,
		featureFlags:          featureFlags,
		router:                router,
		routeExecutionManager: routeExecutionManager,
	}
}

func buildPathProcessors(
	rpcClient *rpc.Client,
	transactor *transactions.Transactor,
	tokenManager *token.Manager,
	ens *ens.Service,
	stickers *stickers.Service,
	featureFlags *protocolCommon.FeatureFlags,
) []pathprocessor.PathProcessor {
	ret := make([]pathprocessor.PathProcessor, 0)

	transfer := pathprocessor.NewTransferProcessor(rpcClient, transactor)
	ret = append(ret, transfer)

	erc721Transfer := pathprocessor.NewERC721Processor(rpcClient, transactor)
	ret = append(ret, erc721Transfer)

	erc1155Transfer := pathprocessor.NewERC1155Processor(rpcClient, transactor)
	ret = append(ret, erc1155Transfer)

	hop := pathprocessor.NewHopBridgeProcessor(rpcClient, transactor, tokenManager, rpcClient.NetworkManager)
	ret = append(ret, hop)

	if featureFlags.EnableCelerBridge {
		// TODO: Celar Bridge is out of scope for 2.30, check it thoroughly once we decide to include it again
		cbridge := pathprocessor.NewCelerBridgeProcessor(rpcClient, transactor, tokenManager)
		ret = append(ret, cbridge)
	}

	paraswap := pathprocessor.NewSwapParaswapProcessor(rpcClient, transactor, tokenManager)
	ret = append(ret, paraswap)

	ensRegister := pathprocessor.NewENSRegisterProcessor(rpcClient, transactor, ens)
	ret = append(ret, ensRegister)

	ensRelease := pathprocessor.NewENSReleaseProcessor(rpcClient, transactor, ens)
	ret = append(ret, ensRelease)

	return ret
}

// Service is a wallet service.
type Service struct {
	db                    *sql.DB
	accountsDB            *accounts.Database
	rpcClient             *rpc.Client
	savedAddressesManager *SavedAddressesManager
	tokenManager          *token.Manager
	communityManager      *community.Manager
	transactionManager    *transfer.TransactionManager
	pendingTxManager      *transactions.PendingTxTracker
	cryptoOnRampManager   *onramp.Manager
	transferController    *transfer.Controller
	marketManager         *market.Manager
	started               bool
	collectiblesManager   *collectibles.Manager
	collectibles          *collectibles.Service
	gethManager           *account.GethManager
	transactor            *transactions.Transactor
	ens                   *ens.Service
	stickers              *stickers.Service
	feed                  *event.Feed
	signals               *walletevent.SignalsTransmitter
	reader                *Reader
	history               *history.Service
	currency              *currency.Service
	activity              *activity.Service
	decoder               *Decoder
	blockChainState       *blockchainstate.BlockChainState
	keycardPairings       *KeycardPairings
	config                *params.NodeConfig
	featureFlags          *protocolCommon.FeatureFlags
	router                *router.Router
	routeExecutionManager *routeexecution.Manager
}

// Start signals transmitter.
func (s *Service) Start() error {
	s.transferController.Start()
	s.currency.Start()
	err := s.signals.Start()
	s.history.Start()
	s.collectibles.Start()
	s.started = true
	return err
}

// Set external Collectibles community info provider
func (s *Service) SetWalletCommunityInfoProvider(provider thirdparty.CommunityInfoProvider) {
	s.communityManager.SetCommunityInfoProvider(provider)
}

// Stop reactor and close db.
func (s *Service) Stop() error {
	logutils.ZapLogger().Info("wallet will be stopped")
	s.router.Stop()
	s.signals.Stop()
	s.transferController.Stop()
	s.currency.Stop()
	s.reader.Stop()
	s.history.Stop()
	s.activity.Stop()
	s.collectibles.Stop()
	s.tokenManager.Stop()
	s.started = false
	logutils.ZapLogger().Info("wallet stopped")
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

// Protocols returns list of p2p protocols.
func (s *Service) Protocols() []p2p.Protocol {
	return nil
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

func (s *Service) GetMarketManager() *market.Manager {
	return s.marketManager
}

func (s *Service) GetCollectiblesService() *collectibles.Service {
	return s.collectibles
}

func (s *Service) GetCollectiblesManager() *collectibles.Manager {
	return s.collectiblesManager
}

func (s *Service) GetEnsService() *ens.Service {
	return s.ens
}

func (s *Service) GetStickersService() *stickers.Service {
	return s.stickers
}
