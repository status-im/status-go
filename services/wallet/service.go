package wallet

import (
	"database/sql"
	"encoding/json"
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
	nftMetadataProvider thirdparty.NFTMetadataProvider,
) *Service {
	cryptoOnRampManager := NewCryptoOnRampManager(&CryptoOnRampOptions{
		dataSourceType: DataSourceStatic,
	})
	walletFeed := &event.Feed{}
	signals := &walletevent.SignalsTransmitter{
		Publisher: walletFeed,
	}
	blockchainStatus := make(map[uint64]string)
	rpcClient.SetWalletNotifier(func(chainID uint64, message string) {
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

		walletFeed.Send(walletevent.Event{
			Type:     EventBlockchainStatusChanged,
			Accounts: []common.Address{},
			Message:  string(encodedmessage),
			At:       time.Now().Unix(),
			ChainID:  chainID,
		})
	})
	tokenManager := token.NewTokenManager(db, rpcClient, rpcClient.NetworkManager)
	savedAddressesManager := &SavedAddressesManager{db: db}
	transactionManager := transfer.NewTransactionManager(db, gethManager, transactor, config, accountsDB)
	transferController := transfer.NewTransferController(db, rpcClient, accountFeed, walletFeed, transactionManager, tokenManager, config.WalletConfig.LoadAllTransfers)
	cryptoCompare := cryptocompare.NewClient()
	coingecko := coingecko.NewClient()
	marketManager := market.NewManager(cryptoCompare, coingecko, walletFeed)
	reader := NewReader(rpcClient, tokenManager, marketManager, accountsDB, NewPersistence(db), walletFeed)
	history := history.NewService(db, walletFeed, rpcClient, tokenManager, marketManager)
	currency := currency.NewService(db, walletFeed, tokenManager, marketManager)
	activity := activity.NewService(db, walletFeed)

	alchemyClient := alchemy.NewClient(config.WalletConfig.AlchemyAPIKeys)
	infuraClient := infura.NewClient(config.WalletConfig.InfuraAPIKey, config.WalletConfig.InfuraAPIKeySecret)
	collectiblesManager := collectibles.NewManager(rpcClient, alchemyClient, infuraClient, nftMetadataProvider, config.WalletConfig.OpenseaAPIKey, walletFeed)
	return &Service{
		db:                    db,
		accountsDB:            accountsDB,
		rpcClient:             rpcClient,
		tokenManager:          tokenManager,
		savedAddressesManager: savedAddressesManager,
		transactionManager:    transactionManager,
		transferController:    transferController,
		cryptoOnRampManager:   cryptoOnRampManager,
		collectiblesManager:   collectiblesManager,
		feesManager:           &FeeManager{rpcClient},
		gethManager:           gethManager,
		marketManager:         marketManager,
		transactor:            transactor,
		ens:                   ens,
		stickers:              stickers,
		feed:                  walletFeed,
		signals:               signals,
		reader:                reader,
		history:               history,
		currency:              currency,
		activity:              activity,
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
	cryptoOnRampManager   *CryptoOnRampManager
	transferController    *transfer.Controller
	feesManager           *FeeManager
	marketManager         *market.Manager
	started               bool
	collectiblesManager   *collectibles.Manager
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
}

// Start signals transmitter.
func (s *Service) Start() error {
	s.transferController.Start()
	s.currency.Start()
	err := s.signals.Start()
	s.history.Start()
	s.started = true
	return err
}

// GetFeed returns signals feed.
func (s *Service) GetFeed() *event.Feed {
	return s.transferController.TransferFeed
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
