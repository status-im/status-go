package ethclient

//go:generate mockgen -destination=mock/client.go . RPCClient

import (
	"context"

	gethec "github.com/ethereum/go-ethereum/ethclient"
	gethrpc "github.com/ethereum/go-ethereum/rpc"
)

type RPCClient interface {
	CallContext(ctx context.Context, result interface{}, method string, args ...interface{}) error
	Close()
}

type Client struct {
	rpcClient     RPCClient
	gethEthClient *gethec.Client
}

// NewClient creates a new Ethereum client
func NewClient(rpcClient RPCClient) *Client {
	client := Client{
		rpcClient: rpcClient,
	}

	// If rpcClient is implemented by a *gethrpc.Client, use it to create a *gethec.Client
	if gethRPCClient, ok := rpcClient.(*gethrpc.Client); ok {
		client.gethEthClient = gethec.NewClient(gethRPCClient)
	}

	return &client
}

// Close closes the client
func (c *Client) Close() {
	if c.rpcClient != nil {
		c.rpcClient.Close()
	}
}
