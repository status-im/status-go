package commands

import (
	"encoding/json"
	"fmt"
)

type ChainIDCommand struct {
	RpcClient RPCClientInterface
}

func (c *ChainIDCommand) Execute(request RPCRequest) (string, error) {
	chainsRequest := RPCRequest{
		Method: "eth_chainId",
		Params: []interface{}{},
	}

	requestJSON, err := json.Marshal(chainsRequest)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %v", err)
	}

	return c.RpcClient.CallRaw(string(requestJSON)), nil
}
