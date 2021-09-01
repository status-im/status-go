package wallet

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/status-im/status-go/services/rpcstats"
)

type chainClient struct {
	eth *ethclient.Client
}

func (cc *chainClient) HeaderByHash(ctx context.Context, hash common.Hash) (*types.Header, error) {
	rpcstats.CountCall("eth_getBlockByHash")
	return cc.eth.HeaderByHash(ctx, hash)
}

func (cc *chainClient) HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error) {
	rpcstats.CountCall("eth_getBlockByNumber")
	return cc.eth.HeaderByNumber(ctx, number)
}

func (cc *chainClient) BlockByHash(ctx context.Context, hash common.Hash) (*types.Block, error) {
	rpcstats.CountCall("eth_getBlockByHash")
	return cc.eth.BlockByHash(ctx, hash)
}

func (cc *chainClient) BlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error) {
	rpcstats.CountCall("eth_getBlockByNumber")
	return cc.eth.BlockByNumber(ctx, number)
}

func (cc *chainClient) BalanceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (*big.Int, error) {
	rpcstats.CountCall("eth_getBalance")
	return cc.eth.BalanceAt(ctx, account, blockNumber)
}

func (cc *chainClient) NonceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (uint64, error) {
	rpcstats.CountCall("eth_getTransactionCount")
	return cc.eth.NonceAt(ctx, account, blockNumber)
}

func (cc *chainClient) TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error) {
	rpcstats.CountCall("eth_getTransactionReceipt")
	return cc.eth.TransactionReceipt(ctx, txHash)
}

func (cc *chainClient) TransactionByHash(ctx context.Context, hash common.Hash) (tx *types.Transaction, isPending bool, err error) {
	rpcstats.CountCall("eth_getTransactionByHash")
	return cc.eth.TransactionByHash(ctx, hash)
}

func (cc *chainClient) FilterLogs(ctx context.Context, q ethereum.FilterQuery) ([]types.Log, error) {
	rpcstats.CountCall("eth_getLogs")
	return cc.eth.FilterLogs(ctx, q)
}

func (cc *chainClient) CodeAt(ctx context.Context, contract common.Address, blockNumber *big.Int) ([]byte, error) {
	rpcstats.CountCall("eth_getCode")
	return cc.eth.CodeAt(ctx, contract, blockNumber)
}

func (cc *chainClient) CallContract(ctx context.Context, call ethereum.CallMsg, blockNumber *big.Int) ([]byte, error) {
	rpcstats.CountCall("eth_call")
	return cc.eth.CallContract(ctx, call, blockNumber)
}
