package rpc

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/les"
	gethnode "github.com/ethereum/go-ethereum/node"
	gethrpc "github.com/ethereum/go-ethereum/rpc"
	"github.com/status-im/status-go/geth/log"
	"github.com/status-im/status-go/geth/params"
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
func NewClient(node *gethnode.Node, upstream params.UpstreamRPCConfig) (*Client, error) {
	c := &Client{}

	var err error
	c.local, err = node.Attach()
	if err != nil {
		return nil, fmt.Errorf("attach to local node: %s", err)
	}

	c.upstreamURL = upstream.URL
	c.upstream, err = gethrpc.Dial(c.upstreamURL)
	if err != nil {
		return nil, fmt.Errorf("dial upstream server: %s", err)
	}

	if upstream.Enabled {
		log.Info("Enabling the upstream node")
		c.upstreamEnabled = upstream.Enabled
	} else {
		// just necessary if the LES protocol is being executed
		// verifies if the local node has enough disk space for the sync
		// operation in case it has been selected has the go to node
		var lesService les.LightEthereum
		if err := node.Service(&lesService); err != gethnode.ErrServiceUnknown {
			log.Info("Starting sync requirements analysis")
			passed, err := SyncPreRequisites(c)
			if err != nil {
				return nil, fmt.Errorf("verify space requirement to sync: %s", err)
			}
			log.Info("Successfully analyzed sync requirements")
			if !passed {
				log.Info("Enabling the upstream node")
				c.upstreamEnabled = true
				// the eth api stays available for exceptional routes (ex: eth_sign)
				if err := lesService.Stop(); err != nil {
					return nil, err
				}
			}
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
	if c.router.routeRemote(method) {
		return c.upstream.CallContext(ctx, result, method, args...)
	}
	return c.local.CallContext(ctx, result, method, args...)
}
