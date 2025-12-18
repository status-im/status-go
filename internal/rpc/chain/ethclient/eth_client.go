package ethclient

//go:generate go tool mockgen -package=mock_ethclient -source=eth_client.go -destination=mock/client/ethclient/eth_client.go

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/v2"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/status-im/go-wallet-sdk/pkg/ethclient"
)

type ChainReader interface {
	HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error)
	EthGetBlockByHashWithTxHashes(ctx context.Context, hash common.Hash) (*ethclient.BlockWithTxHashes, error)
	EthGetBlockByNumberWithTxHashes(ctx context.Context, number *big.Int) (*ethclient.BlockWithTxHashes, error)
	EthGetBlockByHashWithFullTxs(ctx context.Context, hash common.Hash) (*ethclient.BlockWithFullTxs, error)
	EthGetBlockByNumberWithFullTxs(ctx context.Context, number *big.Int) (*ethclient.BlockWithFullTxs, error)
}

type TransactionReader interface {
	EthGetTransactionByHash(ctx context.Context, hash common.Hash) (*ethclient.Transaction, error)
	EthGetTransactionReceipt(ctx context.Context, txHash common.Hash) (*ethclient.Receipt, error)
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
	TransactionReader
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
	LineaEstimateGas(ctx context.Context, msg ethereum.CallMsg) (*ethclient.LineaEstimateGasResult, error)
	// Internal calls
	Close()
}

// EthClientInterface extends BaseEthClientInterface with additional capabilities
type EthClientInterface interface {
	BaseEthClientInterface
	// Additional external calls
	RPCClientInterface
	bind.ContractCaller
	bind.ContractBackend
}

// EthClient implements EthClientInterface
type EthClient struct {
	*ethclient.Client
	rpcClient *rpc.Client
}

func NewEthClient(rpcClient *rpc.Client) *EthClient {
	return &EthClient{
		Client:    ethclient.NewClient(rpcClient),
		rpcClient: rpcClient,
	}
}

func (ec *EthClient) BatchCallContext(ctx context.Context, b []rpc.BatchElem) error {
	return ec.rpcClient.BatchCallContext(ctx, b)
}

func (ec *EthClient) CallContext(ctx context.Context, result interface{}, method string, args ...interface{}) error {
	return ec.rpcClient.CallContext(ctx, result, method, args...)
}
