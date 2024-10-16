package ethclient

//go:generate mockgen -package=mock_ethclient -source=rps_limited_eth_client.go -destination=mock/client/ethclient/rps_limited_eth_client.go

import (
	"github.com/ethereum/go-ethereum/rpc"
)

// RPSLimitedEthClientInterface extends EthClientInterface with additional
// RPS-Limiting related capabilities.
// Ideally this shouldn't exist, instead we should be using EthClientInterface
// everywhere and clients shouldn't be aware of additional capabilities like
// PRS limiting. fallback mechanisms or caching.
type RPSLimitedEthClientInterface interface {
	EthClientInterface
	GetName() string
	CopyWithName(name string) RPSLimitedEthClientInterface
}

type RPSLimitedEthClient struct {
	*EthClient
	name string
}

func NewRPSLimitedEthClient(rpcClient *rpc.Client, name string) *RPSLimitedEthClient {
	return &RPSLimitedEthClient{
		EthClient: NewEthClient(rpcClient),
		name:      name,
	}
}

func (c *RPSLimitedEthClient) GetName() string {
	return c.name
}

func (c *RPSLimitedEthClient) CopyWithName(name string) RPSLimitedEthClientInterface {
	return NewRPSLimitedEthClient(c.rpcClient, name)
}
