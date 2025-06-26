package transactions

import (
	"context"
	"math/big"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	gethtypes "github.com/ethereum/go-ethereum/core/types"

	"github.com/status-im/status-go/eth-node/types"
	walletCommon "github.com/status-im/status-go/services/wallet/common"

	"github.com/status-im/status-go/rpc"
)

// rpcWrapper wraps provides convenient interface for ethereum RPC APIs we need for sending transactions
type rpcWrapper struct {
	RPCClient rpc.ClientInterface
	chainID   uint64
}

func newRPCWrapper(client rpc.ClientInterface, chainID uint64) *rpcWrapper {
	return &rpcWrapper{RPCClient: client, chainID: chainID}
}

// PendingNonceAt returns the account nonce of the given account in the pending state.
// This is the nonce that should be used for the next transaction.
func (w *rpcWrapper) PendingNonceAt(ctx context.Context, account common.Address) (uint64, error) {
	var result hexutil.Uint64
	err := w.RPCClient.CallContext(ctx, &result, w.chainID, "eth_getTransactionCount", account, "pending")
	return uint64(result), err
}

// SuggestGasPrice retrieves the currently suggested gas price to allow a timely
// execution of a transaction.
func (w *rpcWrapper) SuggestGasPrice(ctx context.Context) (*big.Int, error) {
	var hex hexutil.Big
	if err := w.RPCClient.CallContext(ctx, &hex, w.chainID, "eth_gasPrice"); err != nil {
		return nil, err
	}
	return (*big.Int)(&hex), nil
}

// EstimateGas tries to estimate the gas needed to execute a specific transaction based on
// the current pending state of the backend blockchain. There is no guarantee that this is
// the true gas limit requirement as other transactions may be added or removed by miners,
// but it should provide a basis for setting a reasonable default.
func (w *rpcWrapper) EstimateGas(ctx context.Context, msg ethereum.CallMsg) (uint64, error) {
	var hex hexutil.Uint64
	method := "eth_estimateGas"
	if w.chainID == walletCommon.StatusNetworkSepolia {
		method = "linea_estimateGas"
		var result struct {
			GasLimit hexutil.Uint64 `json:"gasLimit"`
		}

		err := w.RPCClient.CallContext(ctx, &result, w.chainID, method, walletCommon.ToCallArg(msg))
		if err != nil {
			return 0, err
		}
		return uint64(result.GasLimit), nil
	}

	err := w.RPCClient.CallContext(ctx, &hex, w.chainID, method, walletCommon.ToCallArg(msg))
	if err != nil {
		return 0, err
	}
	return uint64(hex), nil
}

// Does the `eth_sendRawTransaction` call with the given raw transaction hex string.
func (w *rpcWrapper) SendRawTransaction(ctx context.Context, rawTx string) error {
	return w.RPCClient.CallContext(ctx, nil, w.chainID, "eth_sendRawTransaction", rawTx)
}

// SendTransaction injects a signed transaction into the pending pool for execution.
//
// If the transaction was a contract creation use the TransactionReceipt method to get the
// contract address after the transaction has been mined.
func (w *rpcWrapper) SendTransaction(ctx context.Context, tx *gethtypes.Transaction) error {
	data, err := tx.MarshalBinary()
	if err != nil {
		return err
	}
	return w.SendRawTransaction(ctx, types.EncodeHex(data))
}
