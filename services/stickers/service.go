package stickers

import (
	"context"

	"github.com/ethereum/go-ethereum/p2p"
	ethRpc "github.com/ethereum/go-ethereum/rpc"
	"github.com/status-im/status-go/account"
	"github.com/status-im/status-go/ipfs"
	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/rpc"
	"github.com/status-im/status-go/server"
	"github.com/status-im/status-go/services/rpcfilters"
)

// NewService initializes service instance.
func NewService(acc *accounts.Database, rpcClient *rpc.Client, accountsManager *account.GethManager, rpcFiltersSrvc *rpcfilters.Service, config *params.NodeConfig, downloader *ipfs.Downloader, httpServer *server.Server) *Service {
	ctx, cancel := context.WithCancel(context.Background())

	return &Service{
		accountsDB:      acc,
		rpcClient:       rpcClient,
		accountsManager: accountsManager,
		rpcFiltersSrvc:  rpcFiltersSrvc,
		keyStoreDir:     config.KeyStoreDir,
		downloader:      downloader,
		httpServer:      httpServer,
		ctx:             ctx,
		cancel:          cancel,
	}
}

// Service is a browsers service.
type Service struct {
	accountsDB      *accounts.Database
	rpcClient       *rpc.Client
	accountsManager *account.GethManager
	rpcFiltersSrvc  *rpcfilters.Service
	downloader      *ipfs.Downloader
	keyStoreDir     string
	httpServer      *server.Server
	ctx             context.Context
	cancel          context.CancelFunc
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
			Service:   NewAPI(s.ctx, s.accountsDB, s.rpcClient, s.accountsManager, s.rpcFiltersSrvc, s.keyStoreDir, s.downloader, s.httpServer),
		},
	}
}

// Protocols returns list of p2p protocols.
func (s *Service) Protocols() []p2p.Protocol {
	return nil
}
