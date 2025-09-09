package ethclient

import (
	"context"
	"encoding/json"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
)

// EthBlockNumber returns the number of most recent block
func (c *Client) EthBlockNumber(ctx context.Context) (uint64, error) {
	var result hexutil.Uint64
	err := c.rpcClient.CallContext(ctx, &result, "eth_blockNumber")
	return uint64(result), err
}

// EthCall executes a new message call immediately without creating a transaction on the block chain
func (c *Client) EthCall(ctx context.Context, msg ethereum.CallMsg, blockNumber *big.Int) ([]byte, error) {
	var result hexutil.Bytes
	arg := toCallArg(msg)
	blockArg := toBlockNumArg(blockNumber)
	err := c.rpcClient.CallContext(ctx, &result, "eth_call", arg, blockArg)
	return []byte(result), err
}

// EthChainId returns the chain ID of the current network
func (c *Client) EthChainId(ctx context.Context) (*big.Int, error) {
	var result hexutil.Big
	err := c.rpcClient.CallContext(ctx, &result, "eth_chainId")
	return (*big.Int)(&result), err
}

// EthEstimateGas generates and returns an estimate of how much gas is necessary to allow the transaction to complete
func (c *Client) EthEstimateGas(ctx context.Context, msg ethereum.CallMsg) (uint64, error) {
	var result hexutil.Uint64
	arg := toCallArg(msg)
	err := c.rpcClient.CallContext(ctx, &result, "eth_estimateGas", arg)
	return uint64(result), err
}

// EthFeeHistory retrieves the fee market history.
func (c *Client) EthFeeHistory(ctx context.Context, blockCount uint64, lastBlock *big.Int, rewardPercentiles []float64) (*ethereum.FeeHistory, error) {
	var res feeHistoryJSON
	if err := c.rpcClient.CallContext(ctx, &res, "eth_feeHistory", hexutil.Uint(blockCount), toBlockNumArg(lastBlock), rewardPercentiles); err != nil {
		return nil, err
	}
	reward := make([][]*big.Int, len(res.Reward))
	for i, r := range res.Reward {
		reward[i] = make([]*big.Int, len(r))
		for j, r := range r {
			reward[i][j] = (*big.Int)(r)
		}
	}
	baseFee := make([]*big.Int, len(res.BaseFee))
	for i, b := range res.BaseFee {
		baseFee[i] = (*big.Int)(b)
	}
	return &ethereum.FeeHistory{
		OldestBlock:  (*big.Int)(res.OldestBlock),
		Reward:       reward,
		BaseFee:      baseFee,
		GasUsedRatio: res.GasUsedRatio,
	}, nil
}

// EthGasPrice returns the current price per gas in wei
func (c *Client) EthGasPrice(ctx context.Context) (*big.Int, error) {
	var result hexutil.Big
	err := c.rpcClient.CallContext(ctx, &result, "eth_gasPrice")
	return (*big.Int)(&result), err
}

// EthGetBalance returns the balance of the account of given address at the given block number
func (c *Client) EthGetBalance(ctx context.Context, address common.Address, blockNumber *big.Int) (*big.Int, error) {
	var result hexutil.Big
	blockArg := toBlockNumArg(blockNumber)
	err := c.rpcClient.CallContext(ctx, &result, "eth_getBalance", address, blockArg)
	return (*big.Int)(&result), err
}

// EthGetBlockByHashWithTxHashes returns information about a block by hash with a list of transaction hashes
func (c *Client) EthGetBlockByHashWithTxHashes(ctx context.Context, hash common.Hash) (*BlockWithTxHashes, error) {
	var result *BlockWithTxHashes
	err := c.rpcClient.CallContext(ctx, &result, "eth_getBlockByHash", hash, false)
	return result, err
}

// EthGetBlockByNumberWithTxHashes returns information about a block by block number with a list of transaction hashes
func (c *Client) EthGetBlockByNumberWithTxHashes(ctx context.Context, number *big.Int) (*BlockWithTxHashes, error) {
	var result *BlockWithTxHashes
	blockArg := toBlockNumArg(number)
	err := c.rpcClient.CallContext(ctx, &result, "eth_getBlockByNumber", blockArg, false)
	return result, err
}

// EthGetBlockByHashWithFullTxs returns information about a block by hash with a list of full transactions
func (c *Client) EthGetBlockByHashWithFullTxs(ctx context.Context, hash common.Hash) (*BlockWithFullTxs, error) {
	var result *BlockWithFullTxs
	err := c.rpcClient.CallContext(ctx, &result, "eth_getBlockByHash", hash, true)
	return result, err
}

// EthGetBlockByNumberWithFullTxs returns information about a block by block number with a list of full transactions
func (c *Client) EthGetBlockByNumberWithFullTxs(ctx context.Context, number *big.Int) (*BlockWithFullTxs, error) {
	var result *BlockWithFullTxs
	blockArg := toBlockNumArg(number)
	err := c.rpcClient.CallContext(ctx, &result, "eth_getBlockByNumber", blockArg, true)
	return result, err
}

// EthGetBlockReceipts returns the receipts of all transactions in a block
func (c *Client) EthGetBlockReceipts(ctx context.Context, blockNumber *big.Int) ([]*Receipt, error) {
	var result []*Receipt
	blockArg := toBlockNumArg(blockNumber)
	err := c.rpcClient.CallContext(ctx, &result, "eth_getBlockReceipts", blockArg)
	return result, err
}

// EthGetBlockTransactionCountByHash returns the number of transactions in a block from a block matching the given block hash
func (c *Client) EthGetBlockTransactionCountByHash(ctx context.Context, hash common.Hash) (uint, error) {
	var result hexutil.Uint64
	err := c.rpcClient.CallContext(ctx, &result, "eth_getBlockTransactionCountByHash", hash)
	return uint(result), err
}

// EthGetBlockTransactionCountByNumber returns the number of transactions in a block from a block matching the given block number
func (c *Client) EthGetBlockTransactionCountByNumber(ctx context.Context, number *big.Int) (uint, error) {
	var result hexutil.Uint64
	blockArg := toBlockNumArg(number)
	err := c.rpcClient.CallContext(ctx, &result, "eth_getBlockTransactionCountByNumber", blockArg)
	return uint(result), err
}

// EthGetCode returns code at a given address
func (c *Client) EthGetCode(ctx context.Context, address common.Address, blockNumber *big.Int) ([]byte, error) {
	var result hexutil.Bytes
	blockArg := toBlockNumArg(blockNumber)
	err := c.rpcClient.CallContext(ctx, &result, "eth_getCode", address, blockArg)
	return []byte(result), err
}

// EthGetLogs returns an array of all logs matching a given filter object
func (c *Client) EthGetLogs(ctx context.Context, q ethereum.FilterQuery) ([]types.Log, error) {
	var result []types.Log
	arg, err := toFilterArg(q)
	if err != nil {
		return nil, err
	}
	err = c.rpcClient.CallContext(ctx, &result, "eth_getLogs", arg)
	return result, err
}

// EthGetProof returns the account proof for a given account
func (c *Client) EthGetProof(ctx context.Context, address common.Address, storageKeys []common.Hash, blockNumber *big.Int) (*ProofResult, error) {
	var result ProofResult
	blockArg := toBlockNumArg(blockNumber)
	err := c.rpcClient.CallContext(ctx, &result, "eth_getProof", address, storageKeys, blockArg)
	return &result, err
}

// EthGetStorageAt returns the value from a storage position at a given address
func (c *Client) EthGetStorageAt(ctx context.Context, address common.Address, key common.Hash, blockNumber *big.Int) ([]byte, error) {
	var result hexutil.Bytes
	blockArg := toBlockNumArg(blockNumber)
	err := c.rpcClient.CallContext(ctx, &result, "eth_getStorageAt", address, key, blockArg)
	return []byte(result), err
}

// EthGetTransactionByBlockHashAndIndex returns information about a transaction by block hash and transaction index position
func (c *Client) EthGetTransactionByBlockHashAndIndex(ctx context.Context, blockHash common.Hash, index uint) (*Transaction, error) {
	var result *Transaction
	err := c.rpcClient.CallContext(ctx, &result, "eth_getTransactionByBlockHashAndIndex", blockHash, hexutil.Uint(index))
	return result, err
}

// EthGetTransactionByBlockNumberAndIndex returns information about a transaction by block number and transaction index position
func (c *Client) EthGetTransactionByBlockNumberAndIndex(ctx context.Context, blockNumber *big.Int, index uint) (*Transaction, error) {
	var result *Transaction
	blockArg := toBlockNumArg(blockNumber)
	err := c.rpcClient.CallContext(ctx, &result, "eth_getTransactionByBlockNumberAndIndex", blockArg, hexutil.Uint(index))
	return result, err
}

// EthGetTransactionByHash returns the information about a transaction requested by transaction hash
func (c *Client) EthGetTransactionByHash(ctx context.Context, hash common.Hash) (*Transaction, error) {
	var result *Transaction
	err := c.rpcClient.CallContext(ctx, &result, "eth_getTransactionByHash", hash)
	return result, err
}

// EthGetTransactionCount returns the number of transactions sent from an address
func (c *Client) EthGetTransactionCount(ctx context.Context, address common.Address, blockNumber *big.Int) (uint64, error) {
	var result hexutil.Uint64
	blockArg := toBlockNumArg(blockNumber)
	err := c.rpcClient.CallContext(ctx, &result, "eth_getTransactionCount", address, blockArg)
	return uint64(result), err
}

// EthGetTransactionReceipt returns the receipt of a transaction by transaction hash
func (c *Client) EthGetTransactionReceipt(ctx context.Context, hash common.Hash) (*Receipt, error) {
	var result *Receipt
	err := c.rpcClient.CallContext(ctx, &result, "eth_getTransactionReceipt", hash)
	return result, err
}

// EthGetUncleByBlockHashAndIndex returns information about a uncle of a block by hash and uncle index position
func (c *Client) EthGetUncleByBlockHashAndIndex(ctx context.Context, blockHash common.Hash, index uint) (*BlockWithoutTxs, error) {
	var result *BlockWithoutTxs
	err := c.rpcClient.CallContext(ctx, &result, "eth_getUncleByBlockHashAndIndex", blockHash, hexutil.Uint(index))
	return result, err
}

// EthGetUncleByBlockNumberAndIndex returns information about a uncle of a block by number and uncle index position
func (c *Client) EthGetUncleByBlockNumberAndIndex(ctx context.Context, blockNumber *big.Int, index uint) (*BlockWithoutTxs, error) {
	var result *BlockWithoutTxs
	blockArg := toBlockNumArg(blockNumber)
	err := c.rpcClient.CallContext(ctx, &result, "eth_getUncleByBlockNumberAndIndex", blockArg, hexutil.Uint(index))
	return result, err
}

// EthGetUncleCountByBlockHash returns the number of uncles in a block from a block matching the given block hash
func (c *Client) EthGetUncleCountByBlockHash(ctx context.Context, blockHash common.Hash) (*big.Int, error) {
	var result hexutil.Big
	err := c.rpcClient.CallContext(ctx, &result, "eth_getUncleCountByBlockHash", blockHash)
	return (*big.Int)(&result), err
}

// EthGetUncleCountByBlockNumber returns the number of uncles in a block from a block matching the given block number
func (c *Client) EthGetUncleCountByBlockNumber(ctx context.Context, blockNumber *big.Int) (*big.Int, error) {
	var result hexutil.Big
	blockArg := toBlockNumArg(blockNumber)
	err := c.rpcClient.CallContext(ctx, &result, "eth_getUncleCountByBlockNumber", blockArg)
	return (*big.Int)(&result), err
}

// EthGetWork returns the hash of the current block, the seedHash, and the boundary condition to be met
func (c *Client) EthGetWork(ctx context.Context) (WorkData, error) {
	var result WorkData
	err := c.rpcClient.CallContext(ctx, &result, "eth_getWork")
	return result, err
}

// EthHashrate returns the number of hashes per second that the node is mining with
func (c *Client) EthHashrate(ctx context.Context) (uint64, error) {
	var result hexutil.Uint64
	err := c.rpcClient.CallContext(ctx, &result, "eth_hashrate")
	return uint64(result), err
}

// EthMaxPriorityFeePerGas returns the suggested max priority fee per gas for the next transaction
func (c *Client) EthMaxPriorityFeePerGas(ctx context.Context) (*big.Int, error) {
	var result hexutil.Big
	err := c.rpcClient.CallContext(ctx, &result, "eth_maxPriorityFeePerGas")
	return (*big.Int)(&result), err
}

// EthBlobBaseFee retrieves the current blob base fee.
func (c *Client) EthBlobBaseFee(ctx context.Context) (*big.Int, error) {
	var result hexutil.Big
	err := c.rpcClient.CallContext(ctx, &result, "eth_blobBaseFee")
	return (*big.Int)(&result), err
}

// EthMining returns true if client is actively mining new blocks
func (c *Client) EthMining(ctx context.Context) (bool, error) {
	var result bool
	err := c.rpcClient.CallContext(ctx, &result, "eth_mining")
	return result, err
}

// EthProtocolVersion returns the current ethereum protocol version
func (c *Client) EthProtocolVersion(ctx context.Context) (string, error) {
	var result string
	err := c.rpcClient.CallContext(ctx, &result, "eth_protocolVersion")
	return result, err
}

// EthSendRawTransaction creates new message call transaction or a contract creation for signed transactions
func (c *Client) EthSendRawTransaction(ctx context.Context, encodedTx []byte) (common.Hash, error) {
	var result common.Hash
	err := c.rpcClient.CallContext(ctx, &result, "eth_sendRawTransaction", hexutil.Bytes(encodedTx))
	return result, err
}

// SyncProgress returns the current sync progress
func (c *Client) EthSyncing(ctx context.Context) (*ethereum.SyncProgress, error) {
	var raw json.RawMessage
	if err := c.rpcClient.CallContext(ctx, &raw, "eth_syncing"); err != nil {
		return nil, err
	}
	// Handle the possible response types
	var syncing bool
	if err := json.Unmarshal(raw, &syncing); err == nil {
		return nil, nil // Not syncing (always false)
	}
	var p *rpcProgress
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, err
	}
	return p.toSyncProgress(), nil
}

// EthSendTransaction creates new message call transaction or a contract creation
func (c *Client) EthSendTransaction(ctx context.Context, tx *Transaction) (common.Hash, error) {
	var result common.Hash
	err := c.rpcClient.CallContext(ctx, &result, "eth_sendTransaction", tx)
	return result, err
}

// EthSign signs data with a given address
func (c *Client) EthSign(ctx context.Context, address common.Address, data []byte) ([]byte, error) {
	var result hexutil.Bytes
	err := c.rpcClient.CallContext(ctx, &result, "eth_sign", address, hexutil.Bytes(data))
	return []byte(result), err
}

// EthSignTransaction signs a transaction that can be submitted to the network at a later time using with eth_sendRawTransaction
func (c *Client) EthSignTransaction(ctx context.Context, tx *Transaction) ([]byte, error) {
	var result hexutil.Bytes
	err := c.rpcClient.CallContext(ctx, &result, "eth_signTransaction", tx)
	return []byte(result), err
}

// EthSubmitHashrate submits the mining hashrate
func (c *Client) EthSubmitHashrate(ctx context.Context, hashrate uint64, id common.Hash) (bool, error) {
	var result bool
	err := c.rpcClient.CallContext(ctx, &result, "eth_submitHashrate", hexutil.Uint64(hashrate), id)
	return result, err
}

// EthSubmitWork submits a proof-of-work solution
func (c *Client) EthSubmitWork(ctx context.Context, nonce types.BlockNonce, powHash, mixDigest common.Hash) (bool, error) {
	var result bool
	err := c.rpcClient.CallContext(ctx, &result, "eth_submitWork", nonce, powHash, mixDigest)
	return result, err
}

// EthCoinbase returns the client coinbase address
func (c *Client) EthCoinbase(ctx context.Context) (common.Address, error) {
	var result common.Address
	err := c.rpcClient.CallContext(ctx, &result, "eth_coinbase")
	return result, err
}
