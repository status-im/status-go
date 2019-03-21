package rpc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/log"
	gethrpc "github.com/ethereum/go-ethereum/rpc"
	"github.com/status-im/status-go/params"
)

const (
	// DefaultCallTimeout is a default timeout for an RPC call
	DefaultCallTimeout = time.Minute
)

// List of RPC client errors.
var (
	ErrMethodNotFound = fmt.Errorf("The method does not exist/is not available")
)

// Handler defines handler for RPC methods.
type Handler func(context.Context, ...interface{}) (interface{}, error)

// Client represents RPC client with custom routing
// scheme. It automatically decides where RPC call
// goes - Upstream or Local node.
type Client struct {
	sync.RWMutex

	upstreamEnabled bool
	upstreamURL     string

	local    *gethrpc.Client
	upstream *gethrpc.Client

	router *router

	handlersMx sync.RWMutex       // mx guards handlers
	handlers   map[string]Handler // locally registered handlers
	log        log.Logger
}

// NewClient initializes Client and tries to connect to both,
// upstream and local node.
//
// Client is safe for concurrent use and will automatically
// reconnect to the server if connection is lost.
func NewClient(client *gethrpc.Client, upstream params.UpstreamRPCConfig) (*Client, error) {
	c := Client{
		local:    client,
		handlers: make(map[string]Handler),
		log:      log.New("package", "status-go/rpc.Client"),
	}

	var err error

	if upstream.Enabled {
		c.upstreamEnabled = upstream.Enabled
		c.upstreamURL = upstream.URL
		c.upstream, err = gethrpc.Dial(c.upstreamURL)
		if err != nil {
			return nil, fmt.Errorf("dial upstream server: %s", err)
		}
	}

	c.router = newRouter(c.upstreamEnabled)

	return &c, nil
}

// UpdateUpstreamURL changes the upstream RPC client URL, if the upstream is enabled.
func (c *Client) UpdateUpstreamURL(url string) error {
	if c.upstream == nil {
		return nil
	}

	rpcClient, err := gethrpc.Dial(url)
	if err != nil {
		return err
	}

	c.Lock()
	c.upstream = rpcClient
	c.upstreamURL = url
	c.Unlock()

	return nil
}

// Call performs a JSON-RPC call with the given arguments and unmarshals into
// result if no error occurred.
//
// The result must be a pointer so that package json can unmarshal into it. You
// can also pass nil, in which case the result is ignored.
//
// It uses custom routing scheme for calls.
func (c *Client) Call(result interface{}, method string, args ...interface{}) error {
	ctx := context.Background()
	return c.CallContext(ctx, result, method, args...)
}

// CallContext performs a JSON-RPC call with the given arguments. If the context is
// canceled before the call has successfully returned, CallContext returns immediately.
//
// The result must be a pointer so that package json can unmarshal into it. You
// can also pass nil, in which case the result is ignored.
//
// It uses custom routing scheme for calls.
// If there are any local handlers registered for this call, they will handle it.
func (c *Client) CallContext(ctx context.Context, result interface{}, method string, args ...interface{}) error {
	if c.router.routeBlocked(method) {
		return ErrMethodNotFound
	}

	// check locally registered handlers first
	if handler, ok := c.handler(method); ok {
		return c.callMethod(ctx, result, handler, args...)
	}

	return c.CallContextIgnoringLocalHandlers(ctx, result, method, args...)
}

// CallContextIgnoringLocalHandlers performs a JSON-RPC call with the given
// arguments.
//
// If there are local handlers registered for this call, they would
// be ignored. It is useful if the call is happening from within a local
// handler itself.
// Upstream calls routing will be used anyway.
func (c *Client) CallContextIgnoringLocalHandlers(ctx context.Context, result interface{}, method string, args ...interface{}) error {
	if c.router.routeBlocked(method) {
		return ErrMethodNotFound
	}

	if c.router.routeRemote(method) {
		c.RLock()
		client := c.upstream
		c.RUnlock()
		return client.CallContext(ctx, result, method, args...)
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

// callMethod calls registered RPC handler with given args and pointer to result.
// It handles proper params and result converting
//
// TODO(divan): use cancellation via context here?
func (c *Client) callMethod(ctx context.Context, result interface{}, handler Handler, args ...interface{}) error {
	response, err := handler(ctx, args...)
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
