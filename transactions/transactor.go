package transactions

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"

	ethereum "github.com/ethereum/go-ethereum"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"

	"github.com/status-im/status-go/account"
	"github.com/status-im/status-go/rpc"
)

const (
	// sendTxTimeout defines how many seconds to wait before returning result in sentTransaction().
	sendTxTimeout = 300 * time.Second

	defaultGas = 90000
)

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
	sender               ethereum.TransactionSender
	pendingNonceProvider PendingNonceProvider
	gasCalculator        GasCalculator
	sendTxTimeout        time.Duration
	rpcCallTimeout       time.Duration
	networkID            uint64

	addrLock   *AddrLocker
	localNonce sync.Map
	log        log.Logger
}

// NewTransactor returns a new Manager.
func NewTransactor() *Transactor {
	return &Transactor{
		addrLock:      &AddrLocker{},
		sendTxTimeout: sendTxTimeout,
		localNonce:    sync.Map{},
		log:           log.New("package", "status-go/transactions.Manager"),
	}
}

// SetNetworkID selects a correct network.
func (t *Transactor) SetNetworkID(networkID uint64) {
	t.networkID = networkID
}

// SetRPC sets RPC params, a client and a timeout
func (t *Transactor) SetRPC(rpcClient *rpc.Client, timeout time.Duration) {
	rpcWrapper := newRPCWrapper(rpcClient)
	t.sender = rpcWrapper
	t.pendingNonceProvider = rpcWrapper
	t.gasCalculator = rpcWrapper
	t.rpcCallTimeout = timeout
}

// SendTransaction is an implementation of eth_sendTransaction. It queues the tx to the sign queue.
func (t *Transactor) SendTransaction(sendArgs SendTxArgs, verifiedAccount *account.SelectedExtKey) (hash gethcommon.Hash, err error) {
	hash, err = t.validateAndPropagate(verifiedAccount, sendArgs)
	return
}

// SendTransactionWithSignature receive a transaction and a signature, serialize them together and propage it to the network.
// It's different from eth_sendRawTransaction because it receives a signature and not a serialized transaction with signature.
// Since the transactions is already signed, we assume it was validated and used the right nonce.
func (t *Transactor) SendTransactionWithSignature(args SendTxArgs, sig []byte) (hash gethcommon.Hash, err error) {
	if !args.Valid() {
		return hash, ErrInvalidSendTxArgs
	}

	if len(sig) != 65 {
		return hash, ErrInvalidSignatureSize
	}

	chainID := big.NewInt(int64(t.networkID))
	signer := types.NewEIP155Signer(chainID)

	tx := t.buildTransaction(args)
	t.addrLock.LockAddr(args.From)
	defer func() {
		// nonce should be incremented only if tx completed without error
		// and if no other transactions have been sent while signing the current one.
		if err == nil {
			t.localNonce.Store(args.From, uint64(*args.Nonce)+1)
		}
		t.addrLock.UnlockAddr(args.From)
	}()

	expectedNonce, err := t.getTransactionNonce(args)
	if err != nil {
		return hash, err
	}

	if tx.Nonce() != expectedNonce {
		return hash, &ErrBadNonce{tx.Nonce(), expectedNonce}
	}

	signedTx, err := tx.WithSignature(signer, sig)
	if err != nil {
		return hash, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), t.rpcCallTimeout)
	defer cancel()

	if err := t.sender.SendTransaction(ctx, signedTx); err != nil {
		return hash, err
	}

	return signedTx.Hash(), nil
}

func (t *Transactor) HashTransaction(args SendTxArgs) (validatedArgs SendTxArgs, hash gethcommon.Hash, err error) {
	if !args.Valid() {
		return validatedArgs, hash, ErrInvalidSendTxArgs
	}

	validatedArgs = args

	t.addrLock.LockAddr(args.From)
	defer func() {
		t.addrLock.UnlockAddr(args.From)
	}()

	nonce, err := t.getTransactionNonce(validatedArgs)
	if err != nil {
		return validatedArgs, hash, err
	}

	gasPrice := (*big.Int)(args.GasPrice)
	if args.GasPrice == nil {
		ctx, cancel := context.WithTimeout(context.Background(), t.rpcCallTimeout)
		defer cancel()
		gasPrice, err = t.gasCalculator.SuggestGasPrice(ctx)
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
		gas, err = t.gasCalculator.EstimateGas(ctx, ethereum.CallMsg{
			From:     args.From,
			To:       args.To,
			GasPrice: gasPrice,
			Value:    value,
			Data:     args.GetInput(),
		})
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
	validatedArgs.GasPrice = (*hexutil.Big)(gasPrice)
	validatedArgs.Gas = &newGas

	tx := t.buildTransaction(validatedArgs)
	hash = types.NewEIP155Signer(chainID).Hash(tx)

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

func (t *Transactor) validateAndPropagate(selectedAccount *account.SelectedExtKey, args SendTxArgs) (hash gethcommon.Hash, err error) {
	if err = t.validateAccount(args, selectedAccount); err != nil {
		return hash, err
	}

	if !args.Valid() {
		return hash, ErrInvalidSendTxArgs
	}
	t.addrLock.LockAddr(args.From)
	var localNonce uint64
	if val, ok := t.localNonce.Load(args.From); ok {
		localNonce = val.(uint64)
	}
	var nonce uint64
	defer func() {
		// nonce should be incremented only if tx completed without error
		// if upstream node returned nonce higher than ours we will stick to it
		if err == nil {
			t.localNonce.Store(args.From, nonce+1)
		}
		t.addrLock.UnlockAddr(args.From)

	}()
	ctx, cancel := context.WithTimeout(context.Background(), t.rpcCallTimeout)
	defer cancel()
	nonce, err = t.pendingNonceProvider.PendingNonceAt(ctx, args.From)
	if err != nil {
		return hash, err
	}
	// if upstream node returned nonce higher than ours we will use it, as it probably means
	// that another client was used for sending transactions
	if localNonce > nonce {
		nonce = localNonce
	}
	gasPrice := (*big.Int)(args.GasPrice)
	if args.GasPrice == nil {
		ctx, cancel = context.WithTimeout(context.Background(), t.rpcCallTimeout)
		defer cancel()
		gasPrice, err = t.gasCalculator.SuggestGasPrice(ctx)
		if err != nil {
			return hash, err
		}
	}

	chainID := big.NewInt(int64(t.networkID))
	value := (*big.Int)(args.Value)

	var gas uint64
	if args.Gas == nil {
		ctx, cancel = context.WithTimeout(context.Background(), t.rpcCallTimeout)
		defer cancel()
		gas, err = t.gasCalculator.EstimateGas(ctx, ethereum.CallMsg{
			From:     args.From,
			To:       args.To,
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
	} else {
		gas = uint64(*args.Gas)
	}

	var tx *types.Transaction
	if args.To != nil {
		tx = types.NewTransaction(nonce, *args.To, value, gas, gasPrice, args.GetInput())
		t.logNewTx(args, gas, gasPrice, value)
	} else {
		tx = types.NewContractCreation(nonce, value, gas, gasPrice, args.GetInput())
		t.logNewContract(args, gas, gasPrice, value, nonce)
	}

	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), selectedAccount.AccountKey.PrivateKey)
	if err != nil {
		return hash, err
	}
	ctx, cancel = context.WithTimeout(context.Background(), t.rpcCallTimeout)
	defer cancel()

	if err := t.sender.SendTransaction(ctx, signedTx); err != nil {
		return hash, err
	}
	return signedTx.Hash(), nil
}

func (t *Transactor) buildTransaction(args SendTxArgs) *types.Transaction {
	nonce := uint64(*args.Nonce)
	value := (*big.Int)(args.Value)
	gas := uint64(*args.Gas)
	gasPrice := (*big.Int)(args.GasPrice)

	var tx *types.Transaction

	if args.To != nil {
		tx = types.NewTransaction(nonce, *args.To, value, gas, gasPrice, args.GetInput())
		t.logNewTx(args, gas, gasPrice, value)
	} else {
		tx = types.NewContractCreation(nonce, value, gas, gasPrice, args.GetInput())
		t.logNewContract(args, gas, gasPrice, value, nonce)
	}

	return tx
}

func (t *Transactor) getTransactionNonce(args SendTxArgs) (newNonce uint64, err error) {
	var (
		localNonce  uint64
		remoteNonce uint64
	)

	// get the local nonce
	if val, ok := t.localNonce.Load(args.From); ok {
		localNonce = val.(uint64)
	}

	// get the remote nonce
	ctx, cancel := context.WithTimeout(context.Background(), t.rpcCallTimeout)
	defer cancel()
	remoteNonce, err = t.pendingNonceProvider.PendingNonceAt(ctx, args.From)
	if err != nil {
		return newNonce, err
	}

	// if upstream node returned nonce higher than ours we will use it, as it probably means
	// that another client was used for sending transactions
	if remoteNonce > localNonce {
		newNonce = remoteNonce
	} else {
		newNonce = localNonce
	}

	return newNonce, nil
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
