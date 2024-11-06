package rpc

//go:generate mockgen -package=mock_rpcclient -source=client.go -destination=mock/client/client.go

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"

	gethrpc "github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum/go-ethereum/event"
	appCommon "github.com/status-im/status-go/common"
	"github.com/status-im/status-go/healthmanager"
	"github.com/status-im/status-go/internal/version"
	"github.com/status-im/status-go/logutils"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/rpc/chain"
	"github.com/status-im/status-go/rpc/chain/ethclient"
	"github.com/status-im/status-go/rpc/chain/rpclimiter"
	"github.com/status-im/status-go/rpc/network"
	"github.com/status-im/status-go/services/rpcstats"
	"github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/walletevent"
)

const (
	// DefaultCallTimeout is a default timeout for an RPC call
	DefaultCallTimeout = time.Minute

	mobile  = "mobile"
	desktop = "desktop"

	// rpcUserAgentFormat 'procurator': *an agent representing others*, aka a "proxy"
	// allows for the rpc client to have a dedicated user agent, which is useful for the proxy server logs.
	rpcUserAgentFormat = "procuratee-%s/%s"

	// rpcUserAgentUpstreamFormat a separate user agent format for upstream, because we should not be using upstream
	// if we see this user agent in the logs that means parts of the application are using a malconfigured http client
	rpcUserAgentUpstreamFormat = "procuratee-%s-upstream/%s"

	EventBlockchainHealthChanged walletevent.EventType = "wallet-blockchain-health-changed" // Full status of the blockchain (including provider statuses)
)

// List of RPC client errors.
var (
	ErrMethodNotFound = fmt.Errorf("the method does not exist/is not available")
)

var (
	// rpcUserAgentName the user agent
	rpcUserAgentName         = fmt.Sprintf(rpcUserAgentFormat, "no-GOOS", version.Version())
	rpcUserAgentUpstreamName = fmt.Sprintf(rpcUserAgentUpstreamFormat, "no-GOOS", version.Version())
)

func init() {
	if appCommon.IsMobilePlatform() {
		rpcUserAgentName = fmt.Sprintf(rpcUserAgentFormat, mobile, version.Version())
		rpcUserAgentUpstreamName = fmt.Sprintf(rpcUserAgentUpstreamFormat, mobile, version.Version())
	} else {
		rpcUserAgentName = fmt.Sprintf(rpcUserAgentFormat, desktop, version.Version())
		rpcUserAgentUpstreamName = fmt.Sprintf(rpcUserAgentUpstreamFormat, desktop, version.Version())
	}
}

// Handler defines handler for RPC methods.
type Handler func(context.Context, uint64, ...interface{}) (interface{}, error)

type ClientInterface interface {
	AbstractEthClient(chainID common.ChainID) (ethclient.BatchCallClient, error)
	EthClient(chainID uint64) (chain.ClientInterface, error)
	EthClients(chainIDs []uint64) (map[uint64]chain.ClientInterface, error)
	CallContext(context context.Context, result interface{}, chainID uint64, method string, args ...interface{}) error
	Call(result interface{}, chainID uint64, method string, args ...interface{}) error
	CallRaw(body string) string
	GetNetworkManager() *network.Manager
}

// Client represents RPC client with custom routing
// scheme. It automatically decides where RPC call
// goes - Upstream or Local node.
type Client struct {
	sync.RWMutex

	UpstreamChainID uint64

	local              *gethrpc.Client
	rpcClientsMutex    sync.RWMutex
	rpcClients         map[uint64]chain.ClientInterface
	rpsLimiterMutex    sync.RWMutex
	limiterPerProvider map[string]*rpclimiter.RPCRpsLimiter

	router         *router
	NetworkManager *network.Manager

	healthMgr          *healthmanager.BlockchainHealthManager
	stopMonitoringFunc context.CancelFunc
	walletFeed         *event.Feed

	handlersMx sync.RWMutex       // mx guards handlers
	handlers   map[string]Handler // locally registered handlers
	logger     *zap.Logger

	walletNotifier  func(chainID uint64, message string)
	providerConfigs []params.ProviderConfig
}

// Is initialized in a build-tag-dependent module
var verifProxyInitFn func(c *Client)

// ClientConfig holds the configuration for initializing a new Client.
type ClientConfig struct {
	Client          *gethrpc.Client
	UpstreamChainID uint64
	Networks        []params.Network
	DB              *sql.DB
	WalletFeed      *event.Feed
	ProviderConfigs []params.ProviderConfig
}

// NewClient initializes Client
//
// Client is safe for concurrent use and will automatically
// reconnect to the server if connection is lost.
func NewClient(config ClientConfig) (*Client, error) {
	var err error

	logger := logutils.ZapLogger().Named("rpcClient")
	networkManager := network.NewManager(config.DB)
	if networkManager == nil {
		return nil, errors.New("failed to create network manager")
	}

	err = networkManager.Init(config.Networks)
	if err != nil {
		logger.Error("Network manager failed to initialize", zap.Error(err))
	}

	c := Client{
		local:              config.Client,
		NetworkManager:     networkManager,
		handlers:           make(map[string]Handler),
		rpcClients:         make(map[uint64]chain.ClientInterface),
		limiterPerProvider: make(map[string]*rpclimiter.RPCRpsLimiter),
		logger:             logger,
		providerConfigs:    config.ProviderConfigs,
		healthMgr:          healthmanager.NewBlockchainHealthManager(),
		walletFeed:         config.WalletFeed,
	}

	c.UpstreamChainID = config.UpstreamChainID
	c.router = newRouter(true)

	if verifProxyInitFn != nil {
		verifProxyInitFn(&c)
	}

	return &c, nil
}

func (c *Client) Start(ctx context.Context) {
	if c.stopMonitoringFunc != nil {
		c.logger.Warn("Blockchain health manager already started")
		return
	}

	cancelableCtx, cancel := context.WithCancel(ctx)
	c.stopMonitoringFunc = cancel
	statusCh := c.healthMgr.Subscribe()
	go c.monitorHealth(cancelableCtx, statusCh)
}

func (c *Client) Stop() {
	c.healthMgr.Stop()
	if c.stopMonitoringFunc == nil {
		return
	}
	c.stopMonitoringFunc()
	c.stopMonitoringFunc = nil
}

func (c *Client) monitorHealth(ctx context.Context, statusCh chan struct{}) {
	defer appCommon.LogOnPanic()
	sendFullStatusEventFunc := func() {
		blockchainStatus := c.healthMgr.GetFullStatus()
		encodedMessage, err := json.Marshal(blockchainStatus)
		if err != nil {
			c.logger.Warn("could not marshal full blockchain status", zap.Error(err))
			return
		}
		if c.walletFeed == nil {
			return
		}
		// FIXME: remove these excessive logs in future release (2.31+)
		c.logger.Debug("Sending blockchain health status event", zap.String("status", string(encodedMessage)))
		c.walletFeed.Send(walletevent.Event{
			Type:    EventBlockchainHealthChanged,
			Message: string(encodedMessage),
			At:      time.Now().Unix(),
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

func (c *Client) GetNetworkManager() *network.Manager {
	return c.NetworkManager
}

func (c *Client) SetWalletNotifier(notifier func(chainID uint64, message string)) {
	c.walletNotifier = notifier
}

func extractHostFromURL(inputURL string) (string, error) {
	parsedURL, err := url.Parse(inputURL)
	if err != nil {
		return "", err
	}

	return parsedURL.Host, nil
}

func (c *Client) getRPCRpsLimiter(key string) (*rpclimiter.RPCRpsLimiter, error) {
	c.rpsLimiterMutex.Lock()
	defer c.rpsLimiterMutex.Unlock()
	if limiter, ok := c.limiterPerProvider[key]; ok {
		return limiter, nil
	}
	limiter := rpclimiter.NewRPCRpsLimiter()
	c.limiterPerProvider[key] = limiter
	return limiter, nil
}

func (c *Client) getClientUsingCache(chainID uint64) (chain.ClientInterface, error) {
	c.rpcClientsMutex.Lock()
	defer c.rpcClientsMutex.Unlock()
	if rpcClient, ok := c.rpcClients[chainID]; ok {
		if rpcClient.GetWalletNotifier() == nil {
			rpcClient.SetWalletNotifier(c.walletNotifier)
		}
		return rpcClient, nil
	}

	network := c.NetworkManager.Find(chainID)
	if network == nil {
		return nil, fmt.Errorf("could not find network: %d", chainID)
	}

	ethClients := c.getEthClients(network)
	if len(ethClients) == 0 {
		return nil, fmt.Errorf("could not find any RPC URL for chain: %d", chainID)
	}

	phm := healthmanager.NewProvidersHealthManager(chainID)
	err := c.healthMgr.RegisterProvidersHealthManager(context.Background(), phm)
	if err != nil {
		return nil, fmt.Errorf("register providers health manager: %s", err)
	}

	client := chain.NewClient(ethClients, chainID, phm)
	client.SetWalletNotifier(c.walletNotifier)
	c.rpcClients[chainID] = client
	return client, nil
}

func (c *Client) getEthClients(network *params.Network) []ethclient.RPSLimitedEthClientInterface {
	ethClients := make([]ethclient.RPSLimitedEthClientInterface, 0)

	providers := c.prepareProviders(network)
	for index, provider := range providers {
		var rpcClient *gethrpc.Client
		var rpcLimiter *rpclimiter.RPCRpsLimiter
		var err error
		var hostPort string

		if len(provider.URL) > 0 {
			// For now, we only support auth for status-proxy.
			var opts []gethrpc.ClientOption
			if provider.authenticationNeeded() {
				authEncoded := base64.StdEncoding.EncodeToString([]byte(provider.Auth))
				opts = append(opts,
					gethrpc.WithHeaders(http.Header{
						"Authorization": {"Basic " + authEncoded},
						"User-Agent":    {rpcUserAgentName},
					}),
				)
			}

			rpcClient, err = gethrpc.DialOptions(context.Background(), provider.URL, opts...)
			if err != nil {
				c.logger.Error("dial server "+provider.Key, zap.Error(err))
			}

			// If using the status-proxy, consider each endpoint as a separate provider
			circuitKey := fmt.Sprintf("%s-%d", provider.Key, index)
			// Otherwise host is good enough
			if !strings.Contains(provider.URL, "status.im") {
				hostPort, err = extractHostFromURL(provider.URL)
				if err == nil {
					circuitKey = hostPort
				}
			}

			rpcLimiter, err = c.getRPCRpsLimiter(circuitKey)
			if err != nil {
				c.logger.Error("get RPC limiter "+provider.Key, zap.Error(err))
			}

			ethClients = append(ethClients, ethclient.NewRPSLimitedEthClient(rpcClient, rpcLimiter, circuitKey))
		}
	}

	return ethClients
}

// EthClient returns ethclient.Client per chain
func (c *Client) EthClient(chainID uint64) (chain.ClientInterface, error) {
	client, err := c.getClientUsingCache(chainID)
	if err != nil {
		return nil, err
	}

	return client, nil
}

// AbstractEthClient returns a partial abstraction used by new components for testing purposes
func (c *Client) AbstractEthClient(chainID common.ChainID) (ethclient.BatchCallClient, error) {
	client, err := c.getClientUsingCache(uint64(chainID))
	if err != nil {
		return nil, err
	}

	return client, nil
}

func (c *Client) EthClients(chainIDs []uint64) (map[uint64]chain.ClientInterface, error) {
	clients := make(map[uint64]chain.ClientInterface, 0)
	for _, chainID := range chainIDs {
		client, err := c.getClientUsingCache(chainID)
		if err != nil {
			return nil, err
		}
		clients[chainID] = client
	}

	return clients, nil
}

// SetClient strictly for testing purposes
func (c *Client) SetClient(chainID uint64, client chain.ClientInterface) {
	c.rpcClientsMutex.Lock()
	defer c.rpcClientsMutex.Unlock()
	c.rpcClients[chainID] = client
}

// Call performs a JSON-RPC call with the given arguments and unmarshals into
// result if no error occurred.
//
// The result must be a pointer so that package json can unmarshal into it. You
// can also pass nil, in which case the result is ignored.
//
// It uses custom routing scheme for calls.
func (c *Client) Call(result interface{}, chainID uint64, method string, args ...interface{}) error {
	ctx := context.Background()
	return c.CallContext(ctx, result, chainID, method, args...)
}

// CallContext performs a JSON-RPC call with the given arguments. If the context is
// canceled before the call has successfully returned, CallContext returns immediately.
//
// The result must be a pointer so that package json can unmarshal into it. You
// can also pass nil, in which case the result is ignored.
//
// It uses custom routing scheme for calls.
// If there are any local handlers registered for this call, they will handle it.
func (c *Client) CallContext(ctx context.Context, result interface{}, chainID uint64, method string, args ...interface{}) error {
	rpcstats.CountCall(method)
	if c.router.routeBlocked(method) {
		return ErrMethodNotFound
	}

	// check locally registered handlers first
	if handler, ok := c.handler(method); ok {
		return c.callMethod(ctx, result, chainID, handler, args...)
	}

	return c.CallContextIgnoringLocalHandlers(ctx, result, chainID, method, args...)
}

// CallContextIgnoringLocalHandlers performs a JSON-RPC call with the given
// arguments.
//
// If there are local handlers registered for this call, they would
// be ignored. It is useful if the call is happening from within a local
// handler itself.
// Upstream calls routing will be used anyway.
func (c *Client) CallContextIgnoringLocalHandlers(ctx context.Context, result interface{}, chainID uint64, method string, args ...interface{}) error {
	if c.router.routeBlocked(method) {
		return ErrMethodNotFound
	}

	if c.router.routeRemote(method) {
		client, err := c.getClientUsingCache(chainID)
		if err != nil {
			return err
		}
		return client.CallContext(ctx, result, method, args...)
	}

	if c.local == nil {
		c.logger.Warn("Local JSON-RPC endpoint missing", zap.String("method", method))
		return errors.New("missing local JSON-RPC endpoint")
	}
	return c.local.CallContext(ctx, result, method, args...)
}

// RegisterHandler registers local handler for specific RPC method.
//
// If method is registered, it will be executed with given handler and
// never routed to the upstream or local servers.
func (c *Client) RegisterHandler(method string, handler Handler) {
	c.handlersMx.Lock()
	defer c.handlersMx.Unlock()

	c.handlers[method] = handler
}

// UnregisterHandler removes a previously registered handler.
func (c *Client) UnregisterHandler(method string) {
	c.handlersMx.Lock()
	defer c.handlersMx.Unlock()

	delete(c.handlers, method)
}

// callMethod calls registered RPC handler with given args and pointer to result.
// It handles proper params and result converting
//
// TODO(divan): use cancellation via context here?
func (c *Client) callMethod(ctx context.Context, result interface{}, chainID uint64, handler Handler, args ...interface{}) error {
	response, err := handler(ctx, chainID, args...)
	if err != nil {
		return err
	}

	// if result is nil, just ignore result -
	// the same way as gethrpc.CallContext() caller would expect
	if result == nil {
		return nil
	}

	return setResultFromRPCResponse(result, response)
}

// handler is a concurrently safe method to get registered handler by name.
func (c *Client) handler(method string) (Handler, bool) {
	c.handlersMx.RLock()
	defer c.handlersMx.RUnlock()
	handler, ok := c.handlers[method]
	return handler, ok
}

// setResultFromRPCResponse tries to set result value from response using reflection
// as concrete types are unknown.
func setResultFromRPCResponse(result, response interface{}) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("invalid result type: %s", r)
		}
	}()

	responseValue := reflect.ValueOf(response)

	// If it is called via CallRaw, result has type json.RawMessage and
	// we should marshal the response before setting it.
	// Otherwise, it is called with CallContext and result is of concrete type,
	// thus we should try to set it as it is.
	// If response type and result type are incorrect, an error should be returned.
	// TODO(divan): add additional checks for result underlying value, if needed:
	// some example: https://golang.org/src/encoding/json/decode.go#L596
	switch reflect.ValueOf(result).Elem().Type() {
	case reflect.TypeOf(json.RawMessage{}), reflect.TypeOf([]byte{}):
		data, err := json.Marshal(response)
		if err != nil {
			return err
		}

		responseValue = reflect.ValueOf(data)
	}

	value := reflect.ValueOf(result).Elem()
	if !value.CanSet() {
		return errors.New("can't assign value to result")
	}
	value.Set(responseValue)

	return nil
}
