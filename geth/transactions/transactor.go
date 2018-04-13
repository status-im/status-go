package transactions

import (
	"context"
	"math/big"
	"sync"
	"time"

	ethereum "github.com/ethereum/go-ethereum"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"

	"github.com/status-im/status-go/geth/account"
	"github.com/status-im/status-go/geth/rpc"
	"github.com/status-im/status-go/sign"
)

const (
	// sendTxTimeout defines how many seconds to wait before returning result in sentTransaction().
	sendTxTimeout  = 300 * time.Second
	rpcCallTimeout = time.Minute

	defaultGas = 90000
)

// Transactor validates, signs transactions.
// It uses upstream to propagate transactions to the Ethereum network.
type Transactor struct {
	pendingSignRequests  *sign.PendingRequests
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
func NewTransactor(signRequests *sign.PendingRequests) *Transactor {
	return &Transactor{
		pendingSignRequests: signRequests,
		addrLock:            &AddrLocker{},
		sendTxTimeout:       sendTxTimeout,
		rpcCallTimeout:      rpcCallTimeout,
		localNonce:          sync.Map{},
		log:                 log.New("package", "status-go/geth/transactions.Manager"),
	}
}

// SetNetworkID selects a correct network.
func (t *Transactor) SetNetworkID(networkID uint64) {
	t.networkID = networkID
}

// SetRPCClient an RPC client.
func (t *Transactor) SetRPCClient(rpcClient *rpc.Client) {
	rpcWrapper := newRPCWrapper(rpcClient)
	t.sender = rpcWrapper
	t.pendingNonceProvider = rpcWrapper
	t.gasCalculator = rpcWrapper
}

// SendTransaction is an implementation of eth_sendTransaction. It queues the tx to the sign queue.
func (t *Transactor) SendTransaction(ctx context.Context, args SendTxArgs) (gethcommon.Hash, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	completeFunc := func(acc *account.SelectedExtKey) (gethcommon.Hash, error) {
		return t.validateAndPropagate(acc, args)
	}

	request, err := t.pendingSignRequests.Add(ctx, args, completeFunc)

	if err != nil {
		return gethcommon.Hash{}, err
	}

	rst := t.pendingSignRequests.Wait(request.ID, t.sendTxTimeout)
	return rst.Hash, rst.Error
}

// make sure that only account which created the tx can complete it
func (t *Transactor) validateAccount(args SendTxArgs, selectedAccount *account.SelectedExtKey) error {
	if selectedAccount == nil {
		return account.ErrNoAccountSelected
	}

	if args.From.Hex() != selectedAccount.Address.Hex() {
		err := sign.ErrInvalidCompleteTxSender
		t.log.Error("queued transaction does not belong to the selected account", "err", err)
		return err
	}

	return nil
}

func (t *Transactor) validateAndPropagate(selectedAccount *account.SelectedExtKey, args SendTxArgs) (hash gethcommon.Hash, err error) {
	// TODO (mandrigin): Send sign request ID as a parameter to this function and uncoment the log message
	// m.log.Info("complete transaction", "id", queuedTx.ID)
	if err := t.validateAccount(args, selectedAccount); err != nil {
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
			t.log.Info("default gas will be used. estimated gas", gas, "is lower than", defaultGas)
			gas = defaultGas
		}
	} else {
		gas = uint64(*args.Gas)
	}

	var tx *types.Transaction
	if args.To != nil {
		t.log.Info("New transaction",
			"From", args.From,
			"To", *args.To,
			"Gas", gas,
			"GasPrice", gasPrice,
			"Value", value,
		)
		tx = types.NewTransaction(nonce, *args.To, value, gas, gasPrice, args.GetInput())
	} else {
		// contract creation is rare enough to log an expected address
		t.log.Info("New contract",
			"From", args.From,
			"Gas", gas,
			"GasPrice", gasPrice,
			"Value", value,
			"Contract address", crypto.CreateAddress(args.From, nonce),
		)
		tx = types.NewContractCreation(nonce, value, gas, gasPrice, args.GetInput())
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
