package chain

//go:generate mockgen -package=mock_client -source=client.go -destination=mock/client/client.go

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"sync/atomic"
	"time"

	"go.uber.org/zap"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/status-im/status-go/circuitbreaker"
	"github.com/status-im/status-go/healthmanager"
	"github.com/status-im/status-go/healthmanager/rpcstatus"
	"github.com/status-im/status-go/logutils"
	"github.com/status-im/status-go/rpc/chain/ethclient"
	"github.com/status-im/status-go/rpc/chain/rpclimiter"
	"github.com/status-im/status-go/rpc/chain/tagger"
	"github.com/status-im/status-go/services/rpcstats"
	"github.com/status-im/status-go/services/wallet/connection"
)

type ClientInterface interface {
	ethclient.EthClientInterface
	NetworkID() uint64
	ToBigInt() *big.Int
	GetWalletNotifier() func(chainId uint64, message string)
	SetWalletNotifier(notifier func(chainId uint64, message string))
	connection.Connectable
	GetLimiter() rpclimiter.RequestLimiter
	SetLimiter(rpclimiter.RequestLimiter)
}

type HealthMonitor interface {
	GetCircuitBreaker() *circuitbreaker.CircuitBreaker
	SetCircuitBreaker(cb *circuitbreaker.CircuitBreaker)
}

type Copyable interface {
	Copy() interface{}
}

// Shallow copy of the client with a deep copy of tag and group tag
// To avoid passing tags as parameter to every chain call, it is sufficient for now
// to set the tag and group tag once on the client
func ClientWithTag(chainClient ClientInterface, tag, groupTag string) ClientInterface {
	newClient := chainClient
	if tagIface, ok := chainClient.(tagger.Tagger); ok {
		tagIface = tagger.DeepCopyTagger(tagIface)
		tagIface.SetTag(tag)
		tagIface.SetGroupTag(groupTag)
		newClient = tagIface.(ClientInterface)
	}

	return newClient
}

type ClientWithFallback struct {
	ChainID                uint64
	ethClients             []ethclient.RPSLimitedEthClientInterface
	commonLimiter          rpclimiter.RequestLimiter
	circuitbreaker         *circuitbreaker.CircuitBreaker
	providersHealthManager *healthmanager.ProvidersHealthManager

	WalletNotifier func(chainId uint64, message string)

	isConnected   *atomic.Bool
	LastCheckedAt int64

	tag      string // tag for the limiter
	groupTag string // tag for the limiter group
}

func (c *ClientWithFallback) Copy() interface{} {
	return &ClientWithFallback{
		ChainID:        c.ChainID,
		ethClients:     c.ethClients,
		commonLimiter:  c.commonLimiter,
		circuitbreaker: c.circuitbreaker,
		WalletNotifier: c.WalletNotifier,
		isConnected:    c.isConnected,
		LastCheckedAt:  c.LastCheckedAt,
		tag:            c.tag,
		groupTag:       c.groupTag,
	}
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

func NewClient(ethClients []ethclient.RPSLimitedEthClientInterface, chainID uint64, providersHealthManager *healthmanager.ProvidersHealthManager) *ClientWithFallback {
	cbConfig := circuitbreaker.Config{
		Timeout:               20000,
		MaxConcurrentRequests: 100,
		SleepWindow:           300000,
		ErrorPercentThreshold: 25,
	}

	isConnected := &atomic.Bool{}
	isConnected.Store(true)

	return &ClientWithFallback{
		ChainID:                chainID,
		ethClients:             ethClients,
		isConnected:            isConnected,
		LastCheckedAt:          time.Now().Unix(),
		circuitbreaker:         circuitbreaker.NewCircuitBreaker(cbConfig),
		providersHealthManager: providersHealthManager,
	}
}

func (c *ClientWithFallback) Close() {
	for _, client := range c.ethClients {
		client.Close()
	}
}

// Not found should not be cancelling the requests, as that's returned
// when we are hitting a non archival node for example, it should continue the
// chain as the next provider might have archival support.
func isNotFoundError(err error) bool {
	return strings.Contains(err.Error(), ethereum.NotFound.Error())
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

func isRPSLimitError(err error) bool {
	return strings.Contains(err.Error(), "backoff_seconds") ||
		strings.Contains(err.Error(), "has exceeded its throughput limit") ||
		strings.Contains(err.Error(), "request rate exceeded")
}

func (c *ClientWithFallback) SetIsConnected(value bool) {
	c.LastCheckedAt = time.Now().Unix()
	if !value {
		if c.isConnected.Load() {
			if c.WalletNotifier != nil {
				c.WalletNotifier(c.ChainID, "down")
			}
			c.isConnected.Store(false)
		}

	} else {
		if !c.isConnected.Load() {
			c.isConnected.Store(true)
			if c.WalletNotifier != nil {
				c.WalletNotifier(c.ChainID, "up")
			}
		}
	}
}

func (c *ClientWithFallback) IsConnected() bool {
	return c.isConnected.Load()
}

func (c *ClientWithFallback) makeCall(ctx context.Context, ethClients []ethclient.RPSLimitedEthClientInterface, f func(client ethclient.RPSLimitedEthClientInterface) (interface{}, error)) (interface{}, error) {
	if c.commonLimiter != nil {
		if allow, err := c.commonLimiter.Allow(c.tag); !allow {
			return nil, fmt.Errorf("tag=%s, %w", c.tag, err)
		}

		if allow, err := c.commonLimiter.Allow(c.groupTag); !allow {
			return nil, fmt.Errorf("groupTag=%s, %w", c.groupTag, err)
		}
	}

	c.LastCheckedAt = time.Now().Unix()

	cmd := circuitbreaker.NewCommand(ctx, nil)
	for _, provider := range ethClients {
		provider := provider
		cmd.Add(circuitbreaker.NewFunctor(func() ([]interface{}, error) {
			limiter := provider.GetLimiter()
			if limiter != nil {
				err := provider.GetLimiter().WaitForRequestsAvailability(1)
				if err != nil {
					return nil, err
				}
			}

			res, err := f(provider)
			if err != nil {
				if limiter != nil && isRPSLimitError(err) {
					provider.GetLimiter().ReduceLimit()

					err = provider.GetLimiter().WaitForRequestsAvailability(1)
					if err != nil {
						return nil, err
					}

					res, err = f(provider)
					if err == nil {
						return []interface{}{res}, err
					}
				}

				if isVMError(err) || errors.Is(err, context.Canceled) {
					cmd.Cancel()
				}

				return nil, err
			}
			return []interface{}{res}, err
		}, provider.GetName()))
	}

	result := c.circuitbreaker.Execute(cmd)
	if c.providersHealthManager != nil {
		rpcCallStatuses := convertFunctorCallStatuses(result.FunctorCallStatuses())
		c.providersHealthManager.Update(ctx, rpcCallStatuses)
	}
	if result.Error() != nil {
		return nil, result.Error()
	}

	return result.Result()[0], nil
}

func (c *ClientWithFallback) BlockByHash(ctx context.Context, hash common.Hash) (*types.Block, error) {
	rpcstats.CountCallWithTag("eth_BlockByHash", c.tag)

	res, err := c.makeCall(
		ctx, c.ethClients, func(client ethclient.RPSLimitedEthClientInterface) (interface{}, error) {
			return client.BlockByHash(ctx, hash)
		},
	)

	c.toggleConnectionState(err)

	if err != nil {
		return nil, err
	}

	return res.(*types.Block), nil
}

func (c *ClientWithFallback) BlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error) {
	rpcstats.CountCallWithTag("eth_BlockByNumber", c.tag)
	res, err := c.makeCall(
		ctx, c.ethClients, func(client ethclient.RPSLimitedEthClientInterface) (interface{}, error) {
			return client.BlockByNumber(ctx, number)
		},
	)

	c.toggleConnectionState(err)

	if err != nil {
		return nil, err
	}

	return res.(*types.Block), nil
}

func (c *ClientWithFallback) BlockNumber(ctx context.Context) (uint64, error) {
	rpcstats.CountCallWithTag("eth_BlockNumber", c.tag)

	res, err := c.makeCall(
		ctx, c.ethClients, func(client ethclient.RPSLimitedEthClientInterface) (interface{}, error) {
			return client.BlockNumber(ctx)
		},
	)

	c.toggleConnectionState(err)

	if err != nil {
		return 0, err
	}

	return res.(uint64), nil
}

func (c *ClientWithFallback) HeaderByHash(ctx context.Context, hash common.Hash) (*types.Header, error) {
	rpcstats.CountCallWithTag("eth_HeaderByHash", c.tag)
	res, err := c.makeCall(
		ctx, c.ethClients, func(client ethclient.RPSLimitedEthClientInterface) (interface{}, error) {
			return client.HeaderByHash(ctx, hash)
		},
	)

	c.toggleConnectionState(err)

	if err != nil {
		return nil, err
	}

	return res.(*types.Header), nil
}

func (c *ClientWithFallback) HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error) {
	rpcstats.CountCallWithTag("eth_HeaderByNumber", c.tag)
	res, err := c.makeCall(
		ctx, c.ethClients, func(client ethclient.RPSLimitedEthClientInterface) (interface{}, error) {
			return client.HeaderByNumber(ctx, number)
		},
	)

	c.toggleConnectionState(err)

	if err != nil {
		return nil, err
	}

	return res.(*types.Header), nil
}

func (c *ClientWithFallback) TransactionByHash(ctx context.Context, hash common.Hash) (*types.Transaction, bool, error) {
	rpcstats.CountCallWithTag("eth_TransactionByHash", c.tag)

	res, err := c.makeCall(
		ctx, c.ethClients, func(client ethclient.RPSLimitedEthClientInterface) (interface{}, error) {
			tx, isPending, err := client.TransactionByHash(ctx, hash)
			return []any{tx, isPending}, err
		},
	)

	c.toggleConnectionState(err)

	if err != nil {
		return nil, false, err
	}

	resArr := res.([]any)
	return resArr[0].(*types.Transaction), resArr[1].(bool), nil
}

func (c *ClientWithFallback) TransactionSender(ctx context.Context, tx *types.Transaction, block common.Hash, index uint) (common.Address, error) {
	rpcstats.CountCall("eth_TransactionSender")

	res, err := c.makeCall(
		ctx, c.ethClients, func(client ethclient.RPSLimitedEthClientInterface) (interface{}, error) {
			return client.TransactionSender(ctx, tx, block, index)
		},
	)

	c.toggleConnectionState(err)

	return res.(common.Address), err
}

func (c *ClientWithFallback) TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error) {
	rpcstats.CountCall("eth_TransactionReceipt")

	res, err := c.makeCall(
		ctx, c.ethClients, func(client ethclient.RPSLimitedEthClientInterface) (interface{}, error) {
			return client.TransactionReceipt(ctx, txHash)
		},
	)

	c.toggleConnectionState(err)

	if err != nil {
		return nil, err
	}

	return res.(*types.Receipt), nil
}

func (c *ClientWithFallback) SyncProgress(ctx context.Context) (*ethereum.SyncProgress, error) {
	rpcstats.CountCall("eth_SyncProgress")

	res, err := c.makeCall(
		ctx, c.ethClients, func(client ethclient.RPSLimitedEthClientInterface) (interface{}, error) {
			return client.SyncProgress(ctx)
		},
	)

	c.toggleConnectionState(err)

	if err != nil {
		return nil, err
	}

	return res.(*ethereum.SyncProgress), nil
}

func (c *ClientWithFallback) NetworkID() uint64 {
	return c.ChainID
}

func (c *ClientWithFallback) BalanceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (*big.Int, error) {
	rpcstats.CountCallWithTag("eth_BalanceAt", c.tag)

	res, err := c.makeCall(
		ctx, c.ethClients, func(client ethclient.RPSLimitedEthClientInterface) (interface{}, error) {
			return client.BalanceAt(ctx, account, blockNumber)
		},
	)

	c.toggleConnectionState(err)

	if err != nil {
		return nil, err
	}

	return res.(*big.Int), nil
}

func (c *ClientWithFallback) StorageAt(ctx context.Context, account common.Address, key common.Hash, blockNumber *big.Int) ([]byte, error) {
	rpcstats.CountCall("eth_StorageAt")

	res, err := c.makeCall(
		ctx, c.ethClients, func(client ethclient.RPSLimitedEthClientInterface) (interface{}, error) {
			return client.StorageAt(ctx, account, key, blockNumber)
		},
	)

	c.toggleConnectionState(err)

	if err != nil {
		return nil, err
	}

	return res.([]byte), nil
}

func (c *ClientWithFallback) CodeAt(ctx context.Context, account common.Address, blockNumber *big.Int) ([]byte, error) {
	rpcstats.CountCall("eth_CodeAt")

	res, err := c.makeCall(
		ctx, c.ethClients, func(client ethclient.RPSLimitedEthClientInterface) (interface{}, error) {
			return client.CodeAt(ctx, account, blockNumber)
		},
	)

	c.toggleConnectionState(err)

	if err != nil {
		return nil, err
	}

	return res.([]byte), nil
}

func (c *ClientWithFallback) NonceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (uint64, error) {
	rpcstats.CountCallWithTag("eth_NonceAt", c.tag)

	res, err := c.makeCall(
		ctx, c.ethClients, func(client ethclient.RPSLimitedEthClientInterface) (interface{}, error) {
			return client.NonceAt(ctx, account, blockNumber)
		},
	)

	c.toggleConnectionState(err)

	if err != nil {
		return 0, err
	}

	return res.(uint64), nil
}

func (c *ClientWithFallback) FilterLogs(ctx context.Context, q ethereum.FilterQuery) ([]types.Log, error) {
	rpcstats.CountCallWithTag("eth_FilterLogs", c.tag)

	// Override providers name to use a separate circuit for this command as it more often fails due to rate limiting
	ethClients := make([]ethclient.RPSLimitedEthClientInterface, len(c.ethClients))
	for i, client := range c.ethClients {
		ethClients[i] = client.CopyWithName(client.GetName() + "_FilterLogs")
	}

	res, err := c.makeCall(
		ctx, c.ethClients, func(client ethclient.RPSLimitedEthClientInterface) (interface{}, error) {
			return client.FilterLogs(ctx, q)
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
	rpcstats.CountCall("eth_SubscribeFilterLogs")

	res, err := c.makeCall(
		ctx, c.ethClients, func(client ethclient.RPSLimitedEthClientInterface) (interface{}, error) {
			return client.SubscribeFilterLogs(ctx, q, ch)
		},
	)

	c.toggleConnectionState(err)

	if err != nil {
		return nil, err
	}

	return res.(ethereum.Subscription), nil
}

func (c *ClientWithFallback) PendingBalanceAt(ctx context.Context, account common.Address) (*big.Int, error) {
	rpcstats.CountCall("eth_PendingBalanceAt")

	res, err := c.makeCall(
		ctx, c.ethClients, func(client ethclient.RPSLimitedEthClientInterface) (interface{}, error) {
			return client.PendingBalanceAt(ctx, account)
		},
	)

	c.toggleConnectionState(err)

	if err != nil {
		return nil, err
	}

	return res.(*big.Int), nil
}

func (c *ClientWithFallback) PendingStorageAt(ctx context.Context, account common.Address, key common.Hash) ([]byte, error) {
	rpcstats.CountCall("eth_PendingStorageAt")

	res, err := c.makeCall(
		ctx, c.ethClients, func(client ethclient.RPSLimitedEthClientInterface) (interface{}, error) {
			return client.PendingStorageAt(ctx, account, key)
		},
	)

	c.toggleConnectionState(err)

	if err != nil {
		return nil, err
	}

	return res.([]byte), nil
}

func (c *ClientWithFallback) PendingCodeAt(ctx context.Context, account common.Address) ([]byte, error) {
	rpcstats.CountCall("eth_PendingCodeAt")

	res, err := c.makeCall(
		ctx, c.ethClients, func(client ethclient.RPSLimitedEthClientInterface) (interface{}, error) {
			return client.PendingCodeAt(ctx, account)
		},
	)

	c.toggleConnectionState(err)

	if err != nil {
		return nil, err
	}

	return res.([]byte), nil
}

func (c *ClientWithFallback) PendingNonceAt(ctx context.Context, account common.Address) (uint64, error) {
	rpcstats.CountCall("eth_PendingNonceAt")

	res, err := c.makeCall(
		ctx, c.ethClients, func(client ethclient.RPSLimitedEthClientInterface) (interface{}, error) {
			return client.PendingNonceAt(ctx, account)
		},
	)

	c.toggleConnectionState(err)

	if err != nil {
		return 0, err
	}

	return res.(uint64), nil
}

func (c *ClientWithFallback) PendingTransactionCount(ctx context.Context) (uint, error) {
	rpcstats.CountCall("eth_PendingTransactionCount")

	res, err := c.makeCall(
		ctx, c.ethClients, func(client ethclient.RPSLimitedEthClientInterface) (interface{}, error) {
			return client.PendingTransactionCount(ctx)
		},
	)

	c.toggleConnectionState(err)

	if err != nil {
		return 0, err
	}

	return res.(uint), nil
}

func (c *ClientWithFallback) CallContract(ctx context.Context, msg ethereum.CallMsg, blockNumber *big.Int) ([]byte, error) {
	rpcstats.CountCall("eth_CallContract_" + msg.To.String())

	res, err := c.makeCall(
		ctx, c.ethClients, func(client ethclient.RPSLimitedEthClientInterface) (interface{}, error) {
			return client.CallContract(ctx, msg, blockNumber)
		},
	)

	c.toggleConnectionState(err)

	if err != nil {
		return nil, err
	}

	return res.([]byte), nil
}

func (c *ClientWithFallback) PendingCallContract(ctx context.Context, msg ethereum.CallMsg) ([]byte, error) {
	rpcstats.CountCall("eth_PendingCallContract")

	res, err := c.makeCall(
		ctx, c.ethClients, func(client ethclient.RPSLimitedEthClientInterface) (interface{}, error) {
			return client.PendingCallContract(ctx, msg)
		},
	)

	c.toggleConnectionState(err)

	if err != nil {
		return nil, err
	}

	return res.([]byte), nil
}

func (c *ClientWithFallback) SuggestGasPrice(ctx context.Context) (*big.Int, error) {
	rpcstats.CountCall("eth_SuggestGasPrice")

	res, err := c.makeCall(
		ctx, c.ethClients, func(client ethclient.RPSLimitedEthClientInterface) (interface{}, error) {
			return client.SuggestGasPrice(ctx)
		},
	)

	c.toggleConnectionState(err)

	if err != nil {
		return nil, err
	}

	return res.(*big.Int), nil
}

func (c *ClientWithFallback) SuggestGasTipCap(ctx context.Context) (*big.Int, error) {
	rpcstats.CountCall("eth_SuggestGasTipCap")

	res, err := c.makeCall(
		ctx, c.ethClients, func(client ethclient.RPSLimitedEthClientInterface) (interface{}, error) {
			return client.SuggestGasTipCap(ctx)
		},
	)

	c.toggleConnectionState(err)

	if err != nil {
		return nil, err
	}

	return res.(*big.Int), nil
}

func (c *ClientWithFallback) FeeHistory(ctx context.Context, blockCount uint64, lastBlock *big.Int, rewardPercentiles []float64) (*ethereum.FeeHistory, error) {
	rpcstats.CountCall("eth_FeeHistory")

	res, err := c.makeCall(
		ctx, c.ethClients, func(client ethclient.RPSLimitedEthClientInterface) (interface{}, error) {
			return client.FeeHistory(ctx, blockCount, lastBlock, rewardPercentiles)
		},
	)

	c.toggleConnectionState(err)

	if err != nil {
		return nil, err
	}

	return res.(*ethereum.FeeHistory), nil
}

func (c *ClientWithFallback) EstimateGas(ctx context.Context, msg ethereum.CallMsg) (uint64, error) {
	rpcstats.CountCall("eth_EstimateGas")

	res, err := c.makeCall(
		ctx, c.ethClients, func(client ethclient.RPSLimitedEthClientInterface) (interface{}, error) {
			return client.EstimateGas(ctx, msg)
		},
	)

	c.toggleConnectionState(err)

	if err != nil {
		return 0, err
	}

	return res.(uint64), nil
}

func (c *ClientWithFallback) SendTransaction(ctx context.Context, tx *types.Transaction) error {
	rpcstats.CountCall("eth_SendTransaction")

	_, err := c.makeCall(
		ctx, c.ethClients, func(client ethclient.RPSLimitedEthClientInterface) (interface{}, error) {
			return nil, client.SendTransaction(ctx, tx)
		},
	)

	c.toggleConnectionState(err)

	return err
}

func (c *ClientWithFallback) CallContext(ctx context.Context, result interface{}, method string, args ...interface{}) error {
	rpcstats.CountCall("eth_CallContext")

	_, err := c.makeCall(
		ctx, c.ethClients, func(client ethclient.RPSLimitedEthClientInterface) (interface{}, error) {
			return nil, client.CallContext(ctx, result, method, args...)
		},
	)

	c.toggleConnectionState(err)

	return err
}

func (c *ClientWithFallback) BatchCallContext(ctx context.Context, b []rpc.BatchElem) error {
	rpcstats.CountCall("eth_BatchCallContext")

	_, err := c.makeCall(
		ctx, c.ethClients, func(client ethclient.RPSLimitedEthClientInterface) (interface{}, error) {
			return nil, client.BatchCallContext(ctx, b)
		},
	)

	c.toggleConnectionState(err)

	return err
}

func (c *ClientWithFallback) ToBigInt() *big.Int {
	return big.NewInt(int64(c.ChainID))
}

func (c *ClientWithFallback) GetBaseFeeFromBlock(ctx context.Context, blockNumber *big.Int) (string, error) {
	rpcstats.CountCall("eth_GetBaseFeeFromBlock")

	feeHistory, err := c.FeeHistory(ctx, 1, blockNumber, nil)

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

func (c *ClientWithFallback) GetWalletNotifier() func(chainId uint64, message string) {
	return c.WalletNotifier
}

func (c *ClientWithFallback) SetWalletNotifier(notifier func(chainId uint64, message string)) {
	c.WalletNotifier = notifier
}

func (c *ClientWithFallback) toggleConnectionState(err error) {
	connected := true
	if err != nil {
		if !isNotFoundError(err) && !isVMError(err) && !errors.Is(err, rpclimiter.ErrRequestsOverLimit) && !errors.Is(err, context.Canceled) {
			logutils.ZapLogger().Warn("Error not in chain call", zap.Uint64("chain", c.ChainID), zap.Error(err))
			connected = false
		} else {
			logutils.ZapLogger().Warn("Error in chain call", zap.Error(err))
		}
	}
	c.SetIsConnected(connected)
}

func (c *ClientWithFallback) Tag() string {
	return c.tag
}

func (c *ClientWithFallback) SetTag(tag string) {
	c.tag = tag
}

func (c *ClientWithFallback) GroupTag() string {
	return c.groupTag
}

func (c *ClientWithFallback) SetGroupTag(tag string) {
	c.groupTag = tag
}

func (c *ClientWithFallback) DeepCopyTag() tagger.Tagger {
	copy := *c
	return &copy
}

func (c *ClientWithFallback) GetLimiter() rpclimiter.RequestLimiter {
	return c.commonLimiter
}

func (c *ClientWithFallback) SetLimiter(limiter rpclimiter.RequestLimiter) {
	c.commonLimiter = limiter
}

func (c *ClientWithFallback) GetCircuitBreaker() *circuitbreaker.CircuitBreaker {
	return c.circuitbreaker
}

func (c *ClientWithFallback) SetCircuitBreaker(cb *circuitbreaker.CircuitBreaker) {
	c.circuitbreaker = cb
}

func convertFunctorCallStatuses(statuses []circuitbreaker.FunctorCallStatus) (result []rpcstatus.RpcProviderCallStatus) {
	for _, f := range statuses {
		result = append(result, rpcstatus.RpcProviderCallStatus{Name: f.Name, Timestamp: f.Timestamp, Err: f.Err})
	}
	return
}
