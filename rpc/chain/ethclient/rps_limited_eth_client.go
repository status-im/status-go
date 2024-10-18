package ethclient

//go:generate mockgen -package=mock_ethclient -source=rps_limited_eth_client.go -destination=mock/client/ethclient/rps_limited_eth_client.go

import (
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/status-im/status-go/rpc/chain/rpclimiter"
)

// RPSLimitedEthClientInterface extends EthClientInterface with additional
// RPS-Limiting related capabilities.
// Ideally this shouldn't exist, instead we should be using EthClientInterface
// everywhere and clients shouldn't be aware of additional capabilities like
// PRS limiting. fallback mechanisms or caching.
type RPSLimitedEthClientInterface interface {
	EthClientInterface
	GetLimiter() *rpclimiter.RPCRpsLimiter
	GetName() string
	CopyWithName(name string) RPSLimitedEthClientInterface
}

type RPSLimitedEthClient struct {
	*EthClient
	limiter *rpclimiter.RPCRpsLimiter
	name    string
}

func NewRPSLimitedEthClient(rpcClient *rpc.Client, limiter *rpclimiter.RPCRpsLimiter, name string) *RPSLimitedEthClient {
	return &RPSLimitedEthClient{
		EthClient: NewEthClient(rpcClient),
		limiter:   limiter,
		name:      name,
	}
}

func (c *RPSLimitedEthClient) GetLimiter() *rpclimiter.RPCRpsLimiter {
	return c.limiter
}

func (c *RPSLimitedEthClient) GetName() string {
	return c.name
}

func (c *RPSLimitedEthClient) CopyWithName(name string) RPSLimitedEthClientInterface {
	return NewRPSLimitedEthClient(c.rpcClient, c.limiter, name)
}
