package transactions

//go:generate go tool mockgen -package=mock_transactions -source=transactor.go -destination=mock/transactor.go

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/big"
	"time"

	"go.uber.org/zap"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"

	accsmanagement "github.com/status-im/status-go/accounts-management"
	"github.com/status-im/status-go/accounts-management/generator"
	gocommon "github.com/status-im/status-go/common"
	"github.com/status-im/status-go/internal/crypto"
	types2 "github.com/status-im/status-go/internal/crypto/types"
	"github.com/status-im/status-go/internal/logutils"
	"github.com/status-im/status-go/rpc"
	"github.com/status-im/status-go/services/wallet/bigint"
	wallet_common "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/pendingtxtracker"
	"github.com/status-im/status-go/services/wallet/wallettypes"
)

const (
	// sendTxTimeout defines how many seconds to wait before returning result in sentTransaction().
	sendTxTimeout = 300 * time.Second

	defaultGas = 90000

	ValidSignatureSize = 65
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

// Transactor is an interface that defines the methods for validating and sending transactions.
type TransactorIface interface {
	NextNonce(ctx context.Context, ethClientGetter rpc.EthClientGetter, chainID uint64, from types2.Address) (uint64, error)
	EstimateGas(chainID uint64, from common.Address, to common.Address, value *big.Int, input []byte) (uint64, error)
	SendTransactionWithChainID(chainID uint64, sendArgs wallettypes.SendTxArgs, lastUsedNonce int64, verifiedAccount *generator.Account) (hash types2.Hash, nonce uint64, err error)
	ValidateAndBuildTransaction(chainID uint64, sendArgs wallettypes.SendTxArgs, lastUsedNonce int64) (tx *gethtypes.Transaction, nonce uint64, err error)
	AddSignatureToTransaction(chainID uint64, tx *gethtypes.Transaction, sig []byte) (*gethtypes.Transaction, error)
	SendRawTransaction(chainID uint64, rawTx string) error
	BuildTransactionWithSignature(chainID uint64, args wallettypes.SendTxArgs, sig []byte) (*gethtypes.Transaction, error)
	SendTransactionWithSignature(txArgs *wallettypes.SendTxArgs, txWithSignature *gethtypes.Transaction) (hash types2.Hash, err error)
	StoreAndTrackPendingTx(txArgs *wallettypes.SendTxArgs, txWithSignature *gethtypes.Transaction) error
}

// Transactor validates, signs transactions.
// It uses upstream to propagate transactions to the Ethereum network.
type Transactor struct {
	ethClientGetter rpc.EthClientGetter
	pendingTracker  *pendingtxtracker.PendingTxTracker
	sendTxTimeout   time.Duration
	rpcCallTimeout  time.Duration
	logger          *zap.Logger
}

// NewTransactor returns a new Manager.
func NewTransactor() *Transactor {
	return &Transactor{
		sendTxTimeout: sendTxTimeout,
		logger:        logutils.ZapLogger().Named("transactor"),
	}
}

// SetPendingTracker sets a pending tracker.
func (t *Transactor) SetPendingTracker(tracker *pendingtxtracker.PendingTxTracker) {
	t.pendingTracker = tracker
}

func (t *Transactor) SetEthClientGetter(ethClientGetter rpc.EthClientGetter, rpcCallTimeout time.Duration) {
	t.ethClientGetter = ethClientGetter
	t.rpcCallTimeout = rpcCallTimeout
}

func (t *Transactor) NextNonce(ctx context.Context, ethClientGetter rpc.EthClientGetter, chainID uint64, from types2.Address) (uint64, error) {
	wrapper := newRPCWrapper(ethClientGetter, chainID)
	return wrapper.PendingNonceAt(ctx, common.Address(from))
}

func (t *Transactor) EstimateGas(chainID uint64, from common.Address, to common.Address, value *big.Int, input []byte) (uint64, error) {
	rpcWrapper := newRPCWrapper(t.ethClientGetter, chainID)

	ctx, cancel := context.WithTimeout(context.Background(), t.rpcCallTimeout)
	defer cancel()

	msg := ethereum.CallMsg{
		From:  from,
		To:    &to,
		Value: value,
		Data:  input,
	}

	return rpcWrapper.EstimateGas(ctx, msg)
}

func (t *Transactor) SendTransactionWithChainID(chainID uint64, sendArgs wallettypes.SendTxArgs, lastUsedNonce int64, verifiedAccount *generator.Account) (hash types2.Hash, nonce uint64, err error) {
	wrapper := newRPCWrapper(t.ethClientGetter, chainID)
	hash, nonce, err = t.validateAndPropagate(wrapper, verifiedAccount, sendArgs, lastUsedNonce)
	return
}

func (t *Transactor) ValidateAndBuildTransaction(chainID uint64, sendArgs wallettypes.SendTxArgs, lastUsedNonce int64) (tx *gethtypes.Transaction, nonce uint64, err error) {
	wrapper := newRPCWrapper(t.ethClientGetter, chainID)
	tx, err = t.validateAndBuildTransaction(wrapper, sendArgs, lastUsedNonce)
	if err != nil {
		return nil, 0, err
	}

	return tx, tx.Nonce(), err
}

func (t *Transactor) AddSignatureToTransaction(chainID uint64, tx *gethtypes.Transaction, sig []byte) (*gethtypes.Transaction, error) {
	if len(sig) != ValidSignatureSize {
		return nil, ErrInvalidSignatureSize
	}

	rpcWrapper := newRPCWrapper(t.ethClientGetter, chainID)
	chID := big.NewInt(int64(rpcWrapper.chainID))

	signer := gethtypes.NewLondonSigner(chID)
	txWithSignature, err := tx.WithSignature(signer, sig)
	if err != nil {
		return nil, err
	}

	return txWithSignature, nil
}

func (t *Transactor) SendRawTransaction(chainID uint64, rawTx string) error {
	rpcWrapper := newRPCWrapper(t.ethClientGetter, chainID)

	ctx, cancel := context.WithTimeout(context.Background(), t.rpcCallTimeout)
	defer cancel()

	return rpcWrapper.SendRawTransaction(ctx, rawTx)
}

func createPendingTransaction(txArgs *wallettypes.SendTxArgs, txWithSignature *gethtypes.Transaction) (pTx *pendingtxtracker.PendingTransaction) {
	var toAddress common.Address
	if txWithSignature.To() != nil {
		toAddress = *txWithSignature.To()
	} else {
		// when deploying a new contract we don't know the address beforehand,
		// so we create it from the address the contract was deployed from and the nonce
		toAddress = common.Address(crypto.CreateAddress(txArgs.From, txWithSignature.Nonce()))
	}
	tokenKey := ""
	if txArgs.FromToken != nil {
		tokenKey = txArgs.FromToken.Key()
	}

	pTx = &pendingtxtracker.PendingTransaction{
		Hash:       txWithSignature.Hash(),
		Timestamp:  uint64(time.Now().Unix()),
		Value:      bigint.BigInt{Int: txWithSignature.Value()},
		From:       common.Address(txArgs.From),
		To:         toAddress,
		Nonce:      txWithSignature.Nonce(),
		Data:       string(txWithSignature.Data()),
		Type:       pendingtxtracker.WalletTransfer,
		ChainID:    wallet_common.ChainID(txWithSignature.ChainId().Uint64()),
		Symbol:     tokenKey,
		AutoDelete: new(bool),
	}
	// Stop tracking as soon as the transaction is confirmed
	*pTx.AutoDelete = true
	return
}

func (t *Transactor) StoreAndTrackPendingTx(txArgs *wallettypes.SendTxArgs, txWithSignature *gethtypes.Transaction) error {
	if t.pendingTracker == nil {
		return nil
	}

	pTx := createPendingTransaction(txArgs, txWithSignature)
	return t.pendingTracker.StoreAndTrackPendingTx(pTx)
}

func (t *Transactor) sendTransaction(rpcWrapper *rpcWrapper, txArgs *wallettypes.SendTxArgs, txWithSignature *gethtypes.Transaction) (hash types2.Hash, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), t.rpcCallTimeout)
	defer cancel()

	if err := rpcWrapper.SendTransaction(ctx, txWithSignature); err != nil {
		return hash, err
	}

	err = t.StoreAndTrackPendingTx(txArgs, txWithSignature)
	if err != nil {
		return hash, err
	}

	return types2.Hash(txWithSignature.Hash()), nil
}

func (t *Transactor) SendTransactionWithSignature(txArgs *wallettypes.SendTxArgs, txWithSignature *gethtypes.Transaction) (hash types2.Hash, err error) {
	rpcWrapper := newRPCWrapper(t.ethClientGetter, txWithSignature.ChainId().Uint64())

	return t.sendTransaction(rpcWrapper, txArgs, txWithSignature)
}

// BuildTransactionAndSendWithSignature receive a transaction and a signature, serialize them together
// It's different from eth_sendRawTransaction because it receives a signature and not a serialized transaction with signature.
// Since the transactions is already signed, we assume it was validated and used the right nonce.
func (t *Transactor) BuildTransactionWithSignature(chainID uint64, args wallettypes.SendTxArgs, sig []byte) (*gethtypes.Transaction, error) {
	if !args.Valid() {
		return nil, wallettypes.ErrInvalidSendTxArgs
	}

	if len(sig) != ValidSignatureSize {
		return nil, ErrInvalidSignatureSize
	}

	ctx, cancel := context.WithTimeout(context.Background(), t.rpcCallTimeout)
	defer cancel()

	tx := t.buildTransaction(args)
	expectedNonce, err := t.NextNonce(ctx, t.ethClientGetter, chainID, args.From)
	if err != nil {
		return nil, err
	}

	if tx.Nonce() != expectedNonce {
		return nil, &ErrBadNonce{tx.Nonce(), expectedNonce}
	}

	txWithSignature, err := t.AddSignatureToTransaction(chainID, tx, sig)
	if err != nil {
		return nil, err
	}

	return txWithSignature, nil
}

// make sure that only account which created the tx can complete it
func (t *Transactor) validateAccount(args wallettypes.SendTxArgs, selectedAccount *generator.Account) error {
	if selectedAccount == nil {
		return accsmanagement.ErrNoAccountSelected
	}

	if !args.Valid() {
		return wallettypes.ErrInvalidSendTxArgs
	}

	if !bytes.Equal(args.From.Bytes(), selectedAccount.Address().Bytes()) {
		return wallettypes.ErrInvalidTxSender
	}

	return nil
}

func (t *Transactor) validateAndBuildTransaction(rpcWrapper *rpcWrapper, args wallettypes.SendTxArgs, lastUsedNonce int64) (tx *gethtypes.Transaction, err error) {
	if !args.Valid() {
		return tx, wallettypes.ErrInvalidSendTxArgs
	}

	ctx, cancel := context.WithTimeout(context.Background(), t.rpcCallTimeout)
	defer cancel()

	var nonce uint64
	if args.Nonce != nil {
		nonce = uint64(*args.Nonce)
	} else {
		// some chains, like arbitrum doesn't count pending txs in the nonce, so we need to calculate it manually
		if lastUsedNonce < 0 {
			nonce, err = t.NextNonce(ctx, rpcWrapper.ethClientGetter, rpcWrapper.chainID, args.From)
			if err != nil {
				return tx, err
			}
		} else {
			nonce = uint64(lastUsedNonce) + 1
		}
	}

	gasPrice := (*big.Int)(args.GasPrice)
	// GasPrice should be estimated only for LegacyTx
	if !args.IsDynamicFeeTx() && args.GasPrice == nil {
		gasPrice, err = rpcWrapper.SuggestGasPrice(ctx)
		if err != nil {
			return tx, err
		}
	}

	value := (*big.Int)(args.Value)
	var gas uint64
	if args.Gas != nil {
		gas = uint64(*args.Gas)
	} else {

		var (
			gethTo    common.Address
			gethToPtr *common.Address
		)
		if args.To != nil {
			gethTo = common.Address(*args.To)
			gethToPtr = &gethTo
		}
		if args.IsDynamicFeeTx() {
			gasFeeCap := (*big.Int)(args.MaxFeePerGas)
			gasTipCap := (*big.Int)(args.MaxPriorityFeePerGas)
			gas, err = rpcWrapper.EstimateGas(ctx, ethereum.CallMsg{
				From:      common.Address(args.From),
				To:        gethToPtr,
				GasFeeCap: gasFeeCap,
				GasTipCap: gasTipCap,
				Value:     value,
				Data:      args.GetInput(),
			})
		} else {
			gas, err = rpcWrapper.EstimateGas(ctx, ethereum.CallMsg{
				From:     common.Address(args.From),
				To:       gethToPtr,
				GasPrice: gasPrice,
				Value:    value,
				Data:     args.GetInput(),
			})
		}
		if err != nil {
			return tx, err
		}
	}

	tx = t.buildTransactionWithOverrides(nonce, value, gas, gasPrice, args)
	return tx, nil
}

func (t *Transactor) validateAndPropagate(rpcWrapper *rpcWrapper, selectedAccount *generator.Account, args wallettypes.SendTxArgs, lastUsedNonce int64) (hash types2.Hash, nonce uint64, err error) {
	if err = t.validateAccount(args, selectedAccount); err != nil {
		return hash, nonce, err
	}

	tx, err := t.validateAndBuildTransaction(rpcWrapper, args, lastUsedNonce)
	if err != nil {
		return hash, nonce, err
	}

	chainID := big.NewInt(int64(rpcWrapper.chainID))
	signedTx, err := gethtypes.SignTx(tx, gethtypes.NewLondonSigner(chainID), selectedAccount.PrivateKey())
	if err != nil {
		return hash, nonce, err
	}

	hash, err = t.sendTransaction(rpcWrapper, &args, signedTx)
	return hash, tx.Nonce(), err
}

func (t *Transactor) buildTransaction(args wallettypes.SendTxArgs) *gethtypes.Transaction {
	var (
		nonce    uint64
		value    *big.Int
		gas      uint64
		gasPrice *big.Int
	)
	if args.Nonce != nil {
		nonce = uint64(*args.Nonce)
	}
	if args.Value != nil {
		value = (*big.Int)(args.Value)
	}
	if args.Gas != nil {
		gas = uint64(*args.Gas)
	}
	if args.GasPrice != nil {
		gasPrice = (*big.Int)(args.GasPrice)
	}

	return t.buildTransactionWithOverrides(nonce, value, gas, gasPrice, args)
}

func (t *Transactor) buildTransactionWithOverrides(nonce uint64, value *big.Int, gas uint64, gasPrice *big.Int, args wallettypes.SendTxArgs) *gethtypes.Transaction {
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

func (t *Transactor) logNewTx(args wallettypes.SendTxArgs, gas uint64, gasPrice *big.Int, value *big.Int) {
	if args.IsDynamicFeeTx() {
		t.logger.Info("New dynamic fee transaction",
			zap.String("From", gocommon.TruncateWithDot(args.From.Hex())),
			zap.String("To", gocommon.TruncateWithDot(args.To.Hex())),
			zap.Uint64("Gas", gas),
			zap.Stringer("GasTipCap", (*big.Int)(args.MaxPriorityFeePerGas)),
			zap.Stringer("GasFeeCap", (*big.Int)(args.MaxFeePerGas)),
			zap.Stringer("Value", value),
		)
	} else {
		t.logger.Info("New legacy transaction",
			zap.String("From", gocommon.TruncateWithDot(args.From.Hex())),
			zap.String("To", gocommon.TruncateWithDot(args.To.Hex())),
			zap.Uint64("Gas", gas),
			zap.Stringer("GasPrice", gasPrice),
			zap.Stringer("Value", value),
		)
	}
}

func (t *Transactor) logNewContract(args wallettypes.SendTxArgs, gas uint64, gasPrice *big.Int, value *big.Int, nonce uint64) {
	contractAddress := crypto.CreateAddress(args.From, nonce)

	t.logger.Info("New contract",
		zap.String("From", gocommon.TruncateWithDot(args.From.Hex())),
		zap.Uint64("Gas", gas),
		zap.Stringer("GasPrice", gasPrice),
		zap.Stringer("Value", value),
		zap.String("Contract address", gocommon.TruncateWithDot(contractAddress.Hex())),
	)
}
