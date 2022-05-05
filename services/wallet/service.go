package wallet

import (
	"database/sql"

	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
	gethrpc "github.com/ethereum/go-ethereum/rpc"

	"github.com/status-im/status-go/account"
	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/rpc"
	"github.com/status-im/status-go/services/wallet/transfer"
)

// NewService initializes service instance.
func NewService(db *sql.DB, accountsDB *accounts.Database, rpcClient *rpc.Client, accountFeed *event.Feed, openseaAPIKey string, gethManager *account.GethManager) *Service {
	cryptoOnRampManager := NewCryptoOnRampManager(&CryptoOnRampOptions{
		dataSourceType: DataSourceStatic,
	})
	tokenManager := &TokenManager{db: db, RPCClient: rpcClient, networkManager: rpcClient.NetworkManager}
	savedAddressesManager := &SavedAddressesManager{db: db}
	transactionManager := &TransactionManager{db: db}
	favouriteManager := &FavouriteManager{db: db}
	transferController := transfer.NewTransferController(db, rpcClient, accountFeed)

	return &Service{
		db:                    db,
		accountsDB:            accountsDB,
		rpcClient:             rpcClient,
		favouriteManager:      favouriteManager,
		tokenManager:          tokenManager,
		savedAddressesManager: savedAddressesManager,
		transactionManager:    transactionManager,
		transferController:    transferController,
		cryptoOnRampManager:   cryptoOnRampManager,
		openseaAPIKey:         openseaAPIKey,
		feesManager:           &FeeManager{rpcClient},
		gethManager:           gethManager,
	}
}

// Service is a wallet service.
type Service struct {
	db                    *sql.DB
	accountsDB            *accounts.Database
	rpcClient             *rpc.Client
	savedAddressesManager *SavedAddressesManager
	tokenManager          *TokenManager
	transactionManager    *TransactionManager
	favouriteManager      *FavouriteManager
	cryptoOnRampManager   *CryptoOnRampManager
	transferController    *transfer.Controller
	feesManager           *FeeManager
	started               bool
	openseaAPIKey         string
	gethManager           *account.GethManager
}

// Start signals transmitter.
func (s *Service) Start() error {
	err := s.transferController.Start()
	s.started = true
	return err
}

// GetFeed returns signals feed.
func (s *Service) GetFeed() *event.Feed {
	return s.transferController.TransferFeed
}

// Stop reactor, signals transmitter and close db.
func (s *Service) Stop() error {
	log.Info("wallet will be stopped")
	s.transferController.Stop()
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
