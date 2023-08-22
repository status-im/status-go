package collectibles

import (
	"database/sql"

	"github.com/ethereum/go-ethereum/p2p"
	ethRpc "github.com/ethereum/go-ethereum/rpc"
	"github.com/status-im/status-go/account"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/rpc"
	"github.com/status-im/status-go/transactions"
)

type ServiceInterface interface {
	GetCollectibleContractData(chainID uint64, contractAddress string) (*CollectibleContractData, error)
	GetAssetContractData(chainID uint64, contractAddress string) (*AssetContractData, error)
}

// Collectibles service
type Service struct {
	manager         *Manager
	accountsManager *account.GethManager
	pendingTracker  *transactions.PendingTxTracker
	config          *params.NodeConfig
	db              *Database
}

// Returns a new Collectibles Service.
func NewService(rpcClient *rpc.Client, accountsManager *account.GethManager, pendingTracker *transactions.PendingTxTracker, config *params.NodeConfig, appDb *sql.DB) *Service {
	return &Service{
		manager:         &Manager{rpcClient: rpcClient},
		accountsManager: accountsManager,
		pendingTracker:  pendingTracker,
		config:          config,
		db:              NewCommunityTokensDatabase(appDb),
	}
}

// Protocols returns a new protocols list. In this case, there are none.
func (s *Service) Protocols() []p2p.Protocol {
	return []p2p.Protocol{}
}

// APIs returns a list of new APIs.
func (s *Service) APIs() []ethRpc.API {
	return []ethRpc.API{
		{
			Namespace: "collectibles",
			Version:   "0.1.0",
			Service:   NewAPI(s),
			Public:    true,
		},
	}
}

// Start is run when a service is started.
func (s *Service) Start() error {
	return nil
}

// Stop is run when a service is stopped.
func (s *Service) Stop() error {
	return nil
}

func (s *Service) GetCollectibleContractData(chainID uint64, contractAddress string) (*CollectibleContractData, error) {
	return s.manager.GetCollectibleContractData(chainID, contractAddress)
}

func (s *Service) GetAssetContractData(chainID uint64, contractAddress string) (*AssetContractData, error) {
	return s.manager.GetAssetContractData(chainID, contractAddress)
}
