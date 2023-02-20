package chain

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/afex/hystrix-go/hystrix"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/status-im/status-go/services/rpcstats"
)

type FeeHistory struct {
	BaseFeePerGas []string `json:"baseFeePerGas"`
}

type ClientWithFallback struct {
	ChainID  uint64
	main     *ethclient.Client
	fallback *ethclient.Client

	mainRPC     *rpc.Client
	fallbackRPC *rpc.Client

	IsConnected   bool
	LastCheckedAt int64
}

func NewSimpleClient(main *rpc.Client, chainID uint64) *ClientWithFallback {
	hystrix.ConfigureCommand(fmt.Sprintf("ethClient_%d", chainID), hystrix.CommandConfig{
		Timeout:               10000,
		MaxConcurrentRequests: 100,
		SleepWindow:           300000,
		ErrorPercentThreshold: 25,
	})

	return &ClientWithFallback{
		ChainID:       chainID,
		main:          ethclient.NewClient(main),
		fallback:      nil,
		mainRPC:       main,
		fallbackRPC:   nil,
		IsConnected:   true,
		LastCheckedAt: time.Now().Unix(),
	}
}

func NewClient(main, fallback *rpc.Client, chainID uint64) *ClientWithFallback {
	hystrix.ConfigureCommand(fmt.Sprintf("ethClient_%d", chainID), hystrix.CommandConfig{
		Timeout:               10000,
		MaxConcurrentRequests: 100,
		SleepWindow:           300000,
		ErrorPercentThreshold: 25,
	})

	var fallbackEthClient *ethclient.Client
	if fallback != nil {
		fallbackEthClient = ethclient.NewClient(fallback)
	}
	return &ClientWithFallback{
		ChainID:       chainID,
		main:          ethclient.NewClient(main),
		fallback:      fallbackEthClient,
		mainRPC:       main,
		fallbackRPC:   fallback,
		IsConnected:   true,
		LastCheckedAt: time.Now().Unix(),
	}
}

func (c *ClientWithFallback) Close() {
	c.main.Close()
	if c.fallback != nil {
		c.fallback.Close()
	}
}

func (c *ClientWithFallback) makeCallNoReturn(main func() error, fallback func() error) error {
	output := make(chan struct{}, 1)
	c.LastCheckedAt = time.Now().Unix()
	errChan := hystrix.Go(fmt.Sprintf("ethClient_%d", c.ChainID), func() error {
		err := main()
		if err != nil {
			return err
		}
		c.IsConnected = true
		output <- struct{}{}
		return nil
	}, func(err error) error {
		if c.fallback == nil {
			return err
		}

		err = fallback()
		if err != nil {
			c.IsConnected = false
			return err
		}
		c.IsConnected = true
		output <- struct{}{}
		return nil
	})

	select {
	case <-output:
		return nil
	case err := <-errChan:
		return err
	}
}

func (c *ClientWithFallback) makeCallSingleReturn(main func() (any, error), fallback func() (any, error)) (any, error) {
	resultChan := make(chan any, 1)
	c.LastCheckedAt = time.Now().Unix()
	errChan := hystrix.Go(fmt.Sprintf("ethClient_%d", c.ChainID), func() error {
		res, err := main()
		if err != nil {
			return err
		}
		c.IsConnected = true
		resultChan <- res
		return nil
	}, func(err error) error {
		if c.fallback == nil {
			return err
		}

		res, err := fallback()
		if err != nil {
			c.IsConnected = false
			return err
		}
		c.IsConnected = true
		resultChan <- res
		return nil
	})
	select {
	case result := <-resultChan:
		return result, nil
	case err := <-errChan:

		return nil, err
	}
}

func (c *ClientWithFallback) makeCallDoubleReturn(main func() (any, any, error), fallback func() (any, any, error)) (any, any, error) {
	resultChan := make(chan []any, 1)
	c.LastCheckedAt = time.Now().Unix()
	errChan := hystrix.Go(fmt.Sprintf("ethClient_%d", c.ChainID), func() error {
		a, b, err := main()
		if err != nil {
			return err
		}
		c.IsConnected = true
		resultChan <- []any{a, b}
		return nil
	}, func(err error) error {
		if c.fallback == nil {
			return err
		}

		a, b, err := fallback()
		if err != nil {
			c.IsConnected = false
			return err
		}
		c.IsConnected = true
		resultChan <- []any{a, b}
		return nil
	})

	select {
	case result := <-resultChan:
		return result[0], result[1], nil
	case err := <-errChan:
		return nil, nil, err
	}
}

func (c *ClientWithFallback) BlockByHash(ctx context.Context, hash common.Hash) (*types.Block, error) {
	rpcstats.CountCall("eth_BlockByHash")

	block, err := c.makeCallSingleReturn(
		func() (any, error) { return c.main.BlockByHash(ctx, hash) },
		func() (any, error) { return c.fallback.BlockByHash(ctx, hash) },
	)

	if err != nil {
		return nil, err
	}

	return block.(*types.Block), nil
}

func (c *ClientWithFallback) BlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error) {
	rpcstats.CountCall("eth_BlockByNumber")
	block, err := c.makeCallSingleReturn(
		func() (any, error) { return c.main.BlockByNumber(ctx, number) },
		func() (any, error) { return c.fallback.BlockByNumber(ctx, number) },
	)

	if err != nil {
		return nil, err
	}

	return block.(*types.Block), nil
}

func (c *ClientWithFallback) BlockNumber(ctx context.Context) (uint64, error) {
	rpcstats.CountCall("eth_BlockNumber")

	number, err := c.makeCallSingleReturn(
		func() (any, error) { return c.main.BlockNumber(ctx) },
		func() (any, error) { return c.fallback.BlockNumber(ctx) },
	)

	if err != nil {
		return 0, err
	}

	return number.(uint64), nil
}

func (c *ClientWithFallback) PeerCount(ctx context.Context) (uint64, error) {
	rpcstats.CountCall("eth_PeerCount")

	peerCount, err := c.makeCallSingleReturn(
		func() (any, error) { return c.main.PeerCount(ctx) },
		func() (any, error) { return c.fallback.PeerCount(ctx) },
	)

	if err != nil {
		return 0, err
	}

	return peerCount.(uint64), nil
}

func (c *ClientWithFallback) HeaderByHash(ctx context.Context, hash common.Hash) (*types.Header, error) {
	rpcstats.CountCall("eth_HeaderByHash")
	header, err := c.makeCallSingleReturn(
		func() (any, error) { return c.main.HeaderByHash(ctx, hash) },
		func() (any, error) { return c.fallback.HeaderByHash(ctx, hash) },
	)

	if err != nil {
		return nil, err
	}

	return header.(*types.Header), nil
}

func (c *ClientWithFallback) HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error) {
	rpcstats.CountCall("eth_HeaderByNumber")
	header, err := c.makeCallSingleReturn(
		func() (any, error) { return c.main.HeaderByNumber(ctx, number) },
		func() (any, error) { return c.fallback.HeaderByNumber(ctx, number) },
	)

	if err != nil {
		return nil, err
	}

	return header.(*types.Header), nil
}

func (c *ClientWithFallback) TransactionByHash(ctx context.Context, hash common.Hash) (*types.Transaction, bool, error) {
	rpcstats.CountCall("eth_TransactionByHash")

	tx, isPending, err := c.makeCallDoubleReturn(
		func() (any, any, error) { return c.main.TransactionByHash(ctx, hash) },
		func() (any, any, error) { return c.fallback.TransactionByHash(ctx, hash) },
	)

	if err != nil {
		return nil, false, err
	}

	return tx.(*types.Transaction), isPending.(bool), nil
}

func (c *ClientWithFallback) TransactionSender(ctx context.Context, tx *types.Transaction, block common.Hash, index uint) (common.Address, error) {
	rpcstats.CountCall("eth_TransactionSender")

	address, err := c.makeCallSingleReturn(
		func() (any, error) { return c.main.TransactionSender(ctx, tx, block, index) },
		func() (any, error) { return c.fallback.TransactionSender(ctx, tx, block, index) },
	)

	return address.(common.Address), err
}

func (c *ClientWithFallback) TransactionCount(ctx context.Context, blockHash common.Hash) (uint, error) {
	rpcstats.CountCall("eth_TransactionCount")

	count, err := c.makeCallSingleReturn(
		func() (any, error) { return c.main.TransactionCount(ctx, blockHash) },
		func() (any, error) { return c.fallback.TransactionCount(ctx, blockHash) },
	)

	if err != nil {
		return 0, err
	}

	return count.(uint), nil
}

func (c *ClientWithFallback) TransactionInBlock(ctx context.Context, blockHash common.Hash, index uint) (*types.Transaction, error) {
	rpcstats.CountCall("eth_TransactionInBlock")

	transactions, err := c.makeCallSingleReturn(
		func() (any, error) { return c.main.TransactionInBlock(ctx, blockHash, index) },
		func() (any, error) { return c.fallback.TransactionInBlock(ctx, blockHash, index) },
	)

	if err != nil {
		return nil, err
	}

	return transactions.(*types.Transaction), nil
}

func (c *ClientWithFallback) TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error) {
	rpcstats.CountCall("eth_TransactionReceipt")

	receipt, err := c.makeCallSingleReturn(
		func() (any, error) { return c.main.TransactionReceipt(ctx, txHash) },
		func() (any, error) { return c.fallback.TransactionReceipt(ctx, txHash) },
	)

	if err != nil {
		return nil, err
	}

	return receipt.(*types.Receipt), nil
}

func (c *ClientWithFallback) SyncProgress(ctx context.Context) (*ethereum.SyncProgress, error) {
	rpcstats.CountCall("eth_SyncProgress")

	progress, err := c.makeCallSingleReturn(
		func() (any, error) { return c.main.SyncProgress(ctx) },
		func() (any, error) { return c.fallback.SyncProgress(ctx) },
	)

	if err != nil {
		return nil, err
	}

	return progress.(*ethereum.SyncProgress), nil
}

func (c *ClientWithFallback) SubscribeNewHead(ctx context.Context, ch chan<- *types.Header) (ethereum.Subscription, error) {
	rpcstats.CountCall("eth_SubscribeNewHead")

	sub, err := c.makeCallSingleReturn(
		func() (any, error) { return c.main.SubscribeNewHead(ctx, ch) },
		func() (any, error) { return c.fallback.SubscribeNewHead(ctx, ch) },
	)

	if err != nil {
		return nil, err
	}

	return sub.(ethereum.Subscription), nil
}

func (c *ClientWithFallback) NetworkID(ctx context.Context) (*big.Int, error) {
	rpcstats.CountCall("eth_NetworkID")

	networkID, err := c.makeCallSingleReturn(
		func() (any, error) { return c.main.NetworkID(ctx) },
		func() (any, error) { return c.fallback.NetworkID(ctx) },
	)

	if err != nil {
		return nil, err
	}

	return networkID.(*big.Int), nil
}

func (c *ClientWithFallback) BalanceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (*big.Int, error) {
	rpcstats.CountCall("eth_BalanceAt")

	balance, err := c.makeCallSingleReturn(
		func() (any, error) { return c.main.BalanceAt(ctx, account, blockNumber) },
		func() (any, error) { return c.fallback.BalanceAt(ctx, account, blockNumber) },
	)

	if err != nil {
		return nil, err
	}

	return balance.(*big.Int), nil
}

func (c *ClientWithFallback) StorageAt(ctx context.Context, account common.Address, key common.Hash, blockNumber *big.Int) ([]byte, error) {
	rpcstats.CountCall("eth_StorageAt")

	storage, err := c.makeCallSingleReturn(
		func() (any, error) { return c.main.StorageAt(ctx, account, key, blockNumber) },
		func() (any, error) { return c.fallback.StorageAt(ctx, account, key, blockNumber) },
	)

	if err != nil {
		return nil, err
	}

	return storage.([]byte), nil
}

func (c *ClientWithFallback) CodeAt(ctx context.Context, account common.Address, blockNumber *big.Int) ([]byte, error) {
	rpcstats.CountCall("eth_CodeAt")

	code, err := c.makeCallSingleReturn(
		func() (any, error) { return c.main.CodeAt(ctx, account, blockNumber) },
		func() (any, error) { return c.fallback.CodeAt(ctx, account, blockNumber) },
	)

	if err != nil {
		return nil, err
	}

	return code.([]byte), nil
}

func (c *ClientWithFallback) NonceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (uint64, error) {
	rpcstats.CountCall("eth_NonceAt")

	nonce, err := c.makeCallSingleReturn(
		func() (any, error) { return c.main.NonceAt(ctx, account, blockNumber) },
		func() (any, error) { return c.fallback.NonceAt(ctx, account, blockNumber) },
	)

	if err != nil {
		return 0, err
	}

	return nonce.(uint64), nil
}

func (c *ClientWithFallback) FilterLogs(ctx context.Context, q ethereum.FilterQuery) ([]types.Log, error) {
	rpcstats.CountCall("eth_FilterLogs")

	logs, err := c.makeCallSingleReturn(
		func() (any, error) { return c.main.FilterLogs(ctx, q) },
		func() (any, error) { return c.fallback.FilterLogs(ctx, q) },
	)

	if err != nil {
		return nil, err
	}

	return logs.([]types.Log), nil
}

func (c *ClientWithFallback) SubscribeFilterLogs(ctx context.Context, q ethereum.FilterQuery, ch chan<- types.Log) (ethereum.Subscription, error) {
	rpcstats.CountCall("eth_SubscribeFilterLogs")

	sub, err := c.makeCallSingleReturn(
		func() (any, error) { return c.main.SubscribeFilterLogs(ctx, q, ch) },
		func() (any, error) { return c.fallback.SubscribeFilterLogs(ctx, q, ch) },
	)

	if err != nil {
		return nil, err
	}

	return sub.(ethereum.Subscription), nil
}

func (c *ClientWithFallback) PendingBalanceAt(ctx context.Context, account common.Address) (*big.Int, error) {
	rpcstats.CountCall("eth_PendingBalanceAt")

	balance, err := c.makeCallSingleReturn(
		func() (any, error) { return c.main.PendingBalanceAt(ctx, account) },
		func() (any, error) { return c.fallback.PendingBalanceAt(ctx, account) },
	)

	if err != nil {
		return nil, err
	}

	return balance.(*big.Int), nil
}

func (c *ClientWithFallback) PendingStorageAt(ctx context.Context, account common.Address, key common.Hash) ([]byte, error) {
	rpcstats.CountCall("eth_PendingStorageAt")

	storage, err := c.makeCallSingleReturn(
		func() (any, error) { return c.main.PendingStorageAt(ctx, account, key) },
		func() (any, error) { return c.fallback.PendingStorageAt(ctx, account, key) },
	)

	if err != nil {
		return nil, err
	}

	return storage.([]byte), nil
}

func (c *ClientWithFallback) PendingCodeAt(ctx context.Context, account common.Address) ([]byte, error) {
	rpcstats.CountCall("eth_PendingCodeAt")

	code, err := c.makeCallSingleReturn(
		func() (any, error) { return c.main.PendingCodeAt(ctx, account) },
		func() (any, error) { return c.fallback.PendingCodeAt(ctx, account) },
	)

	if err != nil {
		return nil, err
	}

	return code.([]byte), nil
}

func (c *ClientWithFallback) PendingNonceAt(ctx context.Context, account common.Address) (uint64, error) {
	rpcstats.CountCall("eth_PendingNonceAt")

	nonce, err := c.makeCallSingleReturn(
		func() (any, error) { return c.main.PendingNonceAt(ctx, account) },
		func() (any, error) { return c.fallback.PendingNonceAt(ctx, account) },
	)

	if err != nil {
		return 0, err
	}

	return nonce.(uint64), nil
}

func (c *ClientWithFallback) PendingTransactionCount(ctx context.Context) (uint, error) {
	rpcstats.CountCall("eth_PendingTransactionCount")

	count, err := c.makeCallSingleReturn(
		func() (any, error) { return c.main.PendingTransactionCount(ctx) },
		func() (any, error) { return c.fallback.PendingTransactionCount(ctx) },
	)

	if err != nil {
		return 0, err
	}

	return count.(uint), nil
}

func (c *ClientWithFallback) CallContract(ctx context.Context, msg ethereum.CallMsg, blockNumber *big.Int) ([]byte, error) {
	rpcstats.CountCall("eth_CallContract")

	data, err := c.makeCallSingleReturn(
		func() (any, error) { return c.main.CallContract(ctx, msg, blockNumber) },
		func() (any, error) { return c.fallback.CallContract(ctx, msg, blockNumber) },
	)

	if err != nil {
		return nil, err
	}

	return data.([]byte), nil
}

func (c *ClientWithFallback) CallContractAtHash(ctx context.Context, msg ethereum.CallMsg, blockHash common.Hash) ([]byte, error) {
	rpcstats.CountCall("eth_CallContractAtHash")

	data, err := c.makeCallSingleReturn(
		func() (any, error) { return c.main.CallContractAtHash(ctx, msg, blockHash) },
		func() (any, error) { return c.fallback.CallContractAtHash(ctx, msg, blockHash) },
	)

	if err != nil {
		return nil, err
	}

	return data.([]byte), nil
}

func (c *ClientWithFallback) PendingCallContract(ctx context.Context, msg ethereum.CallMsg) ([]byte, error) {
	rpcstats.CountCall("eth_PendingCallContract")

	data, err := c.makeCallSingleReturn(
		func() (any, error) { return c.main.PendingCallContract(ctx, msg) },
		func() (any, error) { return c.fallback.PendingCallContract(ctx, msg) },
	)

	if err != nil {
		return nil, err
	}

	return data.([]byte), nil
}

func (c *ClientWithFallback) SuggestGasPrice(ctx context.Context) (*big.Int, error) {
	rpcstats.CountCall("eth_SuggestGasPrice")

	gasPrice, err := c.makeCallSingleReturn(
		func() (any, error) { return c.main.SuggestGasPrice(ctx) },
		func() (any, error) { return c.fallback.SuggestGasPrice(ctx) },
	)

	if err != nil {
		return nil, err
	}

	return gasPrice.(*big.Int), nil
}

func (c *ClientWithFallback) SuggestGasTipCap(ctx context.Context) (*big.Int, error) {
	rpcstats.CountCall("eth_SuggestGasTipCap")

	tip, err := c.makeCallSingleReturn(
		func() (any, error) { return c.main.SuggestGasTipCap(ctx) },
		func() (any, error) { return c.fallback.SuggestGasTipCap(ctx) },
	)

	if err != nil {
		return nil, err
	}

	return tip.(*big.Int), nil
}

func (c *ClientWithFallback) FeeHistory(ctx context.Context, blockCount uint64, lastBlock *big.Int, rewardPercentiles []float64) (*ethereum.FeeHistory, error) {
	rpcstats.CountCall("eth_FeeHistory")

	feeHistory, err := c.makeCallSingleReturn(
		func() (any, error) { return c.main.FeeHistory(ctx, blockCount, lastBlock, rewardPercentiles) },
		func() (any, error) { return c.fallback.FeeHistory(ctx, blockCount, lastBlock, rewardPercentiles) },
	)

	if err != nil {
		return nil, err
	}

	return feeHistory.(*ethereum.FeeHistory), nil
}

func (c *ClientWithFallback) EstimateGas(ctx context.Context, msg ethereum.CallMsg) (uint64, error) {
	rpcstats.CountCall("eth_EstimateGas")

	estimate, err := c.makeCallSingleReturn(
		func() (any, error) { return c.main.EstimateGas(ctx, msg) },
		func() (any, error) { return c.fallback.EstimateGas(ctx, msg) },
	)

	if err != nil {
		return 0, err
	}

	return estimate.(uint64), nil
}

func (c *ClientWithFallback) SendTransaction(ctx context.Context, tx *types.Transaction) error {
	rpcstats.CountCall("eth_SendTransaction")

	return c.makeCallNoReturn(
		func() error { return c.main.SendTransaction(ctx, tx) },
		func() error { return c.fallback.SendTransaction(ctx, tx) },
	)
}

func (c *ClientWithFallback) CallContext(ctx context.Context, result interface{}, method string, args ...interface{}) error {
	rpcstats.CountCall("eth_CallContext")

	return c.makeCallNoReturn(
		func() error { return c.mainRPC.CallContext(ctx, result, method, args...) },
		func() error { return c.fallbackRPC.CallContext(ctx, result, method, args...) },
	)
}

func (c *ClientWithFallback) ToBigInt() *big.Int {
	return big.NewInt(int64(c.ChainID))
}

func (c *ClientWithFallback) GetBaseFeeFromBlock(blockNumber *big.Int) (string, error) {
	rpcstats.CountCall("eth_GetBaseFeeFromBlock")
	var feeHistory FeeHistory
	err := c.mainRPC.Call(&feeHistory, "eth_feeHistory", "0x1", (*hexutil.Big)(blockNumber), nil)
	if err != nil {
		if err.Error() == "the method eth_feeHistory does not exist/is not available" {
			return "", nil
		}
		return "", err
	}

	var baseGasFee string = ""
	if len(feeHistory.BaseFeePerGas) > 0 {
		baseGasFee = feeHistory.BaseFeePerGas[0]
	}

	return baseGasFee, err
}
