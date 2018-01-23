package transactions

import (
	"context"
	"math/big"
	"time"

	ethereum "github.com/ethereum/go-ethereum"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/status-im/status-go/geth/common"
	"github.com/status-im/status-go/geth/log"
	"github.com/status-im/status-go/geth/transactions/queue"
)

const (
	// SendTxDefaultErrorCode is sent by default, when error is not nil, but type is unknown/unexpected.
	SendTxDefaultErrorCode = SendTransactionDefaultErrorCode
	// DefaultTxSendCompletionTimeout defines how many seconds to wait before returning result in sentTransaction().
	DefaultTxSendCompletionTimeout = 300

	defaultGas     = 90000
	defaultTimeout = time.Minute
)

// Manager provides means to manage internal Status Backend (injected into LES)
type Manager struct {
	nodeManager    common.NodeManager
	accountManager common.AccountManager
	txQueue        *queue.TxQueue
	ethTxClient    EthTransactor
	addrLock       *AddrLocker
	notify         bool
}

// NewManager returns a new Manager.
func NewManager(nodeManager common.NodeManager, accountManager common.AccountManager) *Manager {
	return &Manager{
		nodeManager:    nodeManager,
		accountManager: accountManager,
		txQueue:        queue.New(),
		addrLock:       &AddrLocker{},
		notify:         true,
	}
}

// DisableNotifications turns off notifications on enqueue and return of tx.
// it is not thread safe and must be called only before manager is started.
func (m *Manager) DisableNotificactions() {
	m.notify = false
}

// Start starts accepting new transactions into the queue.
func (m *Manager) Start() {
	log.Info("start Manager")
	m.ethTxClient = NewEthTxClient(m.nodeManager.RPCClient())
	m.txQueue.Start()
}

// Stop stops accepting new transactions into the queue.
func (m *Manager) Stop() {
	log.Info("stop Manager")
	m.txQueue.Stop()
}

// TransactionQueue returns a reference to the queue.
func (m *Manager) TransactionQueue() common.TxQueue {
	return m.txQueue
}

// QueueTransaction puts a transaction into the queue.
func (m *Manager) QueueTransaction(tx *common.QueuedTx) error {
	to := "<nil>"
	if tx.Args.To != nil {
		to = tx.Args.To.Hex()
	}
	log.Info("queue a new transaction", "id", tx.ID, "from", tx.Args.From.Hex(), "to", to)
	err := m.txQueue.Enqueue(tx)
	if m.notify {
		NotifyOnEnqueue(tx)
	}
	return err
}

func (m *Manager) txDone(tx *common.QueuedTx, hash gethcommon.Hash, err error) {
	m.txQueue.Done(tx.ID, hash, err) //nolint: errcheck
	if m.notify {
		NotifyOnReturn(tx)
	}
}

// WaitForTransaction adds a transaction to the queue and blocks
// until it's completed, discarded or times out.
func (m *Manager) WaitForTransaction(tx *common.QueuedTx) error {
	log.Info("wait for transaction", "id", tx.ID)
	// now wait up until transaction is:
	// - completed (via CompleteQueuedTransaction),
	// - discarded (via DiscardQueuedTransaction)
	// - or times out
	select {
	case <-tx.Done:
	case <-time.After(DefaultTxSendCompletionTimeout * time.Second):
		m.txDone(tx, gethcommon.Hash{}, queue.ErrQueuedTxTimedOut)
	}
	return tx.Err
}

// CompleteTransaction instructs backend to complete sending of a given transaction.
// TODO(adam): investigate a possible bug that calling this method multiple times with the same Transaction ID
// results in sending multiple transactions.
func (m *Manager) CompleteTransaction(id common.QueuedTxID, password string) (hash gethcommon.Hash, err error) {
	log.Info("complete transaction", "id", id)
	tx, err := m.txQueue.LockInprogress(id)
	if err != nil {
		log.Warn("error getting a queued transaction", "err", err)
		return hash, err
	}
	account, err := m.validateAccount(tx)
	if err != nil {
		m.txDone(tx, hash, err)
		return hash, err
	}
	// Send the transaction finally.
	hash, err = m.completeTransaction(tx, account, password)
	log.Info("finally completed transaction", "id", tx.ID, "hash", hash, "err", err)
	m.txDone(tx, hash, err)
	return hash, err
}

func (m *Manager) validateAccount(tx *common.QueuedTx) (*common.SelectedExtKey, error) {
	selectedAccount, err := m.accountManager.SelectedAccount()
	if err != nil {
		log.Warn("failed to get a selected account", "err", err)
		return nil, err
	}
	// make sure that only account which created the tx can complete it
	if tx.Args.From.Hex() != selectedAccount.Address.Hex() {
		log.Warn("queued transaction does not belong to the selected account", "err", queue.ErrInvalidCompleteTxSender)
		return nil, queue.ErrInvalidCompleteTxSender
	}
	return selectedAccount, nil
}

func (m *Manager) completeTransaction(queuedTx *common.QueuedTx, selectedAccount *common.SelectedExtKey, password string) (hash gethcommon.Hash, err error) {
	log.Info("complete transaction", "id", queuedTx.ID)
	log.Info("verifying account password for transaction", "id", queuedTx.ID)
	config, err := m.nodeManager.NodeConfig()
	if err != nil {
		return hash, err
	}
	_, err = m.accountManager.VerifyAccountPassword(config.KeyStoreDir, selectedAccount.Address.String(), password)
	if err != nil {
		log.Warn("failed to verify account", "account", selectedAccount.Address.String(), "error", err.Error())
		return hash, err
	}
	m.addrLock.LockAddr(queuedTx.Args.From)
	defer m.addrLock.UnlockAddr(queuedTx.Args.From)
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()
	nonce, err := m.ethTxClient.PendingNonceAt(ctx, queuedTx.Args.From)
	if err != nil {
		return hash, err
	}
	args := queuedTx.Args
	gasPrice := (*big.Int)(args.GasPrice)
	if args.GasPrice == nil {
		ctx, cancel = context.WithTimeout(context.Background(), defaultTimeout)
		defer cancel()
		gasPrice, err = m.ethTxClient.SuggestGasPrice(ctx)
		if err != nil {
			return hash, err
		}
	}

	chainID := big.NewInt(int64(config.NetworkID))
	data := []byte(args.Data)
	value := (*big.Int)(args.Value)
	toAddr := gethcommon.Address{}
	if args.To != nil {
		toAddr = *args.To
	}

	gas := (*big.Int)(args.Gas)
	if args.Gas == nil {
		ctx, cancel = context.WithTimeout(context.Background(), defaultTimeout)
		defer cancel()
		gas, err = m.ethTxClient.EstimateGas(ctx, ethereum.CallMsg{
			From:     args.From,
			To:       args.To,
			GasPrice: gasPrice,
			Value:    value,
			Data:     data,
		})
		if err != nil {
			return hash, err
		}
		if gas.Cmp(big.NewInt(defaultGas)) == -1 {
			log.Info("default gas will be used. estimated gas", gas, "is lower than", defaultGas)
			gas = big.NewInt(defaultGas)
		}
	}

	log.Info(
		"preparing raw transaction",
		"from", args.From.Hex(),
		"to", toAddr.Hex(),
		"gas", gas,
		"gasPrice", gasPrice,
		"value", value,
	)
	tx := types.NewTransaction(nonce, toAddr, value, gas, gasPrice, data)
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), selectedAccount.AccountKey.PrivateKey)
	if err != nil {
		return hash, err
	}
	ctx, cancel = context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()
	if err := m.ethTxClient.SendTransaction(ctx, signedTx); err != nil {
		return hash, err
	}
	return signedTx.Hash(), nil
}

// CompleteTransactions instructs backend to complete sending of multiple transactions
func (m *Manager) CompleteTransactions(ids []common.QueuedTxID, password string) map[common.QueuedTxID]common.RawCompleteTransactionResult {
	results := make(map[common.QueuedTxID]common.RawCompleteTransactionResult)
	for _, txID := range ids {
		txHash, txErr := m.CompleteTransaction(txID, password)
		results[txID] = common.RawCompleteTransactionResult{
			Hash:  txHash,
			Error: txErr,
		}
	}
	return results
}

// DiscardTransaction discards a given transaction from transaction queue
func (m *Manager) DiscardTransaction(id common.QueuedTxID) error {
	tx, err := m.txQueue.Get(id)
	if err != nil {
		return err
	}
	err = m.txQueue.Done(id, gethcommon.Hash{}, queue.ErrQueuedTxDiscarded)
	if m.notify {
		NotifyOnReturn(tx)
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
	log.Info("SendTransactionRPCHandler called")

	// TODO(adam): it's a hack to parse arguments as common.RPCCall can do that.
	// We should refactor parsing these params to a separate struct.
	rpcCall := common.RPCCall{Params: args}
	tx := common.CreateTransaction(ctx, rpcCall.ToSendTxArgs())

	if err := m.QueueTransaction(tx); err != nil {
		return nil, err
	}

	if err := m.WaitForTransaction(tx); err != nil {
		return nil, err
	}

	return tx.Hash.Hex(), nil
}
