package rpc

//go:generate go tool mockgen -package=mock_rpcclient -source=client.go -destination=mock/client/client.go

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"

	appCommon "github.com/status-im/status-go/common"
	healthmanager2 "github.com/status-im/status-go/internal/healthmanager"
	"github.com/status-im/status-go/internal/logutils"
	chain2 "github.com/status-im/status-go/internal/rpc/chain"
	ethclient2 "github.com/status-im/status-go/internal/rpc/chain/ethclient"
	"github.com/status-im/status-go/internal/rpc/chain/rpclimiter"
	"github.com/status-im/status-go/internal/rpc/network"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/pkg/pubsub"
	"github.com/status-im/status-go/pkg/version"
)

const (
	// DefaultCallTimeout is a default timeout for an RPC call
	DefaultCallTimeout = time.Minute

	mobile  = "mobile"
	desktop = "desktop"

	// rpcUserAgentFormat 'procurator': *an agent representing others*, aka a "proxy"
	// allows for the rpc client to have a dedicated user agent, which is useful for the proxy server logs.
	rpcUserAgentFormat = "procuratee-%s/%s"
)

var (
	// rpcUserAgentName the user agent
	rpcUserAgentName = fmt.Sprintf(rpcUserAgentFormat, "no-GOOS", version.Version())
)

func init() {
	if appCommon.IsMobilePlatform() {
		rpcUserAgentName = fmt.Sprintf(rpcUserAgentFormat, mobile, version.Version())
	} else {
		rpcUserAgentName = fmt.Sprintf(rpcUserAgentFormat, desktop, version.Version())
	}
}

type ClientInterface interface {
	EthClientGetter
	EthClientWithProvider(chainID uint64, provider string) (ethclient2.EthClientInterface, error)
	GetNetworkManager() *network.Manager
}

type EthClientGetter interface {
	EthClient(chainID uint64) (ethclient2.EthClientInterface, error)
}

// Client manages RPC clients for multiple chains with
// multiple providers for each chain.
type Client struct {
	rpcClientsMutex    sync.RWMutex
	rpcClients         map[uint64]chain2.ClientInterface
	rpsLimiterMutex    sync.RWMutex
	limiterPerProvider map[string]*rpclimiter.RPCRpsLimiter

	networkManager *network.Manager

	healthMgr          *healthmanager2.BlockchainHealthManager
	stopMonitoringFunc context.CancelFunc
	accountsPublisher  *pubsub.Publisher
	signalsTransmitter *SignalsTransmitter

	logger *zap.Logger
}

// Is initialized in a build-tag-dependent module
var verifProxyInitFn func(c *Client)

// ClientConfig holds the configuration for initializing a new Client.
type ClientConfig struct {
	Networks          []params.Network
	DB                *sql.DB
	AccountsPublisher *pubsub.Publisher
}

// NewClient initializes Client
//
// Client is safe for concurrent use and will automatically
// reconnect to the server if connection is lost.
func NewClient(config ClientConfig) (*Client, error) {
	logger := logutils.ZapLogger().Named("rpcClient")
	networkManager := network.NewManager(config.DB, config.AccountsPublisher)
	if networkManager == nil {
		return nil, errors.New("failed to create network manager")
	}

	err := networkManager.InitEmbeddedNetworks(config.Networks)
	if err != nil {
		logger.Error("Network manager failed to initialize", zap.Error(err))
		return nil, err
	}

	c := Client{
		networkManager:     networkManager,
		rpcClients:         make(map[uint64]chain2.ClientInterface),
		limiterPerProvider: make(map[string]*rpclimiter.RPCRpsLimiter),
		logger:             logger,
		healthMgr:          healthmanager2.NewBlockchainHealthManager(),
		accountsPublisher:  config.AccountsPublisher,
		signalsTransmitter: NewSignalsTransmitter(networkManager.GetPublisher()),
	}

	if verifProxyInitFn != nil {
		verifProxyInitFn(&c)
	}

	return &c, nil
}

func (c *Client) Start(ctx context.Context) {
	if err := c.signalsTransmitter.Start(); err != nil {
		c.logger.Error("Failed to start signals transmitter", zap.Error(err))
	}
	c.networkManager.Start()

	if c.stopMonitoringFunc != nil {
		c.logger.Warn("Blockchain health manager already started")
		return
	}

	cancelableCtx, cancel := context.WithCancel(ctx)
	c.stopMonitoringFunc = cancel
	statusCh := c.healthMgr.Subscribe()
	go func() {
		defer appCommon.LogOnPanic()
		c.monitorHealth(cancelableCtx, statusCh)
	}()
}

func (c *Client) Stop() {
	c.signalsTransmitter.Stop()
	c.networkManager.Stop()

	c.rpcClientsMutex.Lock()
	for _, client := range c.rpcClients {
		client.Close()
	}
	c.rpcClientsMutex.Unlock()

	c.healthMgr.Stop()
	if c.stopMonitoringFunc == nil {
		return
	}
	c.stopMonitoringFunc()
	c.stopMonitoringFunc = nil
}

func (c *Client) monitorHealth(ctx context.Context, statusCh chan struct{}) {
	sendFullStatusEventFunc := func() {
		publisher := c.GetNetworksPublisher()
		if publisher == nil {
			return
		}
		pubsub.Publish(publisher, EventBlockchainHealthChanged{
			FullStatus: c.healthMgr.GetFullStatus(),
		})
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-statusCh:
			sendFullStatusEventFunc()
		}
	}
}

func (c *Client) GetHealthManagerFullStatus() healthmanager2.BlockchainFullStatus {
	return c.healthMgr.GetFullStatus()
}

func (c *Client) GetNetworkManager() *network.Manager {
	return c.networkManager
}

func (c *Client) GetNetworksPublisher() *pubsub.Publisher {
	if c.networkManager == nil {
		return nil
	}
	return c.networkManager.GetPublisher()
}

func (c *Client) getClientUsingCache(chainID uint64) (chain2.ClientInterface, error) {
	c.rpcClientsMutex.Lock()
	defer c.rpcClientsMutex.Unlock()
	if rpcClient, ok := c.rpcClients[chainID]; ok {
		return rpcClient, nil
	}

	network := c.networkManager.Find(chainID)
	if network == nil {
		return nil, fmt.Errorf("could not find network: %d", chainID)
	}

	ethClients := c.getEthClients(network)
	if len(ethClients) == 0 {
		return nil, fmt.Errorf("could not find any enabled RPC providers for chain: %d", chainID)
	}

	phm := healthmanager2.NewProvidersHealthManager(chainID)
	err := c.healthMgr.RegisterProvidersHealthManager(context.Background(), phm)
	if err != nil {
		return nil, fmt.Errorf("register providers health manager: %s", err)
	}

	client := chain2.NewClient(ethClients, chainID, phm)
	c.rpcClients[chainID] = client
	return client, nil
}

func (c *Client) getProviderRPCLimiter(provider params.RpcProvider) (*rpclimiter.RPCRpsLimiter, string, error) {
	c.rpsLimiterMutex.Lock()
	defer c.rpsLimiterMutex.Unlock()
	if !provider.EnableRPSLimiter {
		return nil, "", nil
	}
	// Generate a unique key for the provider based on its host
	limiterKey := provider.GetHost()

	// Check if the limiter already exists
	if limiter, ok := c.limiterPerProvider[limiterKey]; ok {
		return limiter, limiterKey, nil
	}

	// Create a new limiter and store it
	limiter := rpclimiter.NewRPCRpsLimiter()
	c.limiterPerProvider[limiterKey] = limiter
	return limiter, limiterKey, nil
}

func (c *Client) getEthClients(network *params.Network) []ethclient2.RPSLimitedEthClientInterface {
	var ethClients []ethclient2.RPSLimitedEthClientInterface

	// Iterate over providers in order
	for _, provider := range network.RpcProviders {
		// Skip disabled providers
		if !provider.Enabled {
			continue
		}

		rpcClient, err := chain2.CreateEthClientFromProvider(provider, rpcUserAgentName)
		if err != nil {
			c.logger.Error("create eth client failed", zap.String("provider", provider.Name), zap.Error(err))
			continue
		}
		if rpcClient == nil {
			// Provider is disabled
			continue
		}

		rpcLimiter, limiterKey, err := c.getProviderRPCLimiter(provider)
		if err != nil {
			c.logger.Error("get RPC limiter failed", zap.String("provider", provider.Name), zap.Error(err))
			continue
		}

		// Create ethclient with RPS limiter. If limiter is not enabled, it will be nil
		ethClient := ethclient2.NewRPSLimitedEthClient(rpcClient, rpcLimiter, limiterKey, provider.Name)
		ethClients = append(ethClients, ethClient)
	}

	return ethClients
}

// EthClient returns ethclient.Client per chain
func (c *Client) EthClient(chainID uint64) (ethclient2.EthClientInterface, error) {
	client, err := c.getClientUsingCache(chainID)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func (c *Client) EthClientWithProvider(chainID uint64, provider string) (ethclient2.EthClientInterface, error) {
	client, err := c.getClientUsingCache(chainID)
	if err != nil {
		return nil, err
	}

	return client.GetProviderClient(provider), nil
}
