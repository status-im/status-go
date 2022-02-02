package stickers

import (
	"context"
	"database/sql"

	"github.com/ethereum/go-ethereum/p2p"
	ethRpc "github.com/ethereum/go-ethereum/rpc"
	"github.com/status-im/status-go/account"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/rpc"
	"github.com/status-im/status-go/services/rpcfilters"
)

// NewService initializes service instance.
func NewService(appDB *sql.DB, rpcClient *rpc.Client, accountsManager *account.GethManager, rpcFiltersSrvc *rpcfilters.Service, config *params.NodeConfig) *Service {
	ctx, cancel := context.WithCancel(context.Background())

	return &Service{
		appDB:           appDB,
		rpcClient:       rpcClient,
		accountsManager: accountsManager,
		rpcFiltersSrvc:  rpcFiltersSrvc,
		config:          config,

		ctx:    ctx,
		cancel: cancel,
	}
}

// Service is a browsers service.
type Service struct {
	appDB           *sql.DB
	rpcClient       *rpc.Client
	accountsManager *account.GethManager
	rpcFiltersSrvc  *rpcfilters.Service
	config          *params.NodeConfig

	ctx    context.Context
	cancel context.CancelFunc
}

// Start a service.
func (s *Service) Start() error {
	return nil
}

// Stop a service.
func (s *Service) Stop() error {
	s.cancel()
	return nil
}

// APIs returns list of available RPC APIs.
func (s *Service) APIs() []ethRpc.API {
	return []ethRpc.API{
		{
			Namespace: "stickers",
			Version:   "0.1.0",
			Service:   NewAPI(s.ctx, s.appDB, s.rpcClient, s.accountsManager, s.rpcFiltersSrvc, s.config),
		},
	}
}

// Protocols returns list of p2p protocols.
func (s *Service) Protocols() []p2p.Protocol {
	return nil
}
