package wallet

import (
	"database/sql"
	"encoding/json"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
	gethrpc "github.com/ethereum/go-ethereum/rpc"

	"github.com/status-im/status-go/account"
	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/rpc"
	"github.com/status-im/status-go/services/ens"
	"github.com/status-im/status-go/services/stickers"
	"github.com/status-im/status-go/services/wallet/activity"
	"github.com/status-im/status-go/services/wallet/collectibles"
	"github.com/status-im/status-go/services/wallet/currency"
	"github.com/status-im/status-go/services/wallet/history"
	"github.com/status-im/status-go/services/wallet/market"
	"github.com/status-im/status-go/services/wallet/thirdparty"
	"github.com/status-im/status-go/services/wallet/thirdparty/alchemy"
	"github.com/status-im/status-go/services/wallet/thirdparty/coingecko"
	"github.com/status-im/status-go/services/wallet/thirdparty/cryptocompare"
	"github.com/status-im/status-go/services/wallet/thirdparty/infura"
	"github.com/status-im/status-go/services/wallet/thirdparty/opensea"
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
	rpcClient *rpc.Client,
	accountFeed *event.Feed,
	gethManager *account.GethManager,
	transactor *transactions.Transactor,
	config *params.NodeConfig,
	ens *ens.Service,
	stickers *stickers.Service,
	pendingTxManager *transactions.PendingTxTracker,
	feed *event.Feed,
) *Service {
	cryptoOnRampManager := NewCryptoOnRampManager(&CryptoOnRampOptions{
		dataSourceType: DataSourceStatic,
	})

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
	tokenManager := token.NewTokenManager(db, rpcClient, rpcClient.NetworkManager)
	savedAddressesManager := &SavedAddressesManager{db: db}
	transactionManager := transfer.NewTransactionManager(db, gethManager, transactor, config, accountsDB, pendingTxManager, feed)
	transferController := transfer.NewTransferController(db, rpcClient, accountFeed, feed, transactionManager, pendingTxManager,
		tokenManager, config.WalletConfig.LoadAllTransfers)
	cryptoCompare := cryptocompare.NewClient()
	coingecko := coingecko.NewClient()
	marketManager := market.NewManager(cryptoCompare, coingecko, feed)
	reader := NewReader(rpcClient, tokenManager, marketManager, accountsDB, NewPersistence(db), feed)
	history := history.NewService(db, feed, rpcClient, tokenManager, marketManager)
	currency := currency.NewService(db, feed, tokenManager, marketManager)
	activity := activity.NewService(db, tokenManager, feed, accountsDB)

	openseaHTTPClient := opensea.NewHTTPClient()
	openseaClient := opensea.NewClient(config.WalletConfig.OpenseaAPIKey, openseaHTTPClient, feed)
	openseaV2Client := opensea.NewClientV2(config.WalletConfig.OpenseaAPIKey, openseaHTTPClient, feed)
	infuraClient := infura.NewClient(config.WalletConfig.InfuraAPIKey, config.WalletConfig.InfuraAPIKeySecret)
	alchemyClient := alchemy.NewClient(config.WalletConfig.AlchemyAPIKeys)

	// Try OpenSea, Infura, Alchemy in that order
	contractOwnershipProviders := []thirdparty.CollectibleContractOwnershipProvider{
		infuraClient,
		alchemyClient,
	}

	accountOwnershipProviders := []thirdparty.CollectibleAccountOwnershipProvider{
		openseaClient,
		openseaV2Client,
		infuraClient,
		alchemyClient,
	}

	collectibleDataProviders := []thirdparty.CollectibleDataProvider{
		openseaClient,
		openseaV2Client,
		infuraClient,
		alchemyClient,
	}

	collectionDataProviders := []thirdparty.CollectionDataProvider{
		openseaClient,
		infuraClient,
		alchemyClient,
	}

	collectiblesManager := collectibles.NewManager(db, rpcClient, contractOwnershipProviders, accountOwnershipProviders, collectibleDataProviders, collectionDataProviders, openseaClient)
	collectibles := collectibles.NewService(db, feed, accountsDB, accountFeed, rpcClient.NetworkManager, collectiblesManager)
	return &Service{
		db:                    db,
		accountsDB:            accountsDB,
		rpcClient:             rpcClient,
		tokenManager:          tokenManager,
		savedAddressesManager: savedAddressesManager,
		transactionManager:    transactionManager,
		pendingTxManager:      pendingTxManager,
		transferController:    transferController,
		cryptoOnRampManager:   cryptoOnRampManager,
		collectiblesManager:   collectiblesManager,
		collectibles:          collectibles,
		feesManager:           &FeeManager{rpcClient},
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
	}
}

// Service is a wallet service.
type Service struct {
	db                    *sql.DB
	accountsDB            *accounts.Database
	rpcClient             *rpc.Client
	savedAddressesManager *SavedAddressesManager
	tokenManager          *token.Manager
	transactionManager    *transfer.TransactionManager
	pendingTxManager      *transactions.PendingTxTracker
	cryptoOnRampManager   *CryptoOnRampManager
	transferController    *transfer.Controller
	feesManager           *FeeManager
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

// Set external Collectibles metadata provider
func (s *Service) SetCollectibleMetadataProvider(provider thirdparty.CollectibleMetadataProvider) {
	s.collectiblesManager.SetMetadataProvider(provider)
}

// Stop reactor and close db.
func (s *Service) Stop() error {
	log.Info("wallet will be stopped")
	s.signals.Stop()
	s.transferController.Stop()
	s.currency.Stop()
	s.reader.Stop()
	s.history.Stop()
	s.activity.Stop()
	s.collectibles.Stop()
	s.started = false
	log.Info("wallet stopped")
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
