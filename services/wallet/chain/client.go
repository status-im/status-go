package chain

import (
	"context"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/status-im/status-go/rpc"
	"github.com/status-im/status-go/services/rpcstats"
)

var ChainClientInstances = make(map[uint64]*Client)

type Client struct {
	eth             *ethclient.Client
	ChainID         uint64
	rpcClient       *rpc.Client
	IsConnected     bool
	LastCheckedAt   int64
	IsConnectedLock sync.RWMutex
}

type FeeHistory struct {
	BaseFeePerGas []string `json:"baseFeePerGas"`
}

func NewClient(rpc *rpc.Client, chainID uint64) (*Client, error) {
	if client, ok := ChainClientInstances[chainID]; ok {
		return client, nil
	}

	ethClient, err := rpc.EthClient(chainID)
	if err != nil {
		return nil, err
	}
	client := &Client{eth: ethClient, ChainID: chainID, rpcClient: rpc, IsConnected: true, LastCheckedAt: time.Now().Unix()}
	ChainClientInstances[chainID] = client
	return client, nil
}

func NewLegacyClient(rpc *rpc.Client) (*Client, error) {
	return NewClient(rpc, rpc.UpstreamChainID)
}

func NewClients(rpc *rpc.Client, chainIDs []uint64) (res []*Client, err error) {
	for _, chainID := range chainIDs {
		client, err := NewClient(rpc, chainID)
		if err != nil {
			return nil, err
		}
		res = append(res, client)
	}
	return res, nil
}

func (cc *Client) toggleIsConnected(err error) {
	cc.IsConnectedLock.Lock()
	defer cc.IsConnectedLock.Unlock()
	cc.LastCheckedAt = time.Now().Unix()
	if err != nil {
		cc.IsConnected = false
	} else {
		cc.IsConnected = true
	}
}

func (cc *Client) ToBigInt() *big.Int {
	return big.NewInt(int64(cc.ChainID))
}

func (cc *Client) HeaderByHash(ctx context.Context, hash common.Hash) (*types.Header, error) {
	rpcstats.CountCall("eth_getBlockByHash")
	resp, err := cc.eth.HeaderByHash(ctx, hash)
	defer cc.toggleIsConnected(err)
	return resp, err
}

func (cc *Client) HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error) {
	rpcstats.CountCall("eth_getBlockByNumber")
	resp, err := cc.eth.HeaderByNumber(ctx, number)
	defer cc.toggleIsConnected(err)
	return resp, err
}

func (cc *Client) BlockByHash(ctx context.Context, hash common.Hash) (*types.Block, error) {
	rpcstats.CountCall("eth_getBlockByHash")
	resp, err := cc.eth.BlockByHash(ctx, hash)
	defer cc.toggleIsConnected(err)
	return resp, err
}

func (cc *Client) BlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error) {
	rpcstats.CountCall("eth_getBlockByNumber")
	resp, err := cc.eth.BlockByNumber(ctx, number)
	defer cc.toggleIsConnected(err)
	return resp, err
}

func (cc *Client) BalanceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (*big.Int, error) {
	rpcstats.CountCall("eth_getBalance")
	resp, err := cc.eth.BalanceAt(ctx, account, blockNumber)
	defer cc.toggleIsConnected(err)
	return resp, err
}

func (cc *Client) NonceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (uint64, error) {
	rpcstats.CountCall("eth_getTransactionCount")
	resp, err := cc.eth.NonceAt(ctx, account, blockNumber)
	defer cc.toggleIsConnected(err)
	return resp, err
}

func (cc *Client) TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error) {
	rpcstats.CountCall("eth_getTransactionReceipt")
	resp, err := cc.eth.TransactionReceipt(ctx, txHash)
	defer cc.toggleIsConnected(err)
	return resp, err
}

func (cc *Client) TransactionByHash(ctx context.Context, hash common.Hash) (tx *types.Transaction, isPending bool, err error) {
	rpcstats.CountCall("eth_getTransactionByHash")
	tx, isPending, err = cc.eth.TransactionByHash(ctx, hash)
	defer cc.toggleIsConnected(err)
	return tx, isPending, err
}

func (cc *Client) FilterLogs(ctx context.Context, q ethereum.FilterQuery) ([]types.Log, error) {
	rpcstats.CountCall("eth_getLogs")
	resp, err := cc.eth.FilterLogs(ctx, q)
	defer cc.toggleIsConnected(err)
	return resp, err
}

func (cc *Client) CodeAt(ctx context.Context, contract common.Address, blockNumber *big.Int) ([]byte, error) {
	rpcstats.CountCall("eth_getCode")
	resp, err := cc.eth.CodeAt(ctx, contract, blockNumber)
	defer cc.toggleIsConnected(err)
	return resp, err
}

func (cc *Client) CallContract(ctx context.Context, call ethereum.CallMsg, blockNumber *big.Int) ([]byte, error) {
	rpcstats.CountCall("eth_call")
	resp, err := cc.eth.CallContract(ctx, call, blockNumber)
	defer cc.toggleIsConnected(err)
	return resp, err
}

func (cc *Client) GetBaseFeeFromBlock(blockNumber *big.Int) (string, error) {
	var feeHistory FeeHistory
	err := cc.rpcClient.Call(&feeHistory, cc.ChainID, "eth_feeHistory", "0x1", (*hexutil.Big)(blockNumber), nil)
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
