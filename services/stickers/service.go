package stickers

import (
	"context"

	ethRpc "github.com/ethereum/go-ethereum/rpc"

	accsmanagement "github.com/status-im/status-go/accounts-management"
	"github.com/status-im/status-go/ipfs"
	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/rpc"
	"github.com/status-im/status-go/services/media"
	"github.com/status-im/status-go/services/wallet/pendingtxtracker"
)

// NewService initializes service instance.
func NewService(acc *accounts.Database, rpcClient *rpc.Client, accountsManager *accsmanagement.AccountsManager,
	downloader *ipfs.Downloader, httpServer *media.Service, pendingTracker *pendingtxtracker.PendingTxTracker) *Service {
	ctx, cancel := context.WithCancel(context.Background())

	return &Service{
		accountsDB:      acc,
		rpcClient:       rpcClient,
		accountsManager: accountsManager,
		downloader:      downloader,
		httpServer:      httpServer,
		ctx:             ctx,
		cancel:          cancel,
		api:             NewAPI(ctx, acc, rpcClient, accountsManager, pendingTracker, downloader, httpServer),
	}
}

// Service is a browsers service.
type Service struct {
	accountsDB      *accounts.Database
	rpcClient       *rpc.Client
	accountsManager *accsmanagement.AccountsManager
	downloader      *ipfs.Downloader
	httpServer      *media.Service
	ctx             context.Context
	cancel          context.CancelFunc
	api             *API
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

func (s *Service) API() *API {
	return s.api
}

// APIs returns list of available RPC APIs.
func (s *Service) APIs() []ethRpc.API {
	return []ethRpc.API{
		{
			Namespace: "stickers",
			Version:   "0.1.0",
			Service:   s.api,
		},
	}
}
