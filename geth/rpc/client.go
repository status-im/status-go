package rpc

import (
	"context"
	"fmt"

	"github.com/status-im/status-go/geth/params"

	gethrpc "github.com/ethereum/go-ethereum/rpc"
)

// Handler defines the type of function to be passed to RegisterHandler.
type Handler func(context.Context, ...interface{}) (interface{}, error)

// Client represents RPC client with custom routing
// scheme. It automatically decides where RPC call
// goes - Upstream or Local node.
type Client struct {
	router *router
}

// NewClient initializes Client and tries to connect to both,
// upstream and local node.
//
// Client is safe for concurrent use and will automatically
// reconnect to the server if connection is lost.
func NewClient(client *gethrpc.Client, upstream params.UpstreamRPCConfig) (*Client, error) {
	c := Client{}

	var err error

	var upstreamNode *gethrpc.Client
	if upstream.Enabled {
		upstreamNode, err = gethrpc.Dial(upstream.URL)
		if err != nil {
			return nil, fmt.Errorf("dial upstream server: %s", err)
		}
	}

	c.router = newRouter(client, upstreamNode, upstream.Enabled)

	return &c, nil
}

// Call performs a JSON-RPC call with the given arguments and unmarshals into
// result if no error occurred.
//
// The result must be a pointer so that package json can unmarshal into it. You
// can also pass nil, in which case the result is ignored.
//
// It uses custom routing scheme for calls, as determined by RoutingTable.
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
// It uses custom routing scheme for calls, as determined by RoutingTable.
func (c *Client) CallContext(ctx context.Context, result interface{}, method string, args ...interface{}) error {
	return c.router.callContext(ctx, result, method, args...)
}

// RegisterHandler registers local handler for specific RPC method.
//
// If method handler is registered, and if registration is permitted by
// the router, then it will be executed with given handler and never
// routed to the upstream or local servers.
func (c *Client) RegisterHandler(method string, handler Handler) {
	c.router.registerHandler(method, handler)
}
