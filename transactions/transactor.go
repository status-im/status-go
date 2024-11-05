package transactions

//go:generate mockgen -package=mock_transactor -source=transactor.go -destination=mock/transactor.go

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/big"
	"time"

	"go.uber.org/zap"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	gethtypes "github.com/ethereum/go-ethereum/core/types"

	"github.com/status-im/status-go/account"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/logutils"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/rpc"
	"github.com/status-im/status-go/services/wallet/bigint"
	wallet_common "github.com/status-im/status-go/services/wallet/common"
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
	NextNonce(rpcClient rpc.ClientInterface, chainID uint64, from types.Address) (uint64, error)
	EstimateGas(network *params.Network, from common.Address, to common.Address, value *big.Int, input []byte) (uint64, error)
	SendTransaction(sendArgs wallettypes.SendTxArgs, verifiedAccount *account.SelectedExtKey, lastUsedNonce int64) (hash types.Hash, nonce uint64, err error)
	SendTransactionWithChainID(chainID uint64, sendArgs wallettypes.SendTxArgs, lastUsedNonce int64, verifiedAccount *account.SelectedExtKey) (hash types.Hash, nonce uint64, err error)
	ValidateAndBuildTransaction(chainID uint64, sendArgs wallettypes.SendTxArgs, lastUsedNonce int64) (tx *gethtypes.Transaction, nonce uint64, err error)
	AddSignatureToTransaction(chainID uint64, tx *gethtypes.Transaction, sig []byte) (*gethtypes.Transaction, error)
	SendRawTransaction(chainID uint64, rawTx string) error
	BuildTransactionWithSignature(chainID uint64, args wallettypes.SendTxArgs, sig []byte) (*gethtypes.Transaction, error)
	SendTransactionWithSignature(from common.Address, symbol string, multiTransactionID wallet_common.MultiTransactionIDType, tx *gethtypes.Transaction) (hash types.Hash, err error)
	StoreAndTrackPendingTx(from common.Address, symbol string, chainID uint64, multiTransactionID wallet_common.MultiTransactionIDType, tx *gethtypes.Transaction) error
}

// Transactor validates, signs transactions.
// It uses upstream to propagate transactions to the Ethereum network.
type Transactor struct {
	rpcWrapper     *rpcWrapper
	pendingTracker *PendingTxTracker
	sendTxTimeout  time.Duration
	rpcCallTimeout time.Duration
	networkID      uint64
	logger         *zap.Logger
}

// NewTransactor returns a new Manager.
func NewTransactor() *Transactor {
	return &Transactor{
		sendTxTimeout: sendTxTimeout,
		logger:        logutils.ZapLogger().Named("transactor"),
	}
}

// SetPendingTracker sets a pending tracker.
func (t *Transactor) SetPendingTracker(tracker *PendingTxTracker) {
	t.pendingTracker = tracker
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

func (t *Transactor) NextNonce(rpcClient rpc.ClientInterface, chainID uint64, from types.Address) (uint64, error) {
	wrapper := newRPCWrapper(rpcClient, chainID)
	ctx := context.Background()
	nonce, err := wrapper.PendingNonceAt(ctx, common.Address(from))
	if err != nil {
		return 0, err
	}

	// We need to take into consideration all pending transactions in case of Optimism, cause the network returns always
	// the nonce of last executed tx + 1 for the next nonce value.
	if chainID == wallet_common.OptimismMainnet ||
		chainID == wallet_common.OptimismSepolia {
		if t.pendingTracker != nil {
			countOfPendingTXs, err := t.pendingTracker.CountPendingTxsFromNonce(wallet_common.ChainID(chainID), common.Address(from), nonce)
			if err != nil {
				return 0, err
			}
			return nonce + countOfPendingTXs, nil
		}
	}

	return nonce, err
}

func (t *Transactor) EstimateGas(network *params.Network, from common.Address, to common.Address, value *big.Int, input []byte) (uint64, error) {
	rpcWrapper := newRPCWrapper(t.rpcWrapper.RPCClient, network.ChainID)

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

// SendTransaction is an implementation of eth_sendTransaction. It queues the tx to the sign queue.
func (t *Transactor) SendTransaction(sendArgs wallettypes.SendTxArgs, verifiedAccount *account.SelectedExtKey, lastUsedNonce int64) (hash types.Hash, nonce uint64, err error) {
	hash, nonce, err = t.validateAndPropagate(t.rpcWrapper, verifiedAccount, sendArgs, lastUsedNonce)
	return
}

func (t *Transactor) SendTransactionWithChainID(chainID uint64, sendArgs wallettypes.SendTxArgs, lastUsedNonce int64, verifiedAccount *account.SelectedExtKey) (hash types.Hash, nonce uint64, err error) {
	wrapper := newRPCWrapper(t.rpcWrapper.RPCClient, chainID)
	hash, nonce, err = t.validateAndPropagate(wrapper, verifiedAccount, sendArgs, lastUsedNonce)
	return
}

func (t *Transactor) ValidateAndBuildTransaction(chainID uint64, sendArgs wallettypes.SendTxArgs, lastUsedNonce int64) (tx *gethtypes.Transaction, nonce uint64, err error) {
	wrapper := newRPCWrapper(t.rpcWrapper.RPCClient, chainID)
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

	rpcWrapper := newRPCWrapper(t.rpcWrapper.RPCClient, chainID)
	chID := big.NewInt(int64(rpcWrapper.chainID))

	signer := gethtypes.NewLondonSigner(chID)
	txWithSignature, err := tx.WithSignature(signer, sig)
	if err != nil {
		return nil, err
	}

	return txWithSignature, nil
}

func (t *Transactor) SendRawTransaction(chainID uint64, rawTx string) error {
	rpcWrapper := newRPCWrapper(t.rpcWrapper.RPCClient, chainID)

	ctx, cancel := context.WithTimeout(context.Background(), t.rpcCallTimeout)
	defer cancel()

	return rpcWrapper.SendRawTransaction(ctx, rawTx)
}

func createPendingTransaction(from common.Address, symbol string, chainID uint64, multiTransactionID wallet_common.MultiTransactionIDType, tx *gethtypes.Transaction) (pTx *PendingTransaction) {

	pTx = &PendingTransaction{
		Hash:               tx.Hash(),
		Timestamp:          uint64(time.Now().Unix()),
		Value:              bigint.BigInt{Int: tx.Value()},
		From:               from,
		To:                 *tx.To(),
		Nonce:              tx.Nonce(),
		Data:               string(tx.Data()),
		Type:               WalletTransfer,
		ChainID:            wallet_common.ChainID(chainID),
		MultiTransactionID: multiTransactionID,
		Symbol:             symbol,
		AutoDelete:         new(bool),
	}
	// Transaction downloader will delete pending transaction as soon as it is confirmed
	*pTx.AutoDelete = false
	return
}

func (t *Transactor) StoreAndTrackPendingTx(from common.Address, symbol string, chainID uint64, multiTransactionID wallet_common.MultiTransactionIDType, tx *gethtypes.Transaction) error {
	if t.pendingTracker == nil {
		return nil
	}

	pTx := createPendingTransaction(from, symbol, chainID, multiTransactionID, tx)
	return t.pendingTracker.StoreAndTrackPendingTx(pTx)
}

func (t *Transactor) sendTransaction(rpcWrapper *rpcWrapper, from common.Address, symbol string,
	multiTransactionID wallet_common.MultiTransactionIDType, tx *gethtypes.Transaction) (hash types.Hash, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), t.rpcCallTimeout)
	defer cancel()

	if err := rpcWrapper.SendTransaction(ctx, tx); err != nil {
		return hash, err
	}

	err = t.StoreAndTrackPendingTx(from, symbol, rpcWrapper.chainID, multiTransactionID, tx)
	if err != nil {
		return hash, err
	}

	return types.Hash(tx.Hash()), nil
}

func (t *Transactor) SendTransactionWithSignature(from common.Address, symbol string,
	multiTransactionID wallet_common.MultiTransactionIDType, tx *gethtypes.Transaction) (hash types.Hash, err error) {
	rpcWrapper := newRPCWrapper(t.rpcWrapper.RPCClient, tx.ChainId().Uint64())

	return t.sendTransaction(rpcWrapper, from, symbol, multiTransactionID, tx)
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

	tx := t.buildTransaction(args)
	expectedNonce, err := t.NextNonce(t.rpcWrapper.RPCClient, chainID, args.From)
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

func (t *Transactor) HashTransaction(args wallettypes.SendTxArgs) (validatedArgs wallettypes.SendTxArgs, hash types.Hash, err error) {
	if !args.Valid() {
		return validatedArgs, hash, wallettypes.ErrInvalidSendTxArgs
	}

	validatedArgs = args

	nonce, err := t.NextNonce(t.rpcWrapper.RPCClient, t.rpcWrapper.chainID, args.From)
	if err != nil {
		return validatedArgs, hash, err
	}

	gasPrice := (*big.Int)(args.GasPrice)
	gasFeeCap := (*big.Int)(args.MaxFeePerGas)
	gasTipCap := (*big.Int)(args.MaxPriorityFeePerGas)
	if args.GasPrice == nil && !args.IsDynamicFeeTx() {
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
		if args.IsDynamicFeeTx() {
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
	} else {
		gas = uint64(*args.Gas)
	}

	newNonce := hexutil.Uint64(nonce)
	newGas := hexutil.Uint64(gas)
	validatedArgs.Nonce = &newNonce
	if !args.IsDynamicFeeTx() {
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
func (t *Transactor) validateAccount(args wallettypes.SendTxArgs, selectedAccount *account.SelectedExtKey) error {
	if selectedAccount == nil {
		return account.ErrNoAccountSelected
	}

	if !bytes.Equal(args.From.Bytes(), selectedAccount.Address.Bytes()) {
		return wallettypes.ErrInvalidTxSender
	}

	return nil
}

func (t *Transactor) validateAndBuildTransaction(rpcWrapper *rpcWrapper, args wallettypes.SendTxArgs, lastUsedNonce int64) (tx *gethtypes.Transaction, err error) {
	if !args.Valid() {
		return tx, wallettypes.ErrInvalidSendTxArgs
	}

	var nonce uint64
	if args.Nonce != nil {
		nonce = uint64(*args.Nonce)
	} else {
		// some chains, like arbitrum doesn't count pending txs in the nonce, so we need to calculate it manually
		if lastUsedNonce < 0 {
			nonce, err = t.NextNonce(rpcWrapper.RPCClient, rpcWrapper.chainID, args.From)
			if err != nil {
				return tx, err
			}
		} else {
			nonce = uint64(lastUsedNonce) + 1
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), t.rpcCallTimeout)
	defer cancel()

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

func (t *Transactor) validateAndPropagate(rpcWrapper *rpcWrapper, selectedAccount *account.SelectedExtKey, args wallettypes.SendTxArgs, lastUsedNonce int64) (hash types.Hash, nonce uint64, err error) {
	symbol := args.Symbol
	if args.Version == wallettypes.SendTxArgsVersion1 {
		symbol = args.FromTokenID
	}

	if err = t.validateAccount(args, selectedAccount); err != nil {
		return hash, nonce, err
	}

	tx, err := t.validateAndBuildTransaction(rpcWrapper, args, lastUsedNonce)
	if err != nil {
		return hash, nonce, err
	}

	chainID := big.NewInt(int64(rpcWrapper.chainID))
	signedTx, err := gethtypes.SignTx(tx, gethtypes.NewLondonSigner(chainID), selectedAccount.AccountKey.PrivateKey)
	if err != nil {
		return hash, nonce, err
	}

	hash, err = t.sendTransaction(rpcWrapper, common.Address(args.From), symbol, args.MultiTransactionID, signedTx)
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
	t.logger.Info("New transaction",
		zap.Stringer("From", args.From),
		zap.Stringer("To", args.To),
		zap.Uint64("Gas", gas),
		zap.Stringer("GasPrice", gasPrice),
		zap.Stringer("Value", value),
	)
}

func (t *Transactor) logNewContract(args wallettypes.SendTxArgs, gas uint64, gasPrice *big.Int, value *big.Int, nonce uint64) {
	t.logger.Info("New contract",
		zap.Stringer("From", args.From),
		zap.Uint64("Gas", gas),
		zap.Stringer("GasPrice", gasPrice),
		zap.Stringer("Value", value),
		zap.Stringer("Contract address", crypto.CreateAddress(args.From, nonce)),
	)
}
