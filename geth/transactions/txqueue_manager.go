package transactions

import (
	"context"
	"math/big"
	"sync"
	"time"

	ethereum "github.com/ethereum/go-ethereum"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/status-im/status-go/geth/account"
	"github.com/status-im/status-go/geth/common"
	"github.com/status-im/status-go/geth/rpc"
	"github.com/status-im/status-go/geth/transactions/queue"
)

const (
	// SendTxDefaultErrorCode is sent by default, when error is not nil, but type is unknown/unexpected.
	SendTxDefaultErrorCode = SendTransactionDefaultErrorCode
	// DefaultTxSendCompletionTimeout defines how many seconds to wait before returning result in sentTransaction().
	DefaultTxSendCompletionTimeout = 300 * time.Second

	defaultGas     = 90000
	defaultTimeout = time.Minute
)

// RPCClientProvider is an interface that provides a way
// to obtain an rpc.Client.
type RPCClientProvider interface {
	RPCClient() *rpc.Client
}

// Manager provides means to manage internal Status Backend (injected into LES)
type Manager struct {
	rpcClientProvider RPCClientProvider
	txQueue           *queue.TxQueue
	ethTxClient       EthTransactor
	notify            bool
	completionTimeout time.Duration
	rpcCallTimeout    time.Duration
	networkID         uint64

	addrLock   *AddrLocker
	localNonce sync.Map
	log        log.Logger
}

// NewManager returns a new Manager.
func NewManager(rpcClientProvider RPCClientProvider) *Manager {
	return &Manager{
		rpcClientProvider: rpcClientProvider,
		txQueue:           queue.New(),
		addrLock:          &AddrLocker{},
		notify:            true,
		completionTimeout: DefaultTxSendCompletionTimeout,
		rpcCallTimeout:    defaultTimeout,
		localNonce:        sync.Map{},
		log:               log.New("package", "status-go/geth/transactions.Manager"),
	}
}

// DisableNotificactions turns off notifications on enqueue and return of tx.
// It is not thread safe and must be called only before manager is started.
func (m *Manager) DisableNotificactions() {
	m.notify = false
}

// Start starts accepting new transactions into the queue.
func (m *Manager) Start(networkID uint64) {
	m.log.Info("start Manager")
	m.networkID = networkID
	m.ethTxClient = NewEthTxClient(m.rpcClientProvider.RPCClient())
	m.txQueue.Start()
}

// Stop stops accepting new transactions into the queue.
func (m *Manager) Stop() {
	m.log.Info("stop Manager")
	m.txQueue.Stop()
}

// TransactionQueue returns a reference to the queue.
func (m *Manager) TransactionQueue() *queue.TxQueue {
	return m.txQueue
}

// QueueTransaction puts a transaction into the queue.
func (m *Manager) QueueTransaction(tx *common.QueuedTx) error {
	if !tx.Args.Valid() {
		return common.ErrInvalidSendTxArgs
	}
	to := "<nil>"
	if tx.Args.To != nil {
		to = tx.Args.To.Hex()
	}
	m.log.Info("queue a new transaction", "id", tx.ID, "from", tx.Args.From.Hex(), "to", to)
	if err := m.txQueue.Enqueue(tx); err != nil {
		return err
	}
	if m.notify {
		NotifyOnEnqueue(tx)
	}
	return nil
}

func (m *Manager) txDone(tx *common.QueuedTx, hash gethcommon.Hash, err error) {
	if err := m.txQueue.Done(tx.ID, hash, err); err == queue.ErrQueuedTxIDNotFound {
		m.log.Warn("transaction is already removed from a queue", "ID", tx.ID)
		return
	}
	if m.notify {
		NotifyOnReturn(tx, err)
	}
}

// WaitForTransaction adds a transaction to the queue and blocks
// until it's completed, discarded or times out.
func (m *Manager) WaitForTransaction(tx *common.QueuedTx) common.TransactionResult {
	m.log.Info("wait for transaction", "id", tx.ID)
	// now wait up until transaction is:
	// - completed (via CompleteQueuedTransaction),
	// - discarded (via DiscardQueuedTransaction)
	// - or times out
	for {
		select {
		case rst := <-tx.Result:
			return rst
		case <-time.After(m.completionTimeout):
			m.txDone(tx, gethcommon.Hash{}, ErrQueuedTxTimedOut)
		}
	}
}

// NotifyErrored sends a notification for the given transaction
func (m *Manager) NotifyErrored(id common.QueuedTxID, inputError error) error {
	tx, err := m.txQueue.Get(id)
	if err != nil {
		m.log.Warn("error getting a queued transaction", "err", err)
		return err
	}

	if m.notify {
		NotifyOnReturn(tx, inputError)
	}

	return nil
}

// CompleteTransaction instructs backend to complete sending of a given transaction.
func (m *Manager) CompleteTransaction(id common.QueuedTxID, account *account.SelectedExtKey) (hash gethcommon.Hash, err error) {
	m.log.Info("complete transaction", "id", id)
	tx, err := m.txQueue.Get(id)
	if err != nil {
		m.log.Warn("error getting a queued transaction", "err", err)
		return hash, err
	}
	if err := m.txQueue.LockInprogress(id); err != nil {
		m.log.Warn("can't process transaction", "err", err)
		return hash, err
	}

	if err := m.validateAccount(tx, account); err != nil {
		m.txDone(tx, hash, err)
		return hash, err
	}
	hash, err = m.completeTransaction(account, tx)
	m.log.Info("finally completed transaction", "id", tx.ID, "hash", hash, "err", err)
	m.txDone(tx, hash, err)
	return hash, err
}

// make sure that only account which created the tx can complete it
func (m *Manager) validateAccount(tx *common.QueuedTx, selectedAccount *account.SelectedExtKey) error {
	if selectedAccount == nil {
		return account.ErrNoAccountSelected
	}

	// make sure that only account which created the tx can complete it
	if tx.Args.From.Hex() != selectedAccount.Address.Hex() {
		m.log.Warn("queued transaction does not belong to the selected account", "err", queue.ErrInvalidCompleteTxSender)
		return queue.ErrInvalidCompleteTxSender
	}

	return nil
}

func (m *Manager) completeTransaction(selectedAccount *account.SelectedExtKey, queuedTx *common.QueuedTx) (hash gethcommon.Hash, err error) {
	m.log.Info("complete transaction", "id", queuedTx.ID)
	m.addrLock.LockAddr(queuedTx.Args.From)
	var localNonce uint64
	if val, ok := m.localNonce.Load(queuedTx.Args.From); ok {
		localNonce = val.(uint64)
	}
	var nonce uint64
	defer func() {
		// nonce should be incremented only if tx completed without error
		// if upstream node returned nonce higher than ours we will stick to it
		if err == nil {
			m.localNonce.Store(queuedTx.Args.From, nonce+1)
		}
		m.addrLock.UnlockAddr(queuedTx.Args.From)

	}()
	ctx, cancel := context.WithTimeout(context.Background(), m.rpcCallTimeout)
	defer cancel()
	nonce, err = m.ethTxClient.PendingNonceAt(ctx, queuedTx.Args.From)
	if err != nil {
		return hash, err
	}
	// if upstream node returned nonce higher than ours we will use it, as it probably means
	// that another client was used for sending transactions
	if localNonce > nonce {
		nonce = localNonce
	}
	args := queuedTx.Args
	if !args.Valid() {
		return hash, common.ErrInvalidSendTxArgs
	}
	gasPrice := (*big.Int)(args.GasPrice)
	if args.GasPrice == nil {
		ctx, cancel = context.WithTimeout(context.Background(), m.rpcCallTimeout)
		defer cancel()
		gasPrice, err = m.ethTxClient.SuggestGasPrice(ctx)
		if err != nil {
			return hash, err
		}
	}

	chainID := big.NewInt(int64(m.networkID))
	value := (*big.Int)(args.Value)
	toAddr := gethcommon.Address{}
	if args.To != nil {
		toAddr = *args.To
	}

	var gas uint64
	if args.Gas == nil {
		ctx, cancel = context.WithTimeout(context.Background(), m.rpcCallTimeout)
		defer cancel()
		gas, err = m.ethTxClient.EstimateGas(ctx, ethereum.CallMsg{
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
			m.log.Info("default gas will be used. estimated gas", gas, "is lower than", defaultGas)
			gas = defaultGas
		}
	} else {
		gas = uint64(*args.Gas)
	}

	m.log.Info(
		"preparing raw transaction",
		"from", args.From.Hex(),
		"to", toAddr.Hex(),
		"gas", gas,
		"gasPrice", gasPrice,
		"value", value,
	)
	tx := types.NewTransaction(nonce, toAddr, value, gas, gasPrice, args.GetInput())
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), selectedAccount.AccountKey.PrivateKey)
	if err != nil {
		return hash, err
	}
	ctx, cancel = context.WithTimeout(context.Background(), m.rpcCallTimeout)
	defer cancel()
	if err := m.ethTxClient.SendTransaction(ctx, signedTx); err != nil {
		return hash, err
	}
	return signedTx.Hash(), nil
}

// DiscardTransaction discards a given transaction from transaction queue
func (m *Manager) DiscardTransaction(id common.QueuedTxID) error {
	tx, err := m.txQueue.Get(id)
	if err != nil {
		return err
	}
	err = m.txQueue.Done(id, gethcommon.Hash{}, ErrQueuedTxDiscarded)
	if m.notify {
		NotifyOnReturn(tx, ErrQueuedTxDiscarded)
	}
	return err
}

// DiscardTransactions discards given multiple transactions from transaction queue
func (m *Manager) DiscardTransactions(ids []common.QueuedTxID) map[common.QueuedTxID]common.RawDiscardTransactionResult {
	results := make(map[common.QueuedTxID]common.RawDiscardTransactionResult)

	for _, txID := range ids {
		err := m.DiscardTransaction(txID)
		if err != nil {
			results[txID] = common.RawDiscardTransactionResult{
				Error: err,
			}
		}
	}

	return results
}

// SendTransactionRPCHandler is a handler for eth_sendTransaction method.
// It accepts one param which is a slice with a map of transaction params.
func (m *Manager) SendTransactionRPCHandler(ctx context.Context, args ...interface{}) (interface{}, error) {
	m.log.Info("SendTransactionRPCHandler called")
	rpcCall := rpc.Call{Params: args}
	tx := common.CreateTransaction(ctx, rpcCall.ToSendTxArgs())
	if err := m.QueueTransaction(tx); err != nil {
		return nil, err
	}
	rst := m.WaitForTransaction(tx)
	if rst.Error != nil {
		return nil, rst.Error
	}
	return rst.Hash.Hex(), nil
}
