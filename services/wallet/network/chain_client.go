package network

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/status-im/status-go/services/rpcstats"
)

type ChainClient struct {
	eth     *ethclient.Client
	ChainID uint64
}

func (cc *ChainClient) ToBigInt() *big.Int {
	return big.NewInt(int64(cc.ChainID))
}

func (cc *ChainClient) HeaderByHash(ctx context.Context, hash common.Hash) (*types.Header, error) {
	rpcstats.CountCall("eth_getBlockByHash")
	return cc.eth.HeaderByHash(ctx, hash)
}

func (cc *ChainClient) HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error) {
	rpcstats.CountCall("eth_getBlockByNumber")
	return cc.eth.HeaderByNumber(ctx, number)
}

func (cc *ChainClient) BlockByHash(ctx context.Context, hash common.Hash) (*types.Block, error) {
	rpcstats.CountCall("eth_getBlockByHash")
	return cc.eth.BlockByHash(ctx, hash)
}

func (cc *ChainClient) BlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error) {
	rpcstats.CountCall("eth_getBlockByNumber")
	return cc.eth.BlockByNumber(ctx, number)
}

func (cc *ChainClient) BalanceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (*big.Int, error) {
	rpcstats.CountCall("eth_getBalance")
	return cc.eth.BalanceAt(ctx, account, blockNumber)
}

func (cc *ChainClient) NonceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (uint64, error) {
	rpcstats.CountCall("eth_getTransactionCount")
	return cc.eth.NonceAt(ctx, account, blockNumber)
}

func (cc *ChainClient) TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error) {
	rpcstats.CountCall("eth_getTransactionReceipt")
	return cc.eth.TransactionReceipt(ctx, txHash)
}

func (cc *ChainClient) TransactionByHash(ctx context.Context, hash common.Hash) (tx *types.Transaction, isPending bool, err error) {
	rpcstats.CountCall("eth_getTransactionByHash")
	return cc.eth.TransactionByHash(ctx, hash)
}

func (cc *ChainClient) FilterLogs(ctx context.Context, q ethereum.FilterQuery) ([]types.Log, error) {
	rpcstats.CountCall("eth_getLogs")
	return cc.eth.FilterLogs(ctx, q)
}

func (cc *ChainClient) CodeAt(ctx context.Context, contract common.Address, blockNumber *big.Int) ([]byte, error) {
	rpcstats.CountCall("eth_getCode")
	return cc.eth.CodeAt(ctx, contract, blockNumber)
}

func (cc *ChainClient) CallContract(ctx context.Context, call ethereum.CallMsg, blockNumber *big.Int) ([]byte, error) {
	rpcstats.CountCall("eth_call")
	return cc.eth.CallContract(ctx, call, blockNumber)
}
