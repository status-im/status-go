package ethapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/les/status"
	"github.com/ethereum/go-ethereum/log"
	"golang.org/x/net/context"
)

// StatusBackend exposes Ethereum internals to support custom semantics in status-go bindings
type StatusBackend struct {
	eapi  *PublicEthereumAPI        // Wrapper around the Ethereum object to access metadata
	bcapi *PublicBlockChainAPI      // Wrapper around the blockchain to access chain data
	txapi *PublicTransactionPoolAPI // Wrapper around the transaction pool to access transaction data

	am *status.AccountManager
}

var (
	ErrStatusBackendNotInited = errors.New("StatusIM backend is not properly inited")
)

// NewStatusBackend creates a new backend using an existing Ethereum object.
func NewStatusBackend(apiBackend Backend) *StatusBackend {
	log.Info("StatusIM: backend service inited")
	return &StatusBackend{
		eapi:  NewPublicEthereumAPI(apiBackend),
		bcapi: NewPublicBlockChainAPI(apiBackend),
		txapi: NewPublicTransactionPoolAPI(apiBackend, new(AddrLocker)),
		am:    status.NewAccountManager(apiBackend.AccountManager()),
	}
}

// SetAccountsFilterHandler sets a callback that is triggered when account list is requested
func (b *StatusBackend) SetAccountsFilterHandler(fn status.AccountsFilterHandler) {
	b.am.SetAccountsFilterHandler(fn)
}

// AccountManager returns reference to account manager
func (b *StatusBackend) AccountManager() *status.AccountManager {
	return b.am
}

// SendTransaction wraps call to PublicTransactionPoolAPI.SendTransaction
func (b *StatusBackend) SendTransaction(ctx context.Context, rawArgs []byte, passphrase string) (common.Hash, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	var args SendTxArgs
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		return common.Hash{}, fmt.Errorf("failed to unmarshal rawArgs: %s", err)
	}

	if estimatedGas, err := b.EstimateGas(ctx, args); err == nil {
		if estimatedGas.ToInt().Cmp(big.NewInt(defaultGas)) == 1 { // gas > defaultGas
			args.Gas = estimatedGas
		}
	}

	return b.txapi.SendTransaction(ctx, args, passphrase)
}

// EstimateGas uses underlying blockchain API to obtain gas for a given tx arguments
func (b *StatusBackend) EstimateGas(ctx context.Context, args SendTxArgs) (*hexutil.Big, error) {
	if args.Gas != nil {
		return args.Gas, nil
	}

	var gasPrice hexutil.Big
	if args.GasPrice != nil {
		gasPrice = *args.GasPrice
	}

	var value hexutil.Big
	if args.Value != nil {
		value = *args.Value
	}

	callArgs := CallArgs{
		From:     args.From,
		To:       args.To,
		GasPrice: gasPrice,
		Value:    value,
		Data:     args.Data,
	}

	return b.bcapi.EstimateGas(ctx, callArgs)
}
