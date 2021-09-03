package wallet

import (
	"database/sql"

	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/status-im/status-go/services/wallet/network"
	"github.com/status-im/status-go/services/wallet/transfer"
)

// NewService initializes service instance.
func NewService(db *sql.DB, chainID uint64, feed *event.Feed) *Service {
	cryptoOnRampManager := NewCryptoOnRampManager(&CryptoOnRampOptions{
		dataSourceType: DataSourceStatic,
	})
	tokenManager := &TokenManager{db: db}
	transactionManager := &TransactionManager{db: db}
	favouriteManager := &FavouriteManager{db: db}
	networkManager := network.NewManager(db)
	err := networkManager.Init()
	if err != nil {
		log.Error("Network manager failed to initialize", "error", err)
	}

	transferController := transfer.NewTransferController(db, networkManager, feed)

	return &Service{
		favouriteManager:    favouriteManager,
		networkManager:      networkManager,
		tokenManager:        tokenManager,
		transactionManager:  transactionManager,
		transferController:  transferController,
		opensea:             newOpenseaClient(),
		cryptoOnRampManager: cryptoOnRampManager,
		legacyChainID:       chainID,
	}
}

// Service is a wallet service.
type Service struct {
	networkManager      *network.Manager
	tokenManager        *TokenManager
	transactionManager  *TransactionManager
	favouriteManager    *FavouriteManager
	cryptoOnRampManager *CryptoOnRampManager
	transferController  *transfer.Controller
	opensea             *OpenseaClient
	legacyChainID       uint64
	started             bool
}

// Start signals transmitter.
func (s *Service) Start() error {
	err := s.transferController.Start()
	s.started = true
	return err
}

// GetFeed returns signals feed.
func (s *Service) GetFeed() *event.Feed {
	return s.transferController.Feed
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
