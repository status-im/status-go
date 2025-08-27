package ethclient

import (
	"context"

	"github.com/ethereum/go-ethereum/common/hexutil"
)

// NetListening returns true if client is actively listening for network connections
func (c *Client) NetListening(ctx context.Context) (bool, error) {
	var result bool
	err := c.rpcClient.CallContext(ctx, &result, "net_listening")
	return result, err
}

// NetPeerCount returns number of peers currently connected to the client
func (c *Client) NetPeerCount(ctx context.Context) (uint64, error) {
	var result hexutil.Uint64
	err := c.rpcClient.CallContext(ctx, &result, "net_peerCount")
	return uint64(result), err
}

// NetVersion returns the current network ID
func (c *Client) NetVersion(ctx context.Context) (string, error) {
	var result string
	err := c.rpcClient.CallContext(ctx, &result, "net_version")
	return result, err
}
