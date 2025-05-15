package web3provider

import (
	"database/sql"

	gethrpc "github.com/ethereum/go-ethereum/rpc"

	"github.com/status-im/status-go/v10/account"
	"github.com/status-im/status-go/v10/transactions"

	"github.com/status-im/status-go/v10/multiaccounts/accounts"

	"github.com/status-im/status-go/v10/params"
	"github.com/status-im/status-go/v10/rpc"
	"github.com/status-im/status-go/v10/services/permissions"
	"github.com/status-im/status-go/v10/services/rpcfilters"
)

func NewService(appDB *sql.DB, accountsDB *accounts.Database, rpcClient *rpc.Client, config *params.NodeConfig, accountsManager *account.GethManager, rpcFiltersSrvc *rpcfilters.Service, transactor *transactions.Transactor) *Service {
	return &Service{
		permissionsDB:   permissions.NewDB(appDB),
		accountsDB:      accountsDB,
		rpcClient:       rpcClient,
		rpcFiltersSrvc:  rpcFiltersSrvc,
		config:          config,
		accountsManager: accountsManager,
		transactor:      transactor,
	}
}

type Service struct {
	permissionsDB   *permissions.Database
	accountsDB      *accounts.Database
	rpcClient       *rpc.Client
	rpcFiltersSrvc  *rpcfilters.Service
	accountsManager *account.GethManager
	config          *params.NodeConfig
	transactor      *transactions.Transactor
}

func (s *Service) Start() error {
	return nil
}

func (s *Service) Stop() error {
	return nil
}

func (s *Service) APIs() []gethrpc.API {
	return []gethrpc.API{
		{
			Namespace: "provider",
			Version:   "0.1.0",
			Service:   NewAPI(s),
		},
	}
}
