package ethclient

//go:generate mockgen -package=mock_ethclient -source=rps_limited_eth_client.go -destination=mock/client/ethclient/rps_limited_eth_client.go

import (
	"strings"

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
	GetCircuitName() string
	GetProviderName() string
	CopyWithCircuitName(name string) RPSLimitedEthClientInterface
	ExecuteWithRPSLimit(f func(client EthClientInterface) (interface{}, error)) (interface{}, error)
}

type RPSLimitedEthClient struct {
	*EthClient
	limiter      *rpclimiter.RPCRpsLimiter
	circuitName  string
	providerName string
}

func NewRPSLimitedEthClient(rpcClient *rpc.Client, limiter *rpclimiter.RPCRpsLimiter, circuitName, providerName string) *RPSLimitedEthClient {
	return &RPSLimitedEthClient{
		EthClient:    NewEthClient(rpcClient),
		limiter:      limiter,
		circuitName:  circuitName,
		providerName: providerName,
	}
}

func (c *RPSLimitedEthClient) GetLimiter() *rpclimiter.RPCRpsLimiter {
	return c.limiter
}

func (c *RPSLimitedEthClient) GetCircuitName() string {
	return c.circuitName
}

func (c *RPSLimitedEthClient) GetProviderName() string {
	return c.providerName
}

func (c *RPSLimitedEthClient) CopyWithCircuitName(circuitName string) RPSLimitedEthClientInterface {
	return NewRPSLimitedEthClient(c.rpcClient, c.limiter, circuitName, c.providerName)
}

func (c *RPSLimitedEthClient) ExecuteWithRPSLimit(f func(client EthClientInterface) (interface{}, error)) (interface{}, error) {
	limiter := c.GetLimiter()
	if limiter != nil {
		err := limiter.WaitForRequestsAvailability(1)
		if err != nil {
			return nil, err
		}
	}

	res, err := f(c)
	if err != nil {
		if limiter != nil && isRPSLimitError(err) {
			limiter.ReduceLimit()

			err = limiter.WaitForRequestsAvailability(1)
			if err != nil {
				return nil, err
			}

			res, err = f(c)
			if err == nil {
				return res, nil
			}
		}

		return nil, err
	}
	return res, nil
}

// isRPSLimitError checks if the error is related to RPS limit.
func isRPSLimitError(err error) bool {
	return strings.Contains(err.Error(), "backoff_seconds") ||
		strings.Contains(err.Error(), "has exceeded its throughput limit") ||
		strings.Contains(err.Error(), "request rate exceeded")
}
