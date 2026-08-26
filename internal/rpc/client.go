package rpc

//go:generate go tool mockgen -package=mock_rpcclient -source=client.go -destination=mock/client/client.go

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"

	healthmanager "github.com/status-im/status-go/internal/healthmanager"
	"github.com/status-im/status-go/internal/logutils"
	"github.com/status-im/status-go/internal/panics"
	"github.com/status-im/status-go/internal/pausable"
	"github.com/status-im/status-go/internal/platform"
	chain "github.com/status-im/status-go/internal/rpc/chain"
	ethclient "github.com/status-im/status-go/internal/rpc/chain/ethclient"
	"github.com/status-im/status-go/internal/rpc/chain/rpclimiter"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/pkg/pubsub"
	"github.com/status-im/status-go/pkg/services/networks"
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
	if platform.IsMobilePlatform() {
		rpcUserAgentName = fmt.Sprintf(rpcUserAgentFormat, mobile, version.Version())
	} else {
		rpcUserAgentName = fmt.Sprintf(rpcUserAgentFormat, desktop, version.Version())
	}
}

type ClientInterface interface {
	EthClientGetter
	EthClientWithProvider(chainID uint64, provider string) (ethclient.EthClientInterface, error)
	GetNetworkManager() networks.ManagerInterface
}

type EthClientGetter interface {
	EthClient(chainID uint64) (ethclient.EthClientInterface, error)
}

// Client manages RPC clients for multiple chains with
// multiple providers for each chain.
type Client struct {
	pausable.PauseBroadcaster

	rpcClientsMutex    sync.RWMutex
	rpcClients         map[uint64]chain.ClientInterface
	rpsLimiterMutex    sync.RWMutex
	limiterPerProvider map[string]*rpclimiter.RPCRpsLimiter

	networkManager networks.ManagerInterface

	healthMgr          *healthmanager.BlockchainHealthManager
	stopMonitoringFunc context.CancelFunc
	accountsPublisher  *pubsub.Publisher
	signalsTransmitter *SignalsTransmitter

	logger *zap.Logger
}

// Is initialized in a build-tag-dependent module
var verifProxyInitFn func(c *Client)

// ClientConfig holds the configuration for initializing a new Client.
type ClientConfig struct {
	NetworkManager    networks.ManagerInterface
	AccountsPublisher *pubsub.Publisher
}

// NewClient initializes Client
//
// Client is safe for concurrent use and will automatically
// reconnect to the server if connection is lost.
func NewClient(config ClientConfig) (*Client, error) {
	logger := logutils.ZapLogger().Named("rpcClient")
	if config.NetworkManager == nil {
		return nil, errors.New("network manager is required")
	}

	c := Client{
		networkManager:     config.NetworkManager,
		rpcClients:         make(map[uint64]chain.ClientInterface),
		limiterPerProvider: make(map[string]*rpclimiter.RPCRpsLimiter),
		logger:             logger,
		healthMgr:          healthmanager.NewBlockchainHealthManager(),
		accountsPublisher:  config.AccountsPublisher,
		signalsTransmitter: NewSignalsTransmitter(config.NetworkManager.GetPublisher()),
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

	if c.stopMonitoringFunc != nil {
		c.logger.Warn("Blockchain health manager already started")
		return
	}

	cancelableCtx, cancel := context.WithCancel(ctx)
	c.stopMonitoringFunc = cancel
	statusCh := c.healthMgr.Subscribe()
	go func() {
		defer panics.LogOnPanic()
		c.monitorHealth(cancelableCtx, statusCh)
	}()
}

func (c *Client) Stop() {
	c.signalsTransmitter.Stop()

	c.rpsLimiterMutex.Lock()
	for key, limiter := range c.limiterPerProvider {
		if limiter != nil {
			limiter.Stop()
		}
		delete(c.limiterPerProvider, key)
	}
	c.rpsLimiterMutex.Unlock()

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

func (c *Client) GetHealthManagerFullStatus() healthmanager.BlockchainFullStatus {
	return c.healthMgr.GetFullStatus()
}

func (c *Client) GetNetworkManager() networks.ManagerInterface {
	return c.networkManager
}

func (c *Client) GetNetworksPublisher() *pubsub.Publisher {
	if c.networkManager == nil {
		return nil
	}
	return c.networkManager.GetPublisher()
}

func (c *Client) getClientUsingCache(chainID uint64) (chain.ClientInterface, error) {
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

	phm := healthmanager.NewProvidersHealthManager(chainID)
	if c.IsPaused() {
		phm.Pause()
	}
	err := c.healthMgr.RegisterProvidersHealthManager(context.Background(), phm)
	if err != nil {
		return nil, fmt.Errorf("register providers health manager: %s", err)
	}

	client := chain.NewClient(ethClients, chainID, phm)
	c.rpcClients[chainID] = client
	return client, nil
}

// getProviderRPCLimiter returns the (optional) RPS limiter for the provider and the circuit
// name to use for the circuit breaker.
// The circuit name is always non-empty and isolated per chain+host, so the circuit breaker can
// short-circuit a consistently failing endpoint without affecting other chains that share the
// same host (e.g. the smart proxy). The RPS limiter, in contrast, is intentionally shared per
// host so we throttle the whole host regardless of chain.
func (c *Client) getProviderRPCLimiter(provider params.RpcProvider) (*rpclimiter.RPCRpsLimiter, string, error) {
	circuitName := fmt.Sprintf("%s-%d", provider.GetHost(), provider.ChainID)

	c.rpsLimiterMutex.Lock()
	defer c.rpsLimiterMutex.Unlock()
	if !provider.EnableRPSLimiter {
		return nil, circuitName, nil
	}
	// Generate a unique key for the limiter based on its host (shared across chains)
	limiterKey := provider.GetHost()

	// Check if the limiter already exists
	if limiter, ok := c.limiterPerProvider[limiterKey]; ok {
		return limiter, circuitName, nil
	}

	limiter := rpclimiter.NewRPCRpsLimiter(&c.PauseBroadcaster)
	c.limiterPerProvider[limiterKey] = limiter
	return limiter, circuitName, nil
}

// SetPaused stops/resumes the 1s tickers of every RPC rate limiter owned by this Client.
func (c *Client) SetPaused(paused bool) {
	if paused {
		c.MarkPaused()
		c.healthMgr.Pause()
	} else {
		c.MarkResumed()
		c.healthMgr.Resume()
	}
}

func (c *Client) getEthClients(network *params.Network) []ethclient.RPSLimitedEthClientInterface {
	var ethClients []ethclient.RPSLimitedEthClientInterface

	// Iterate over providers in order
	for _, provider := range network.RpcProviders {
		// Skip disabled providers
		if !provider.Enabled {
			continue
		}

		rpcClient, err := chain.CreateEthClientFromProvider(provider, rpcUserAgentName)
		if err != nil {
			c.logger.Error("create eth client failed", zap.String("provider", provider.Name), zap.Error(err))
			continue
		}
		if rpcClient == nil {
			// Provider is disabled
			continue
		}

		rpcLimiter, circuitName, err := c.getProviderRPCLimiter(provider)
		if err != nil {
			c.logger.Error("get RPC limiter failed", zap.String("provider", provider.Name), zap.Error(err))
			continue
		}

		// Create ethclient with RPS limiter. If limiter is not enabled, it will be nil
		ethClient := ethclient.NewRPSLimitedEthClient(rpcClient, rpcLimiter, circuitName, provider.Name)
		ethClients = append(ethClients, ethClient)
	}

	return ethClients
}

// EthClient returns ethclient.Client per chain
func (c *Client) EthClient(chainID uint64) (ethclient.EthClientInterface, error) {
	client, err := c.getClientUsingCache(chainID)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func (c *Client) EthClientWithProvider(chainID uint64, provider string) (ethclient.EthClientInterface, error) {
	client, err := c.getClientUsingCache(chainID)
	if err != nil {
		return nil, err
	}

	return client.GetProviderClient(provider), nil
}
