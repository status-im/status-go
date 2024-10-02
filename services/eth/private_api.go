//go:build enable_private_api

package eth

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	geth_rpc "github.com/ethereum/go-ethereum/rpc"

	"github.com/status-im/status-go/rpc"
)

func privateAPIs(client *rpc.Client) (apis []geth_rpc.API) {
	return []geth_rpc.API{
		{
			Namespace: "ethclient",
			Version:   "1.0",
			Service:   NewPrivateAPI(client),
			Public:    true,
		},
	}
}

type PrivateAPI struct {
	client *rpc.Client
}

func NewPrivateAPI(client *rpc.Client) *PrivateAPI {
	return &PrivateAPI{client: client}
}

type blockResponse struct {
	Header       *types.Header      `json:"header"`
	Transactions types.Transactions `json:"transactions"`
	Withdrawals  types.Withdrawals  `json:"withdrawals"`
}

func newBlockResponse(b *types.Block) *blockResponse {
	return &blockResponse{
		Header:       b.Header(),
		Transactions: b.Transactions(),
		Withdrawals:  b.Withdrawals(),
	}
}

func (pa *PrivateAPI) BlockByHash(ctx context.Context, chainId uint64, hash common.Hash) (*blockResponse, error) {
	client, err := pa.client.EthClient(chainId)
	if err != nil {
		return nil, err
	}

	block, err := client.BlockByHash(ctx, hash)
	if err != nil {
		return nil, err
	}

	return newBlockResponse(block), nil
}

func (pa *PrivateAPI) BlockByNumber(ctx context.Context, chainId uint64, number *hexutil.Big) (*blockResponse, error) {
	client, err := pa.client.EthClient(chainId)
	if err != nil {
		return nil, err
	}

	block, err := client.BlockByNumber(ctx, (*big.Int)(number))
	if err != nil {
		return nil, err
	}

	return newBlockResponse(block), nil
}

func (pa *PrivateAPI) HeaderByHash(ctx context.Context, chainId uint64, hash common.Hash) (*types.Header, error) {
	client, err := pa.client.EthClient(chainId)
	if err != nil {
		return nil, err
	}

	return client.HeaderByHash(ctx, hash)
}

func (pa *PrivateAPI) HeaderByNumber(ctx context.Context, chainId uint64, number *hexutil.Big) (*types.Header, error) {
	client, err := pa.client.EthClient(chainId)
	if err != nil {
		return nil, err
	}

	return client.HeaderByNumber(ctx, (*big.Int)(number))
}

type transactionByHashResponse struct {
	Tx        *types.Transaction `json:"tx"`
	IsPending bool               `json:"isPending"`
}

func (pa *PrivateAPI) TransactionByHash(ctx context.Context, chainId uint64, txHash common.Hash) (*transactionByHashResponse, error) {

	client, err := pa.client.EthClient(chainId)
	if err != nil {
		return nil, err
	}

	tx, isPending, err := client.TransactionByHash(ctx, txHash)
	if err != nil {
		return nil, err
	}

	ret := &transactionByHashResponse{
		Tx:        tx,
		IsPending: isPending,
	}

	return ret, nil
}

func (pa *PrivateAPI) TransactionReceipt(ctx context.Context, chainId uint64, txHash common.Hash) (*types.Receipt, error) {
	client, err := pa.client.EthClient(chainId)
	if err != nil {
		return nil, err
	}

	return client.TransactionReceipt(ctx, txHash)
}

func (pa *PrivateAPI) SuggestGasPrice(ctx context.Context, chainId uint64) (*hexutil.Big, error) {
	client, err := pa.client.EthClient(chainId)
	if err != nil {
		return nil, err
	}

	ret, err := client.SuggestGasPrice(ctx)
	if err != nil {
		return nil, err
	}

	return (*hexutil.Big)(ret), nil
}

func (pa *PrivateAPI) BlockNumber(ctx context.Context, chainId uint64) (uint64, error) {
	client, err := pa.client.EthClient(chainId)
	if err != nil {
		return 0, err
	}

	return client.BlockNumber(ctx)
}
