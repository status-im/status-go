package chain

//go:generate go tool mockgen -package=mock_client -source=client.go -destination=mock/client/client.go

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/rpc"

	gocommon "github.com/status-im/status-go/common"
	"github.com/status-im/status-go/internal/circuitbreaker"
	"github.com/status-im/status-go/internal/healthmanager"
	"github.com/status-im/status-go/internal/healthmanager/rpcstatus"
	ethclient2 "github.com/status-im/status-go/internal/rpc/chain/ethclient"
	"github.com/status-im/status-go/services/rpcstats"

	sdkethclient "github.com/status-im/go-wallet-sdk/pkg/ethclient"
)

type ClientInterface interface {
	ethclient2.EthClientInterface
	NetworkID() uint64
	GetProviderClient(provider string) ethclient2.EthClientInterface
}

type ClientWithFallback struct {
	ChainID                uint64
	ethClients             []ethclient2.RPSLimitedEthClientInterface
	circuitbreaker         *circuitbreaker.CircuitBreaker
	providersHealthManager *healthmanager.ProvidersHealthManager

	done   chan struct{}  // channel to signal client closure
	wg     sync.WaitGroup // wait group to track active operations
	closed atomic.Bool    // flag to track if client is closed
}

// Don't mark connection as failed if we get one of these errors
var propagateErrors = []error{
	vm.ErrOutOfGas,
	vm.ErrCodeStoreOutOfGas,
	vm.ErrDepth,
	vm.ErrInsufficientBalance,
	vm.ErrContractAddressCollision,
	vm.ErrExecutionReverted,
	vm.ErrMaxCodeSizeExceeded,
	vm.ErrInvalidJump,
	vm.ErrWriteProtection,
	vm.ErrReturnDataOutOfBounds,
	vm.ErrGasUintOverflow,
	vm.ErrInvalidCode,
	vm.ErrNonceUintOverflow,

	// Used by balance history to check state
	bind.ErrNoCode,
}

func NewClient(ethClients []ethclient2.RPSLimitedEthClientInterface, chainID uint64, providersHealthManager *healthmanager.ProvidersHealthManager) *ClientWithFallback {
	cbConfig := circuitbreaker.Config{
		Timeout:               20000,
		MaxConcurrentRequests: 100,
		SleepWindow:           300000,
		ErrorPercentThreshold: 25,
	}

	return &ClientWithFallback{
		ChainID:                chainID,
		ethClients:             ethClients,
		circuitbreaker:         circuitbreaker.NewCircuitBreaker(cbConfig),
		providersHealthManager: providersHealthManager,
		done:                   make(chan struct{}),
	}
}

func (c *ClientWithFallback) Close() {
	if !c.closed.CompareAndSwap(false, true) {
		return // already closed
	}

	close(c.done) // signal all ongoing operations to stop

	// Wait for all operations to complete
	c.wg.Wait()

	// Close all eth clients
	for _, client := range c.ethClients {
		client.Close()
	}
}

func isVMError(err error) bool {
	if strings.Contains(err.Error(), core.ErrInsufficientFunds.Error()) {
		return true
	}
	for _, vmError := range propagateErrors {
		if strings.Contains(err.Error(), vmError.Error()) {
			return true
		}
	}
	return false
}

func (c *ClientWithFallback) GetConnectionStatus() rpcstatus.StatusType {
	return c.providersHealthManager.Status().Status
}

func (c *ClientWithFallback) makeCall(ctx context.Context, f MakeCallFunctor) (interface{}, error) {
	rpcstats.CountCall(f.MethodName)
	if c.closed.Load() {
		return nil, errors.New("client is closed")
	}

	// Add the operation to the wait group
	c.wg.Add(1)
	defer c.wg.Done()

	// Create a context that will be cancelled when either the parent context is done or the client is closed
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Start a goroutine to watch for client closure
	go func() {
		defer gocommon.LogOnPanic()
		select {
		case <-c.done:
			cancel()
		case <-ctx.Done():
		}
	}()

	cmd := circuitbreaker.NewCommand(ctx, nil)
	// Try making requests with each RPC provider.
	// Cancel the command if we get a VM error or a context cancellation.
	for _, ethProviderClient := range c.ethClients {
		ethProviderClient := ethProviderClient
		cmd.Add(circuitbreaker.NewFunctor(func() ([]interface{}, error) {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			res, err := ethProviderClient.ExecuteWithRPSLimit(f.Func)
			if err != nil && (isVMError(err) || errors.Is(err, context.Canceled)) {
				cmd.Cancel()
			}
			return []interface{}{res}, err
		}, ethProviderClient.GetCircuitName(), ethProviderClient.GetProviderName()))
	}

	result := c.circuitbreaker.Execute(cmd)
	if c.providersHealthManager != nil {
		rpcCallStatuses := convertFunctorCallStatuses(result.FunctorCallStatuses(), f.MethodName)
		c.providersHealthManager.Update(ctx, rpcCallStatuses)
	}
	if result.Error() != nil {
		return nil, fmt.Errorf("%w (%s)", result.Error(), f.MethodName)
	}

	return result.Result()[0], nil
}

type MakeCallFunctor struct {
	MethodName string
	Func       func(client ethclient2.EthClientInterface) (interface{}, error)
}

func (c *ClientWithFallback) EthGetBlockByHashWithFullTxs(ctx context.Context, hash common.Hash) (*sdkethclient.BlockWithFullTxs, error) {
	res, err := c.makeCall(
		ctx, MakeCallFunctor{
			MethodName: "eth_GetBlockByHashWithFullTxs",
			Func: func(client ethclient2.EthClientInterface) (interface{}, error) {
				return client.EthGetBlockByHashWithFullTxs(ctx, hash)
			},
		},
	)
	if err != nil {
		return nil, err
	}

	return res.(*sdkethclient.BlockWithFullTxs), nil
}

func (c *ClientWithFallback) EthGetBlockByNumberWithFullTxs(ctx context.Context, number *big.Int) (*sdkethclient.BlockWithFullTxs, error) {
	res, err := c.makeCall(
		ctx, MakeCallFunctor{
			MethodName: "eth_EthGetBlockByNumberWithFullTxs",
			Func: func(client ethclient2.EthClientInterface) (interface{}, error) {
				return client.EthGetBlockByNumberWithFullTxs(ctx, number)
			},
		},
	)
	if err != nil {
		return nil, err
	}

	return res.(*sdkethclient.BlockWithFullTxs), nil
}

func (c *ClientWithFallback) BlockNumber(ctx context.Context) (uint64, error) {
	res, err := c.makeCall(
		ctx, MakeCallFunctor{
			MethodName: "eth_BlockNumber",
			Func: func(client ethclient2.EthClientInterface) (interface{}, error) {
				return client.BlockNumber(ctx)
			},
		},
	)
	if err != nil {
		return 0, err
	}

	return res.(uint64), nil
}

func (c *ClientWithFallback) HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error) {
	res, err := c.makeCall(
		ctx, MakeCallFunctor{
			MethodName: "eth_HeaderByNumber",
			Func: func(client ethclient2.EthClientInterface) (interface{}, error) {
				return client.HeaderByNumber(ctx, number)
			},
		},
	)
	if err != nil {
		return nil, err
	}

	return res.(*types.Header), nil
}

func (c *ClientWithFallback) EthGetBlockByHashWithTxHashes(ctx context.Context, hash common.Hash) (*sdkethclient.BlockWithTxHashes, error) {
	res, err := c.makeCall(
		ctx, MakeCallFunctor{
			MethodName: "eth_HeaderByHash",
			Func: func(client ethclient2.EthClientInterface) (interface{}, error) {
				return client.EthGetBlockByHashWithTxHashes(ctx, hash)
			},
		},
	)
	if err != nil {
		return nil, err
	}

	return res.(*sdkethclient.BlockWithTxHashes), nil
}

func (c *ClientWithFallback) EthGetBlockByNumberWithTxHashes(ctx context.Context, number *big.Int) (*sdkethclient.BlockWithTxHashes, error) {
	res, err := c.makeCall(
		ctx, MakeCallFunctor{
			MethodName: "eth_HeaderByNumber",
			Func: func(client ethclient2.EthClientInterface) (interface{}, error) {
				return client.EthGetBlockByNumberWithTxHashes(ctx, number)
			},
		},
	)
	if err != nil {
		return nil, err
	}

	return res.(*sdkethclient.BlockWithTxHashes), nil
}

func (c *ClientWithFallback) EthGetTransactionByHash(ctx context.Context, hash common.Hash) (*sdkethclient.Transaction, error) {
	res, err := c.makeCall(
		ctx, MakeCallFunctor{
			MethodName: "eth_TransactionByHash",
			Func: func(client ethclient2.EthClientInterface) (interface{}, error) {
				tx, err := client.EthGetTransactionByHash(ctx, hash)
				return []any{tx}, err
			},
		},
	)
	if err != nil {
		return nil, err
	}

	resArr := res.([]any)
	return resArr[0].(*sdkethclient.Transaction), nil
}

func (c *ClientWithFallback) EthGetTransactionReceipt(ctx context.Context, txHash common.Hash) (*sdkethclient.Receipt, error) {
	res, err := c.makeCall(
		ctx, MakeCallFunctor{
			MethodName: "eth_GetTransactionReceipt",
			Func: func(client ethclient2.EthClientInterface) (interface{}, error) {
				return client.EthGetTransactionReceipt(ctx, txHash)
			},
		},
	)
	if err != nil {
		return nil, err
	}

	return res.(*sdkethclient.Receipt), nil
}

func (c *ClientWithFallback) SyncProgress(ctx context.Context) (*ethereum.SyncProgress, error) {
	res, err := c.makeCall(
		ctx, MakeCallFunctor{
			MethodName: "eth_SyncProgress",
			Func: func(client ethclient2.EthClientInterface) (interface{}, error) {
				return client.SyncProgress(ctx)
			},
		},
	)
	if err != nil {
		return nil, err
	}

	return res.(*ethereum.SyncProgress), nil
}

func (c *ClientWithFallback) NetworkID() uint64 {
	return c.ChainID
}

func (c *ClientWithFallback) BalanceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (*big.Int, error) {
	res, err := c.makeCall(
		ctx, MakeCallFunctor{
			MethodName: "eth_BalanceAt",
			Func: func(client ethclient2.EthClientInterface) (interface{}, error) {
				return client.BalanceAt(ctx, account, blockNumber)
			},
		},
	)
	if err != nil {
		return nil, err
	}

	return res.(*big.Int), nil
}

func (c *ClientWithFallback) StorageAt(ctx context.Context, account common.Address, key common.Hash, blockNumber *big.Int) ([]byte, error) {
	res, err := c.makeCall(
		ctx, MakeCallFunctor{
			MethodName: "eth_StorageAt",
			Func: func(client ethclient2.EthClientInterface) (interface{}, error) {
				return client.StorageAt(ctx, account, key, blockNumber)
			},
		},
	)
	if err != nil {
		return nil, err
	}

	return res.([]byte), nil
}

func (c *ClientWithFallback) CodeAt(ctx context.Context, account common.Address, blockNumber *big.Int) ([]byte, error) {
	res, err := c.makeCall(
		ctx, MakeCallFunctor{
			MethodName: "eth_CodeAt",
			Func: func(client ethclient2.EthClientInterface) (interface{}, error) {
				return client.CodeAt(ctx, account, blockNumber)
			},
		},
	)
	if err != nil {
		return nil, err
	}

	return res.([]byte), nil
}

func (c *ClientWithFallback) NonceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (uint64, error) {
	res, err := c.makeCall(
		ctx, MakeCallFunctor{
			MethodName: "eth_NonceAt",
			Func: func(client ethclient2.EthClientInterface) (interface{}, error) {
				return client.NonceAt(ctx, account, blockNumber)
			},
		},
	)
	if err != nil {
		return 0, err
	}

	return res.(uint64), nil
}

func (c *ClientWithFallback) FilterLogs(ctx context.Context, q ethereum.FilterQuery) ([]types.Log, error) {
	// Override providers name to use a separate circuit for this command as it more often fails due to rate limiting
	ethClients := make([]ethclient2.RPSLimitedEthClientInterface, len(c.ethClients))
	for i, client := range c.ethClients {
		ethClients[i] = client.CopyWithCircuitName(client.GetCircuitName() + "_FilterLogs")
	}

	res, err := c.makeCall(
		ctx, MakeCallFunctor{
			MethodName: "eth_FilterLogs",
			Func: func(client ethclient2.EthClientInterface) (interface{}, error) {
				return client.FilterLogs(ctx, q)
			},
		},
	)

	// No connection state toggling here, as it often mail fail due to archive node rate limiting
	// which does not impact other calls

	if err != nil {
		return nil, err
	}

	return res.([]types.Log), nil
}

func (c *ClientWithFallback) SubscribeFilterLogs(ctx context.Context, q ethereum.FilterQuery, ch chan<- types.Log) (ethereum.Subscription, error) {
	res, err := c.makeCall(
		ctx, MakeCallFunctor{
			MethodName: "eth_SubscribeFilterLogs",
			Func: func(client ethclient2.EthClientInterface) (interface{}, error) {
				return client.SubscribeFilterLogs(ctx, q, ch)
			},
		},
	)
	if err != nil {
		return nil, err
	}

	return res.(ethereum.Subscription), nil
}

func (c *ClientWithFallback) PendingBalanceAt(ctx context.Context, account common.Address) (*big.Int, error) {
	res, err := c.makeCall(
		ctx, MakeCallFunctor{
			MethodName: "eth_PendingBalanceAt",
			Func: func(client ethclient2.EthClientInterface) (interface{}, error) {
				return client.PendingBalanceAt(ctx, account)
			},
		},
	)
	if err != nil {
		return nil, err
	}

	return res.(*big.Int), nil
}

func (c *ClientWithFallback) PendingStorageAt(ctx context.Context, account common.Address, key common.Hash) ([]byte, error) {
	res, err := c.makeCall(
		ctx, MakeCallFunctor{
			MethodName: "eth_PendingStorageAt",
			Func: func(client ethclient2.EthClientInterface) (interface{}, error) {
				return client.PendingStorageAt(ctx, account, key)
			},
		},
	)
	if err != nil {
		return nil, err
	}

	return res.([]byte), nil
}

func (c *ClientWithFallback) PendingCodeAt(ctx context.Context, account common.Address) ([]byte, error) {
	res, err := c.makeCall(
		ctx, MakeCallFunctor{
			MethodName: "eth_PendingCodeAt",
			Func: func(client ethclient2.EthClientInterface) (interface{}, error) {
				return client.PendingCodeAt(ctx, account)
			},
		},
	)
	if err != nil {
		return nil, err
	}

	return res.([]byte), nil
}

func (c *ClientWithFallback) PendingNonceAt(ctx context.Context, account common.Address) (uint64, error) {
	res, err := c.makeCall(
		ctx, MakeCallFunctor{
			MethodName: "eth_PendingNonceAt",
			Func: func(client ethclient2.EthClientInterface) (interface{}, error) {
				return client.PendingNonceAt(ctx, account)
			},
		},
	)
	if err != nil {
		return 0, err
	}

	return res.(uint64), nil
}

func (c *ClientWithFallback) PendingTransactionCount(ctx context.Context) (uint, error) {
	res, err := c.makeCall(
		ctx, MakeCallFunctor{
			MethodName: "eth_PendingTransactionCount",
			Func: func(client ethclient2.EthClientInterface) (interface{}, error) {
				return client.PendingTransactionCount(ctx)
			},
		},
	)
	if err != nil {
		return 0, err
	}

	return res.(uint), nil
}

func (c *ClientWithFallback) CallContract(ctx context.Context, msg ethereum.CallMsg, blockNumber *big.Int) ([]byte, error) {
	res, err := c.makeCall(
		ctx, MakeCallFunctor{
			MethodName: "eth_CallContract",
			Func: func(client ethclient2.EthClientInterface) (interface{}, error) {
				return client.CallContract(ctx, msg, blockNumber)
			},
		},
	)
	if err != nil {
		return nil, err
	}

	return res.([]byte), nil
}

func (c *ClientWithFallback) PendingCallContract(ctx context.Context, msg ethereum.CallMsg) ([]byte, error) {
	res, err := c.makeCall(
		ctx, MakeCallFunctor{
			MethodName: "eth_PendingCallContract",
			Func: func(client ethclient2.EthClientInterface) (interface{}, error) {
				return client.PendingCallContract(ctx, msg)
			},
		},
	)
	if err != nil {
		return nil, err
	}

	return res.([]byte), nil
}

func (c *ClientWithFallback) SuggestGasPrice(ctx context.Context) (*big.Int, error) {
	res, err := c.makeCall(
		ctx, MakeCallFunctor{
			MethodName: "eth_SuggestGasPrice",
			Func: func(client ethclient2.EthClientInterface) (interface{}, error) {
				return client.SuggestGasPrice(ctx)
			},
		},
	)
	if err != nil {
		return nil, err
	}

	return res.(*big.Int), nil
}

func (c *ClientWithFallback) SuggestGasTipCap(ctx context.Context) (*big.Int, error) {
	res, err := c.makeCall(
		ctx, MakeCallFunctor{
			MethodName: "eth_SuggestGasTipCap",
			Func: func(client ethclient2.EthClientInterface) (interface{}, error) {
				return client.SuggestGasTipCap(ctx)
			},
		},
	)
	if err != nil {
		return nil, err
	}

	return res.(*big.Int), nil
}

func (c *ClientWithFallback) FeeHistory(ctx context.Context, blockCount uint64, lastBlock *big.Int, rewardPercentiles []float64) (*ethereum.FeeHistory, error) {
	res, err := c.makeCall(
		ctx, MakeCallFunctor{
			MethodName: "eth_FeeHistory",
			Func: func(client ethclient2.EthClientInterface) (interface{}, error) {
				return client.FeeHistory(ctx, blockCount, lastBlock, rewardPercentiles)
			},
		},
	)
	if err != nil {
		return nil, err
	}

	return res.(*ethereum.FeeHistory), nil
}

func (c *ClientWithFallback) EstimateGas(ctx context.Context, msg ethereum.CallMsg) (uint64, error) {
	res, err := c.makeCall(
		ctx, MakeCallFunctor{
			MethodName: "eth_EstimateGas",
			Func: func(client ethclient2.EthClientInterface) (interface{}, error) {
				return client.EstimateGas(ctx, msg)
			},
		},
	)
	if err != nil {
		return 0, err
	}

	return res.(uint64), nil
}

func (c *ClientWithFallback) LineaEstimateGas(ctx context.Context, msg ethereum.CallMsg) (*sdkethclient.LineaEstimateGasResult, error) {
	res, err := c.makeCall(
		ctx, MakeCallFunctor{
			MethodName: "linea_estimateGas",
			Func: func(client ethclient2.EthClientInterface) (interface{}, error) {
				return client.LineaEstimateGas(ctx, msg)
			},
		},
	)
	if err != nil {
		return nil, err
	}

	return res.(*sdkethclient.LineaEstimateGasResult), nil
}

func (c *ClientWithFallback) SendTransaction(ctx context.Context, tx *types.Transaction) error {
	_, err := c.makeCall(
		ctx, MakeCallFunctor{
			MethodName: "eth_SendTransaction",
			Func: func(client ethclient2.EthClientInterface) (interface{}, error) {
				return nil, client.SendTransaction(ctx, tx)
			},
		},
	)
	return err
}

func (c *ClientWithFallback) CallContext(ctx context.Context, result interface{}, method string, args ...interface{}) error {
	_, err := c.makeCall(
		ctx, MakeCallFunctor{
			MethodName: "eth_CallContext",
			Func: func(client ethclient2.EthClientInterface) (interface{}, error) {
				return nil, client.CallContext(ctx, result, method, args...)
			},
		},
	)
	return err
}

func (c *ClientWithFallback) BatchCallContext(ctx context.Context, b []rpc.BatchElem) error {
	_, err := c.makeCall(
		ctx, MakeCallFunctor{
			MethodName: "eth_BatchCallContext",
			Func: func(client ethclient2.EthClientInterface) (interface{}, error) {
				return nil, client.BatchCallContext(ctx, b)
			},
		},
	)
	return err
}

func convertFunctorCallStatuses(statuses []circuitbreaker.FunctorCallStatus, methodName string) (result []rpcstatus.RpcProviderCallStatus) {
	for _, f := range statuses {
		result = append(result, rpcstatus.RpcProviderCallStatus{
			Name:      f.Name,
			Method:    methodName,
			Timestamp: f.Timestamp,
			Err:       f.Err,
			StartTime: f.StartTime,
		})
	}
	return
}

// Returns provider instance with a specific provider name
func (c *ClientWithFallback) GetProviderClient(provider string) ethclient2.EthClientInterface {
	for _, client := range c.ethClients {
		if client.GetProviderName() == provider {
			return client
		}
	}
	return nil
}
