package ethclient

import (
	"context"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/core/types"
)

// EthGetFilterChanges returns an array of all logs matching filter with given id
func (c *Client) EthGetFilterChanges(ctx context.Context, filterID FilterID) ([]*types.Log, error) {
	var result []*types.Log
	err := c.rpcClient.CallContext(ctx, &result, "eth_getFilterChanges", filterID)
	return result, err
}

// EthGetFilterLogs returns an array of all logs matching filter with given id
func (c *Client) EthGetFilterLogs(ctx context.Context, filterID FilterID) ([]types.Log, error) {
	var result []types.Log
	err := c.rpcClient.CallContext(ctx, &result, "eth_getFilterLogs", filterID)
	return result, err
}

// EthNewBlockFilter creates a filter in the node, to notify when a new block arrives
func (c *Client) EthNewBlockFilter(ctx context.Context) (FilterID, error) {
	var result FilterID
	err := c.rpcClient.CallContext(ctx, &result, "eth_newBlockFilter")
	return result, err
}

// EthNewFilter creates a filter object, based on filter options, to notify when the state changes
func (c *Client) EthNewFilter(ctx context.Context, query ethereum.FilterQuery) (FilterID, error) {
	var result FilterID
	filterArg, err := toFilterArg(query)
	if err != nil {
		return "", err
	}
	err = c.rpcClient.CallContext(ctx, &result, "eth_newFilter", filterArg)
	return result, err
}

// EthUninstallFilter uninstalls a filter with given id
func (c *Client) EthUninstallFilter(ctx context.Context, filterID FilterID) (bool, error) {
	var result bool
	err := c.rpcClient.CallContext(ctx, &result, "eth_uninstallFilter", filterID)
	return result, err
}
