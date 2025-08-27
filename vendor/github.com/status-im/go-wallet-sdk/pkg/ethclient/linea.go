package ethclient

import (
	"context"

	"github.com/ethereum/go-ethereum"
)

func (c *Client) LineaEstimateGas(ctx context.Context, msg ethereum.CallMsg) (*LineaEstimateGasResult, error) {
	var result LineaEstimateGasResult
	err := c.rpcClient.CallContext(ctx, &result, "linea_estimateGas", toCallArg(msg))
	if err != nil {
		return nil, err
	}
	return &result, nil
}
