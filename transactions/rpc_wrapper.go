package transactions

import (
	"context"
	"math/big"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	gethtypes "github.com/ethereum/go-ethereum/core/types"

	"github.com/status-im/status-go/crypto/types"
	"github.com/status-im/status-go/rpc"
	"github.com/status-im/status-go/rpc/chain/ethclient"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
)

// rpcWrapper wraps provides convenient interface for ethereum RPC APIs we need for sending transactions
type rpcWrapper struct {
	ethClientGetter rpc.EthClientGetter
	chainID         uint64
}

func newRPCWrapper(ethClientGetter rpc.EthClientGetter, chainID uint64) *rpcWrapper {
	return &rpcWrapper{ethClientGetter: ethClientGetter, chainID: chainID}
}

func (w *rpcWrapper) EthClient() (ethclient.EthClientInterface, error) {
	return w.ethClientGetter.EthClient(w.chainID)
}

// PendingNonceAt returns the account nonce of the given account in the pending state.
// This is the nonce that should be used for the next transaction.
func (w *rpcWrapper) PendingNonceAt(ctx context.Context, account common.Address) (uint64, error) {
	var result hexutil.Uint64
	ethClient, err := w.EthClient()
	if err != nil {
		return 0, err
	}
	err = ethClient.CallContext(ctx, &result, "eth_getTransactionCount", account, "pending")
	return uint64(result), err
}

// SuggestGasPrice retrieves the currently suggested gas price to allow a timely
// execution of a transaction.
func (w *rpcWrapper) SuggestGasPrice(ctx context.Context) (*big.Int, error) {
	var hex hexutil.Big
	ethClient, err := w.EthClient()
	if err != nil {
		return nil, err
	}
	err = ethClient.CallContext(ctx, &hex, "eth_gasPrice")
	if err != nil {
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
	ethClient, err := w.EthClient()
	if err != nil {
		return 0, err
	}
	method := "eth_estimateGas"
	if w.chainID == walletCommon.StatusNetworkSepolia {
		method = "linea_estimateGas"
		var result struct {
			GasLimit hexutil.Uint64 `json:"gasLimit"`
		}

		err := ethClient.CallContext(ctx, &result, method, walletCommon.ToCallArg(msg))
		if err != nil {
			return 0, err
		}
		return uint64(result.GasLimit), nil
	}

	err = ethClient.CallContext(ctx, &hex, method, walletCommon.ToCallArg(msg))
	if err != nil {
		return 0, err
	}
	return uint64(hex), nil
}

// Does the `eth_sendRawTransaction` call with the given raw transaction hex string.
func (w *rpcWrapper) SendRawTransaction(ctx context.Context, rawTx string) error {
	ethClient, err := w.EthClient()
	if err != nil {
		return err
	}
	return ethClient.CallContext(ctx, nil, "eth_sendRawTransaction", rawTx)
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
