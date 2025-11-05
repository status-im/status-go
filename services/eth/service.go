package eth

import (
	gethrpc "github.com/ethereum/go-ethereum/rpc"

	accounts "github.com/status-im/status-go/internal/accounts-management"
	"github.com/status-im/status-go/rpc"
)

const namespace = "eth"

type Service struct {
	client          *rpc.Client
	accountsManager *accounts.AccountsManager
}

func NewService(client *rpc.Client, accountsManager *accounts.AccountsManager) *Service {
	return &Service{
		client:          client,
		accountsManager: accountsManager,
	}
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
			Namespace: namespace,
			Version:   "0.1.0",
			Service:   NewAPI(s.client, s.accountsManager),
		},
	}
}
