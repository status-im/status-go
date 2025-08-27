package ethclient

import (
	"context"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

var ErrGethEthClientNotSupported = errors.New("method not supported when using a non-geth rpc client")

// Go-ethereum ethclient compatible methods

func (c *Client) BlockNumber(ctx context.Context) (uint64, error) {
	if c.gethEthClient == nil {
		return 0, ErrGethEthClientNotSupported
	}
	return c.gethEthClient.BlockNumber(ctx)
}

func (c *Client) ChainID(ctx context.Context) (*big.Int, error) {
	if c.gethEthClient == nil {
		return nil, ErrGethEthClientNotSupported
	}
	return c.gethEthClient.ChainID(ctx)
}

func (c *Client) SuggestGasPrice(ctx context.Context) (*big.Int, error) {
	if c.gethEthClient == nil {
		return nil, ErrGethEthClientNotSupported
	}
	return c.gethEthClient.SuggestGasPrice(ctx)
}

func (c *Client) BalanceAt(ctx context.Context, address common.Address, blockNumber *big.Int) (*big.Int, error) {
	if c.gethEthClient == nil {
		return nil, ErrGethEthClientNotSupported
	}
	return c.gethEthClient.BalanceAt(ctx, address, blockNumber)
}

// Might fail with ErrTxTypeNotSupported for some chains, use EthGetBlockByHashWithFullTxs instead
func (c *Client) BlockByHash(ctx context.Context, hash common.Hash) (*types.Block, error) {
	if c.gethEthClient == nil {
		return nil, ErrGethEthClientNotSupported
	}
	return c.gethEthClient.BlockByHash(ctx, hash)
}

// Might fail with ErrTxTypeNotSupported for some chains, use EthGetBlockByNumberWithFullTxs instead
func (c *Client) BlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error) {
	if c.gethEthClient == nil {
		return nil, ErrGethEthClientNotSupported
	}
	return c.gethEthClient.BlockByNumber(ctx, number)
}

func (c *Client) CallContract(ctx context.Context, msg ethereum.CallMsg, blockNumber *big.Int) ([]byte, error) {
	if c.gethEthClient == nil {
		return nil, ErrGethEthClientNotSupported
	}
	return c.gethEthClient.CallContract(ctx, msg, blockNumber)
}

func (c *Client) EstimateGas(ctx context.Context, msg ethereum.CallMsg) (uint64, error) {
	if c.gethEthClient == nil {
		return 0, ErrGethEthClientNotSupported
	}
	return c.gethEthClient.EstimateGas(ctx, msg)
}

func (c *Client) FeeHistory(ctx context.Context, blockCount uint64, lastBlock *big.Int, rewardPercentiles []float64) (*ethereum.FeeHistory, error) {
	if c.gethEthClient == nil {
		return nil, ErrGethEthClientNotSupported
	}
	return c.gethEthClient.FeeHistory(ctx, blockCount, lastBlock, rewardPercentiles)
}

func (c *Client) CodeAt(ctx context.Context, address common.Address, blockNumber *big.Int) ([]byte, error) {
	if c.gethEthClient == nil {
		return nil, ErrGethEthClientNotSupported
	}
	return c.gethEthClient.CodeAt(ctx, address, blockNumber)
}

func (c *Client) StorageAt(ctx context.Context, address common.Address, key common.Hash, blockNumber *big.Int) ([]byte, error) {
	if c.gethEthClient == nil {
		return nil, ErrGethEthClientNotSupported
	}
	return c.gethEthClient.StorageAt(ctx, address, key, blockNumber)
}

// Might fail with ErrTxTypeNotSupported for some chains, use EthGetTransactionByHash instead
func (c *Client) TransactionByHash(ctx context.Context, hash common.Hash) (tx *types.Transaction, isPending bool, err error) {
	if c.gethEthClient == nil {
		return nil, false, ErrGethEthClientNotSupported
	}
	return c.gethEthClient.TransactionByHash(ctx, hash)
}

func (c *Client) TransactionCount(ctx context.Context, blockHash common.Hash) (uint, error) {
	if c.gethEthClient == nil {
		return 0, ErrGethEthClientNotSupported
	}
	return c.gethEthClient.TransactionCount(ctx, blockHash)
}

// Might fail with ErrTxTypeNotSupported for some chains, use EthGetTransactionByBlockHashAndIndex instead
func (c *Client) TransactionInBlock(ctx context.Context, blockHash common.Hash, index uint) (*types.Transaction, error) {
	if c.gethEthClient == nil {
		return nil, ErrGethEthClientNotSupported
	}
	return c.gethEthClient.TransactionInBlock(ctx, blockHash, index)
}

// Might fail with ErrTxTypeNotSupported for some chains, use EthGetTransactionReceipt instead
func (c *Client) TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error) {
	if c.gethEthClient == nil {
		return nil, ErrGethEthClientNotSupported
	}
	return c.gethEthClient.TransactionReceipt(ctx, txHash)
}

func (c *Client) NonceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (uint64, error) {
	if c.gethEthClient == nil {
		return 0, ErrGethEthClientNotSupported
	}
	return c.gethEthClient.NonceAt(ctx, account, blockNumber)
}

func (c *Client) NetworkID(ctx context.Context) (*big.Int, error) {
	if c.gethEthClient == nil {
		return nil, ErrGethEthClientNotSupported
	}
	return c.gethEthClient.NetworkID(ctx)
}

func (c *Client) PeerCount(ctx context.Context) (uint64, error) {
	if c.gethEthClient == nil {
		return 0, ErrGethEthClientNotSupported
	}
	return c.gethEthClient.PeerCount(ctx)
}

func (c *Client) SuggestGasTipCap(ctx context.Context) (*big.Int, error) {
	if c.gethEthClient == nil {
		return nil, ErrGethEthClientNotSupported
	}
	return c.gethEthClient.SuggestGasTipCap(ctx)
}

func (c *Client) BlobBaseFee(ctx context.Context) (*big.Int, error) {
	if c.gethEthClient == nil {
		return nil, ErrGethEthClientNotSupported
	}
	return c.gethEthClient.BlobBaseFee(ctx)
}

func (c *Client) SyncProgress(ctx context.Context) (*ethereum.SyncProgress, error) {
	if c.gethEthClient == nil {
		return nil, ErrGethEthClientNotSupported
	}
	return c.gethEthClient.SyncProgress(ctx)
}

func (c *Client) FilterLogs(ctx context.Context, query ethereum.FilterQuery) ([]types.Log, error) {
	if c.gethEthClient == nil {
		return nil, ErrGethEthClientNotSupported
	}
	return c.gethEthClient.FilterLogs(ctx, query)
}

func (c *Client) SubscribeNewHead(ctx context.Context, ch chan<- *types.Header) (ethereum.Subscription, error) {
	if c.gethEthClient == nil {
		return nil, ErrGethEthClientNotSupported
	}
	return c.gethEthClient.SubscribeNewHead(ctx, ch)
}

func (c *Client) SubscribeFilterLogs(ctx context.Context, q ethereum.FilterQuery, ch chan<- types.Log) (ethereum.Subscription, error) {
	if c.gethEthClient == nil {
		return nil, ErrGethEthClientNotSupported
	}
	return c.gethEthClient.SubscribeFilterLogs(ctx, q, ch)
}

func (c *Client) HeaderByHash(ctx context.Context, hash common.Hash) (*types.Header, error) {
	if c.gethEthClient == nil {
		return nil, ErrGethEthClientNotSupported
	}
	return c.gethEthClient.HeaderByHash(ctx, hash)
}

func (c *Client) HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error) {
	if c.gethEthClient == nil {
		return nil, ErrGethEthClientNotSupported
	}
	return c.gethEthClient.HeaderByNumber(ctx, number)
}

func (c *Client) PendingCodeAt(ctx context.Context, account common.Address) ([]byte, error) {
	if c.gethEthClient == nil {
		return nil, ErrGethEthClientNotSupported
	}
	return c.gethEthClient.PendingCodeAt(ctx, account)
}

func (c *Client) PendingBalanceAt(ctx context.Context, account common.Address) (*big.Int, error) {
	if c.gethEthClient == nil {
		return nil, ErrGethEthClientNotSupported
	}
	return c.gethEthClient.PendingBalanceAt(ctx, account)
}

func (c *Client) PendingStorageAt(ctx context.Context, account common.Address, key common.Hash) ([]byte, error) {
	if c.gethEthClient == nil {
		return nil, ErrGethEthClientNotSupported
	}
	return c.gethEthClient.PendingStorageAt(ctx, account, key)
}

func (c *Client) PendingTransactionCount(ctx context.Context) (uint, error) {
	if c.gethEthClient == nil {
		return 0, ErrGethEthClientNotSupported
	}
	return c.gethEthClient.PendingTransactionCount(ctx)
}

func (c *Client) PendingNonceAt(ctx context.Context, account common.Address) (uint64, error) {
	if c.gethEthClient == nil {
		return 0, ErrGethEthClientNotSupported
	}
	return c.gethEthClient.PendingNonceAt(ctx, account)
}

func (c *Client) PendingCallContract(ctx context.Context, msg ethereum.CallMsg) ([]byte, error) {
	if c.gethEthClient == nil {
		return nil, ErrGethEthClientNotSupported
	}
	return c.gethEthClient.PendingCallContract(ctx, msg)
}

// Supports only Ethereum transaction types, use EthSendRawTransaction instead for others.
func (c *Client) SendTransaction(ctx context.Context, tx *types.Transaction) error {
	if c.gethEthClient == nil {
		return ErrGethEthClientNotSupported
	}
	return c.gethEthClient.SendTransaction(ctx, tx)
}

// Supports only Ethereum transaction types, use EthGetTransactionByBlockHashAndIndex instead for others.
func (c *Client) TransactionSender(ctx context.Context, tx *types.Transaction, block common.Hash, index uint) (common.Address, error) {
	if c.gethEthClient == nil {
		return common.Address{}, ErrGethEthClientNotSupported
	}
	return c.gethEthClient.TransactionSender(ctx, tx, block, index)
}

func (c *Client) EstimateGasAtBlock(ctx context.Context, msg ethereum.CallMsg, blockNumber *big.Int) (uint64, error) {
	if c.gethEthClient == nil {
		return 0, ErrGethEthClientNotSupported
	}
	return c.gethEthClient.EstimateGasAtBlock(ctx, msg, blockNumber)
}
