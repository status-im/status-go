package wallet

import (
	"database/sql"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/status-im/status-go/services/wallet/network"
	"github.com/status-im/status-go/services/wallet/transfer"
)

// NewService initializes service instance.
func NewService(db *sql.DB, legacyChainID uint64, legacyClient *ethclient.Client, networks []network.Network, accountFeed *event.Feed) *Service {
	cryptoOnRampManager := NewCryptoOnRampManager(&CryptoOnRampOptions{
		dataSourceType: DataSourceStatic,
	})
	tokenManager := &TokenManager{db: db}
	savedAddressesManager := &SavedAddressesManager{db: db}
	transactionManager := &TransactionManager{db: db}
	favouriteManager := &FavouriteManager{db: db}
	networkManager := network.NewManager(db, legacyChainID, legacyClient)
	err := networkManager.Init(networks)
	if err != nil {
		log.Error("Network manager failed to initialize", "error", err)
	}

	transferController := transfer.NewTransferController(db, networkManager, accountFeed)

	return &Service{
		favouriteManager:      favouriteManager,
		networkManager:        networkManager,
		tokenManager:          tokenManager,
		savedAddressesManager: savedAddressesManager,
		transactionManager:    transactionManager,
		transferController:    transferController,
		cryptoOnRampManager:   cryptoOnRampManager,
		legacyChainID:         legacyChainID,
	}
}

// Service is a wallet service.
type Service struct {
	networkManager        *network.Manager
	savedAddressesManager *SavedAddressesManager
	tokenManager          *TokenManager
	transactionManager    *TransactionManager
	favouriteManager      *FavouriteManager
	cryptoOnRampManager   *CryptoOnRampManager
	transferController    *transfer.Controller
	legacyChainID         uint64
	started               bool
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
func (s *Service) APIs() []rpc.API {
	return []rpc.API{
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
