package rpc

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/node"
	"github.com/status-im/status-go/geth/params"

	gethrpc "github.com/ethereum/go-ethereum/rpc"
)

// Client represents RPC client with custom routing
// scheme. It automatically decides where RPC call
// goes - Upstream or Local node.
type Client struct {
	upstreamEnabled bool
	upstreamURL     string

	local    *gethrpc.Client
	upstream *gethrpc.Client

	router *router
}

// NewClient initializes Client and tries to connect to both,
// upstream and local node.
//
// Client is safe for concurrent use and will automatically
// reconnect to the server if connection is lost.
func NewClient(node *node.Node, upstream params.UpstreamRPCConfig) (*Client, error) {
	c := &Client{}

	var err error
	c.local, err = node.Attach()
	if err != nil {
		return nil, fmt.Errorf("attach to local node: %s", err)
	}

	if upstream.Enabled {
		c.upstreamEnabled = upstream.Enabled
		c.upstreamURL = upstream.URL

		c.upstream, err = gethrpc.Dial(c.upstreamURL)
		if err != nil {
			return nil, fmt.Errorf("dial upstream server: %s", err)
		}
	}

	c.router = newRouter(c.upstreamEnabled)

	return c, nil
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
func (c *Client) CallContext(ctx context.Context, result interface{}, method string, args ...interface{}) error {
	if c.router.routeLocally(method) {
		return c.local.CallContext(ctx, result, method, args...)
	}

	return c.upstream.CallContext(ctx, result, method, args...)
}
