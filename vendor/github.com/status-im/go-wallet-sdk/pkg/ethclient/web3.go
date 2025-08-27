package ethclient

import (
	"context"
)

// Web3ClientVersion returns the version of the current client
func (c *Client) Web3ClientVersion(ctx context.Context) (string, error) {
	var result string
	err := c.rpcClient.CallContext(ctx, &result, "web3_clientVersion")
	return result, err
}
