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

	"github.com/ethereum/go-ethereum/log"
	gethrpc "github.com/ethereum/go-ethereum/rpc"

	appCommon "github.com/status-im/status-go/common"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/rpc/chain"
	"github.com/status-im/status-go/rpc/chain/ethclient"
	"github.com/status-im/status-go/rpc/chain/rpclimiter"
	"github.com/status-im/status-go/rpc/network"
	"github.com/status-im/status-go/services/rpcstats"
	"github.com/status-im/status-go/services/wallet/common"
)

const (
	// DefaultCallTimeout is a default timeout for an RPC call
	DefaultCallTimeout = time.Minute

	// Names of providers
	providerGrove       = "grove"
	providerInfura      = "infura"
	ProviderStatusProxy = "status-proxy"

	mobile  = "mobile"
	desktop = "desktop"

	// rpcUserAgentFormat 'procurator': *an agent representing others*, aka a "proxy"
	// allows for the rpc client to have a dedicated user agent, which is useful for the proxy server logs.
	rpcUserAgentFormat = "procuratee-%s/%s"

	// rpcUserAgentUpstreamFormat a separate user agent format for upstream, because we should not be using upstream
	// if we see this user agent in the logs that means parts of the application are using a malconfigured http client
	rpcUserAgentUpstreamFormat = "procuratee-%s-upstream/%s"
)

// List of RPC client errors.
var (
	ErrMethodNotFound = fmt.Errorf("the method does not exist/is not available")
)

var (
	// rpcUserAgentName the user agent
	rpcUserAgentName         = fmt.Sprintf(rpcUserAgentFormat, "no-GOOS", params.Version)
	rpcUserAgentUpstreamName = fmt.Sprintf(rpcUserAgentUpstreamFormat, "no-GOOS", params.Version)
)

func init() {
	if appCommon.IsMobilePlatform() {
		rpcUserAgentName = fmt.Sprintf(rpcUserAgentFormat, mobile, params.Version)
		rpcUserAgentUpstreamName = fmt.Sprintf(rpcUserAgentUpstreamFormat, mobile, params.Version)
	} else {
		rpcUserAgentName = fmt.Sprintf(rpcUserAgentFormat, desktop, params.Version)
		rpcUserAgentUpstreamName = fmt.Sprintf(rpcUserAgentUpstreamFormat, desktop, params.Version)
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

	handlersMx sync.RWMutex       // mx guards handlers
	handlers   map[string]Handler // locally registered handlers
	log        log.Logger

	walletNotifier  func(chainID uint64, message string)
	providerConfigs []params.ProviderConfig
}

// Is initialized in a build-tag-dependent module
var verifProxyInitFn func(c *Client)

// NewClient initializes Client
//
// Client is safe for concurrent use and will automatically
// reconnect to the server if connection is lost.
func NewClient(client *gethrpc.Client, upstreamChainID uint64, networks []params.Network, db *sql.DB, providerConfigs []params.ProviderConfig) (*Client, error) {
	var err error

	log := log.New("package", "status-go/rpc.Client")
	networkManager := network.NewManager(db)
	if networkManager == nil {
		return nil, errors.New("failed to create network manager")
	}

	err = networkManager.Init(networks)
	if err != nil {
		log.Error("Network manager failed to initialize", "error", err)
	}

	c := Client{
		local:              client,
		NetworkManager:     networkManager,
		handlers:           make(map[string]Handler),
		rpcClients:         make(map[uint64]chain.ClientInterface),
		limiterPerProvider: make(map[string]*rpclimiter.RPCRpsLimiter),
		log:                log,
		providerConfigs:    providerConfigs,
	}

	c.UpstreamChainID = upstreamChainID
	c.router = newRouter(true)

	if verifProxyInitFn != nil {
		verifProxyInitFn(&c)
	}

	return &c, nil
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

func getProviderConfig(providerConfigs []params.ProviderConfig, providerName string) (params.ProviderConfig, error) {
	for _, providerConfig := range providerConfigs {
		if providerConfig.Name == providerName {
			return providerConfig, nil
		}
	}
	return params.ProviderConfig{}, fmt.Errorf("provider config not found for provider: %s", providerName)
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

	client := chain.NewClient(ethClients, chainID)
	client.SetWalletNotifier(c.walletNotifier)
	c.rpcClients[chainID] = client
	return client, nil
}

func (c *Client) getEthClients(network *params.Network) []ethclient.RPSLimitedEthClientInterface {
	urls := make(map[string]string)
	keys := make([]string, 0)
	authMap := make(map[string]string)

	// find proxy provider
	proxyProvider, err := getProviderConfig(c.providerConfigs, ProviderStatusProxy)
	if err != nil {
		c.log.Warn("could not find provider config for status-proxy", "error", err)
	}

	if proxyProvider.Enabled {
		key := ProviderStatusProxy
		keyFallback := ProviderStatusProxy + "-fallback"
		keyFallback2 := ProviderStatusProxy + "-fallback2"
		urls[key] = network.DefaultRPCURL
		urls[keyFallback] = network.DefaultFallbackURL
		urls[keyFallback2] = network.DefaultFallbackURL2
		keys = []string{key, keyFallback, keyFallback2}
		authMap[key] = proxyProvider.User + ":" + proxyProvider.Password
		authMap[keyFallback] = authMap[key]
		authMap[keyFallback2] = authMap[key]
	}
	keys = append(keys, []string{"main", "fallback"}...)
	urls["main"] = network.RPCURL
	urls["fallback"] = network.FallbackURL

	ethClients := make([]ethclient.RPSLimitedEthClientInterface, 0)
	for index, key := range keys {
		var rpcClient *gethrpc.Client
		var rpcLimiter *rpclimiter.RPCRpsLimiter
		var err error
		var hostPort string
		url := urls[key]

		if len(url) > 0 {
			// For now, we only support auth for status-proxy.
			authStr, ok := authMap[key]
			var opts []gethrpc.ClientOption
			if ok {
				authEncoded := base64.StdEncoding.EncodeToString([]byte(authStr))
				opts = append(opts,
					gethrpc.WithHeaders(http.Header{
						"Authorization": {"Basic " + authEncoded},
						"User-Agent":    {rpcUserAgentName},
					}),
				)
			}

			rpcClient, err = gethrpc.DialOptions(context.Background(), url, opts...)
			if err != nil {
				c.log.Error("dial server "+key, "error", err)
			}

			// If using the status-proxy, consider each endpoint as a separate provider
			circuitKey := fmt.Sprintf("%s-%d", key, index)
			// Otherwise host is good enough
			if !strings.Contains(url, "status.im") {
				hostPort, err = extractHostFromURL(url)
				if err == nil {
					circuitKey = hostPort
				}
			}

			rpcLimiter, err = c.getRPCRpsLimiter(circuitKey)
			if err != nil {
				c.log.Error("get RPC limiter "+key, "error", err)
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
		c.log.Warn("Local JSON-RPC endpoint missing", "method", method)
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
