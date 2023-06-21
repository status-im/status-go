package transactions

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/big"
	"time"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"

	"github.com/status-im/status-go/account"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/rpc"
)

const (
	// sendTxTimeout defines how many seconds to wait before returning result in sentTransaction().
	sendTxTimeout = 300 * time.Second

	defaultGas = 90000

	validSignatureSize = 65
)

// ErrInvalidSignatureSize is returned if a signature is not 65 bytes to avoid panic from go-ethereum
var ErrInvalidSignatureSize = errors.New("signature size must be 65")

type ErrBadNonce struct {
	nonce         uint64
	expectedNonce uint64
}

func (e *ErrBadNonce) Error() string {
	return fmt.Sprintf("bad nonce. expected %d, got %d", e.expectedNonce, e.nonce)
}

// Transactor validates, signs transactions.
// It uses upstream to propagate transactions to the Ethereum network.
type Transactor struct {
	rpcWrapper     *rpcWrapper
	sendTxTimeout  time.Duration
	rpcCallTimeout time.Duration
	networkID      uint64
	nonce          *Nonce
	log            log.Logger
}

// NewTransactor returns a new Manager.
func NewTransactor() *Transactor {
	return &Transactor{
		sendTxTimeout: sendTxTimeout,
		nonce:         NewNonce(),
		log:           log.New("package", "status-go/transactions.Manager"),
	}
}

// SetNetworkID selects a correct network.
func (t *Transactor) SetNetworkID(networkID uint64) {
	t.networkID = networkID
}

func (t *Transactor) NetworkID() uint64 {
	return t.networkID
}

// SetRPC sets RPC params, a client and a timeout
func (t *Transactor) SetRPC(rpcClient *rpc.Client, timeout time.Duration) {
	t.rpcWrapper = newRPCWrapper(rpcClient, rpcClient.UpstreamChainID)
	t.rpcCallTimeout = timeout
}

func (t *Transactor) NextNonce(rpcClient *rpc.Client, chainID uint64, from types.Address) (uint64, func(inc bool, n uint64), error) {
	wrapper := newRPCWrapper(rpcClient, chainID)
	return t.nonce.Next(wrapper, from)
}

// SendTransaction is an implementation of eth_sendTransaction. It queues the tx to the sign queue.
func (t *Transactor) SendTransaction(sendArgs SendTxArgs, verifiedAccount *account.SelectedExtKey) (hash types.Hash, err error) {
	hash, err = t.validateAndPropagate(t.rpcWrapper, verifiedAccount, sendArgs)
	return
}

func (t *Transactor) SendTransactionWithChainID(chainID uint64, sendArgs SendTxArgs, verifiedAccount *account.SelectedExtKey) (hash types.Hash, err error) {
	wrapper := newRPCWrapper(t.rpcWrapper.RPCClient, chainID)
	hash, err = t.validateAndPropagate(wrapper, verifiedAccount, sendArgs)
	return
}

// SendTransactionWithSignature receive a transaction and a signature, serialize them together and propage it to the network.
// It's different from eth_sendRawTransaction because it receives a signature and not a serialized transaction with signature.
// Since the transactions is already signed, we assume it was validated and used the right nonce.
func (t *Transactor) SendTransactionWithSignature(args SendTxArgs, sig []byte) (hash types.Hash, err error) {
	if !args.Valid() {
		return hash, ErrInvalidSendTxArgs
	}

	if len(sig) != validSignatureSize {
		return hash, ErrInvalidSignatureSize
	}

	chainID := big.NewInt(int64(t.networkID))
	signer := gethtypes.NewLondonSigner(chainID)

	tx := t.buildTransaction(args)
	expectedNonce, unlock, err := t.nonce.Next(t.rpcWrapper, args.From)
	if err != nil {
		return hash, err
	}
	defer func() {
		unlock(err == nil, expectedNonce)
	}()

	if tx.Nonce() != expectedNonce {
		return hash, &ErrBadNonce{tx.Nonce(), expectedNonce}
	}

	signedTx, err := tx.WithSignature(signer, sig)
	if err != nil {
		return hash, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), t.rpcCallTimeout)
	defer cancel()

	if err := t.rpcWrapper.SendTransaction(ctx, signedTx); err != nil {
		return hash, err
	}
	return types.Hash(signedTx.Hash()), nil
}

func (t *Transactor) HashTransaction(args SendTxArgs) (validatedArgs SendTxArgs, hash types.Hash, err error) {
	if !args.Valid() {
		return validatedArgs, hash, ErrInvalidSendTxArgs
	}

	validatedArgs = args

	nonce, unlock, err := t.nonce.Next(t.rpcWrapper, args.From)
	if err != nil {
		return validatedArgs, hash, err
	}
	defer unlock(false, 0)

	gasPrice := (*big.Int)(args.GasPrice)
	gasFeeCap := (*big.Int)(args.MaxFeePerGas)
	gasTipCap := (*big.Int)(args.MaxPriorityFeePerGas)
	if args.GasPrice == nil && args.MaxFeePerGas == nil {
		ctx, cancel := context.WithTimeout(context.Background(), t.rpcCallTimeout)
		defer cancel()
		gasPrice, err = t.rpcWrapper.SuggestGasPrice(ctx)
		if err != nil {
			return validatedArgs, hash, err
		}
	}

	chainID := big.NewInt(int64(t.networkID))
	value := (*big.Int)(args.Value)

	var gas uint64
	if args.Gas == nil {
		ctx, cancel := context.WithTimeout(context.Background(), t.rpcCallTimeout)
		defer cancel()

		var (
			gethTo    common.Address
			gethToPtr *common.Address
		)
		if args.To != nil {
			gethTo = common.Address(*args.To)
			gethToPtr = &gethTo
		}
		if args.GasPrice == nil {
			gas, err = t.rpcWrapper.EstimateGas(ctx, ethereum.CallMsg{
				From:      common.Address(args.From),
				To:        gethToPtr,
				GasFeeCap: gasFeeCap,
				GasTipCap: gasTipCap,
				Value:     value,
				Data:      args.GetInput(),
			})
		} else {
			gas, err = t.rpcWrapper.EstimateGas(ctx, ethereum.CallMsg{
				From:     common.Address(args.From),
				To:       gethToPtr,
				GasPrice: gasPrice,
				Value:    value,
				Data:     args.GetInput(),
			})
		}
		if err != nil {
			return validatedArgs, hash, err
		}
		if gas < defaultGas {
			t.log.Info("default gas will be used because estimated is lower", "estimated", gas, "default", defaultGas)
			gas = defaultGas
		}
	} else {
		gas = uint64(*args.Gas)
	}

	newNonce := hexutil.Uint64(nonce)
	newGas := hexutil.Uint64(gas)
	validatedArgs.Nonce = &newNonce
	if args.GasPrice != nil {
		validatedArgs.GasPrice = (*hexutil.Big)(gasPrice)
	} else {
		validatedArgs.MaxPriorityFeePerGas = (*hexutil.Big)(gasTipCap)
		validatedArgs.MaxPriorityFeePerGas = (*hexutil.Big)(gasFeeCap)
	}
	validatedArgs.Gas = &newGas

	tx := t.buildTransaction(validatedArgs)
	hash = types.Hash(gethtypes.NewLondonSigner(chainID).Hash(tx))

	return validatedArgs, hash, nil
}

// make sure that only account which created the tx can complete it
func (t *Transactor) validateAccount(args SendTxArgs, selectedAccount *account.SelectedExtKey) error {
	if selectedAccount == nil {
		return account.ErrNoAccountSelected
	}

	if !bytes.Equal(args.From.Bytes(), selectedAccount.Address.Bytes()) {
		return ErrInvalidTxSender
	}

	return nil
}

func (t *Transactor) validateAndPropagate(rpcWrapper *rpcWrapper, selectedAccount *account.SelectedExtKey, args SendTxArgs) (hash types.Hash, err error) {
	if err = t.validateAccount(args, selectedAccount); err != nil {
		return hash, err
	}

	if !args.Valid() {
		return hash, ErrInvalidSendTxArgs
	}

	nonce, unlock, err := t.nonce.Next(rpcWrapper, args.From)
	if err != nil {
		return hash, err
	}
	if args.Nonce != nil {
		nonce = uint64(*args.Nonce)
	}
	defer func() {
		unlock(err == nil, nonce)
	}()
	ctx, cancel := context.WithTimeout(context.Background(), t.rpcCallTimeout)
	defer cancel()

	gasPrice := (*big.Int)(args.GasPrice)
	if !args.IsDynamicFeeTx() && args.GasPrice == nil {
		gasPrice, err = rpcWrapper.SuggestGasPrice(ctx)
		if err != nil {
			return hash, err
		}
	}
	chainID := big.NewInt(int64(rpcWrapper.chainID))
	value := (*big.Int)(args.Value)
	var gas uint64
	if args.Gas != nil {
		gas = uint64(*args.Gas)
	} else if args.Gas == nil && !args.IsDynamicFeeTx() {
		ctx, cancel = context.WithTimeout(context.Background(), t.rpcCallTimeout)
		defer cancel()

		var (
			gethTo    common.Address
			gethToPtr *common.Address
		)
		if args.To != nil {
			gethTo = common.Address(*args.To)
			gethToPtr = &gethTo
		}
		gas, err = rpcWrapper.EstimateGas(ctx, ethereum.CallMsg{
			From:     common.Address(args.From),
			To:       gethToPtr,
			GasPrice: gasPrice,
			Value:    value,
			Data:     args.GetInput(),
		})
		if err != nil {
			return hash, err
		}
		if gas < defaultGas {
			t.log.Info("default gas will be used because estimated is lower", "estimated", gas, "default", defaultGas)
			gas = defaultGas
		}
	}
	tx := t.buildTransactionWithOverrides(nonce, value, gas, gasPrice, args)
	signedTx, err := gethtypes.SignTx(tx, gethtypes.NewLondonSigner(chainID), selectedAccount.AccountKey.PrivateKey)
	if err != nil {
		return hash, err
	}
	// ctx, cancel = context.WithTimeout(context.Background(), t.rpcCallTimeout)
	// defer cancel()

	if err := rpcWrapper.SendTransaction(ctx, signedTx); err != nil {
		return hash, err
	}
	return types.Hash(signedTx.Hash()), nil
}

func (t *Transactor) buildTransaction(args SendTxArgs) *gethtypes.Transaction {
	nonce := uint64(*args.Nonce)
	value := (*big.Int)(args.Value)
	gas := uint64(*args.Gas)
	gasPrice := (*big.Int)(args.GasPrice)

	return t.buildTransactionWithOverrides(nonce, value, gas, gasPrice, args)
}

func (t *Transactor) buildTransactionWithOverrides(nonce uint64, value *big.Int, gas uint64, gasPrice *big.Int, args SendTxArgs) *gethtypes.Transaction {
	var tx *gethtypes.Transaction

	if args.To != nil {
		to := common.Address(*args.To)
		var txData gethtypes.TxData

		if args.IsDynamicFeeTx() {
			gasTipCap := (*big.Int)(args.MaxPriorityFeePerGas)
			gasFeeCap := (*big.Int)(args.MaxFeePerGas)

			txData = &gethtypes.DynamicFeeTx{
				Nonce:     nonce,
				Gas:       gas,
				GasTipCap: gasTipCap,
				GasFeeCap: gasFeeCap,
				To:        &to,
				Value:     value,
				Data:      args.GetInput(),
			}
		} else {
			txData = &gethtypes.LegacyTx{
				Nonce:    nonce,
				GasPrice: gasPrice,
				Gas:      gas,
				To:       &to,
				Value:    value,
				Data:     args.GetInput(),
			}
		}
		tx = gethtypes.NewTx(txData)
		t.logNewTx(args, gas, gasPrice, value)
	} else {
		if args.IsDynamicFeeTx() {
			gasTipCap := (*big.Int)(args.MaxPriorityFeePerGas)
			gasFeeCap := (*big.Int)(args.MaxFeePerGas)

			txData := &gethtypes.DynamicFeeTx{
				Nonce:     nonce,
				Value:     value,
				Gas:       gas,
				GasTipCap: gasTipCap,
				GasFeeCap: gasFeeCap,
				Data:      args.GetInput(),
			}
			tx = gethtypes.NewTx(txData)
		} else {
			tx = gethtypes.NewContractCreation(nonce, value, gas, gasPrice, args.GetInput())
		}
		t.logNewContract(args, gas, gasPrice, value, nonce)
	}

	return tx
}

func (t *Transactor) logNewTx(args SendTxArgs, gas uint64, gasPrice *big.Int, value *big.Int) {
	t.log.Info("New transaction",
		"From", args.From,
		"To", *args.To,
		"Gas", gas,
		"GasPrice", gasPrice,
		"Value", value,
	)
}

func (t *Transactor) logNewContract(args SendTxArgs, gas uint64, gasPrice *big.Int, value *big.Int, nonce uint64) {
	t.log.Info("New contract",
		"From", args.From,
		"Gas", gas,
		"GasPrice", gasPrice,
		"Value", value,
		"Contract address", crypto.CreateAddress(args.From, nonce),
	)
}
