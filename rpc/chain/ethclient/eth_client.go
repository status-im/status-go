package ethclient

//go:generate mockgen -package=mock_ethclient -source=eth_client.go -destination=mock/client/ethclient/eth_client.go

import (
	"context"
	"math/big"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

type ChainReader interface {
	BlockByHash(ctx context.Context, hash common.Hash) (*types.Block, error)
	BlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error)
	HeaderByHash(ctx context.Context, hash common.Hash) (*types.Header, error)
	HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error)
}

type CallClient interface {
	CallContext(ctx context.Context, result interface{}, method string, args ...interface{}) error
}

type BatchCallClient interface {
	BatchCallContext(ctx context.Context, b []rpc.BatchElem) error
}

// Interface for rpc.Client
type RPCClientInterface interface {
	CallClient
	BatchCallClient
}

// Interface for ethclient.Client
type BaseEthClientInterface interface {
	// External calls
	ChainReader
	ethereum.TransactionReader
	ethereum.ChainStateReader
	ethereum.ChainSyncReader
	ethereum.ContractCaller
	ethereum.LogFilterer
	ethereum.TransactionSender
	ethereum.GasPricer
	ethereum.PendingStateReader
	ethereum.PendingContractCaller
	ethereum.GasEstimator
	FeeHistory(ctx context.Context, blockCount uint64, lastBlock *big.Int, rewardPercentiles []float64) (*ethereum.FeeHistory, error)
	BlockNumber(ctx context.Context) (uint64, error)
	TransactionSender(ctx context.Context, tx *types.Transaction, block common.Hash, index uint) (common.Address, error)
	// Internal calls
	Close()
}

// EthClientInterface extends BaseEthClientInterface with additional capabilities
type EthClientInterface interface {
	BaseEthClientInterface
	// Additional external calls
	RPCClientInterface
	GetName() string
	CopyWithName(name string) EthClientInterface
	GetBaseFeeFromBlock(ctx context.Context, blockNumber *big.Int) (string, error)
	bind.ContractCaller
	bind.ContractBackend
}

// EthClient implements EthClientInterface
type EthClient struct {
	*ethclient.Client
	name      string
	rpcClient *rpc.Client
}

func NewEthClient(rpcClient *rpc.Client, name string) *EthClient {
	return &EthClient{
		Client:    ethclient.NewClient(rpcClient),
		name:      name,
		rpcClient: rpcClient,
	}
}

func (c *EthClient) GetName() string {
	return c.name
}

func (c *EthClient) CopyWithName(name string) EthClientInterface {
	return NewEthClient(c.rpcClient, name)
}
func (ec *EthClient) BatchCallContext(ctx context.Context, b []rpc.BatchElem) error {
	return ec.rpcClient.BatchCallContext(ctx, b)
}

func (ec *EthClient) GetBaseFeeFromBlock(ctx context.Context, blockNumber *big.Int) (string, error) {
	feeHistory, err := ec.FeeHistory(ctx, 1, blockNumber, nil)

	if err != nil {
		if err.Error() == "the method eth_feeHistory does not exist/is not available" {
			return "", nil
		}
		return "", err
	}

	var baseGasFee string = ""
	if len(feeHistory.BaseFee) > 0 {
		baseGasFee = feeHistory.BaseFee[0].String()
	}

	return baseGasFee, err
}

func (ec *EthClient) CallContext(ctx context.Context, result interface{}, method string, args ...interface{}) error {
	return ec.rpcClient.CallContext(ctx, result, method, args...)
}
