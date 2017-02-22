package ethapi

import (
	"errors"
	"time"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/les/status"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/pborman/uuid"
	"golang.org/x/net/context"
)

// StatusBackend exposes Ethereum internals to support custom semantics in status-go bindings
type StatusBackend struct {
	eapi  *PublicEthereumAPI        // Wrapper around the Ethereum object to access metadata
	bcapi *PublicBlockChainAPI      // Wrapper around the blockchain to access chain data
	txapi *PublicTransactionPoolAPI // Wrapper around the transaction pool to access transaction data

	txQueue *status.TxQueue
	am      *status.AccountManager
}

var (
	ErrStatusBackendNotInited = errors.New("StatusIM backend is not properly inited")
)

// NewStatusBackend creates a new backend using an existing Ethereum object.
func NewStatusBackend(apiBackend Backend) *StatusBackend {
	glog.V(logger.Info).Infof("StatusIM: backend service inited")
	return &StatusBackend{
		eapi:    NewPublicEthereumAPI(apiBackend),
		bcapi:   NewPublicBlockChainAPI(apiBackend),
		txapi:   NewPublicTransactionPoolAPI(apiBackend),
		txQueue: status.NewTransactionQueue(),
		am:      status.NewAccountManager(apiBackend.AccountManager()),
	}
}

func (b *StatusBackend) Start() {
	glog.V(logger.Info).Infof("StatusIM: started as LES sub-protocol")
	b.txQueue.Start()
}

func (b *StatusBackend) Stop() {
	glog.V(logger.Info).Infof("StatusIM: stopped as LES sub-protocol")
	b.txQueue.Stop()
}

func (b *StatusBackend) NotifyOnQueuedTxReturn(queuedTx *status.QueuedTx, err error) {
	if b == nil {
		return
	}

	b.txQueue.NotifyOnQueuedTxReturn(queuedTx, err)
}

func (b *StatusBackend) SetTransactionReturnHandler(fn status.EnqueuedTxReturnHandler) {
	b.txQueue.SetTxReturnHandler(fn)
}

func (b *StatusBackend) SetTransactionQueueHandler(fn status.EnqueuedTxHandler) {
	b.txQueue.SetEnqueueHandler(fn)
}

func (b *StatusBackend) TransactionQueue() *status.TxQueue {
	return b.txQueue
}

func (b *StatusBackend) SetAccountsFilterHandler(fn status.AccountsFilterHandler) {
	b.am.SetAccountsFilterHandler(fn)
}

func (b *StatusBackend) AccountManager() *status.AccountManager {
	return b.am
}

// SendTransaction wraps call to PublicTransactionPoolAPI.SendTransaction
func (b *StatusBackend) SendTransaction(ctx context.Context, args status.SendTxArgs) (common.Hash, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	queuedTx := &status.QueuedTx{
		Id:      status.QueuedTxId(uuid.New()),
		Hash:    common.Hash{},
		Context: ctx,
		Args:    status.SendTxArgs(args),
		Done:    make(chan struct{}, 1),
		Discard: make(chan struct{}, 1),
	}

	// send transaction to pending pool, w/o blocking
	b.txQueue.EnqueueAsync(queuedTx)

	// now wait up until transaction is:
	// - completed (via CompleteQueuedTransaction),
	// - discarded (via DiscardQueuedTransaction)
	// - or times out
	select {
	case <-queuedTx.Done:
		b.NotifyOnQueuedTxReturn(queuedTx, queuedTx.Err)
		return queuedTx.Hash, queuedTx.Err
	case <-queuedTx.Discard:
		b.NotifyOnQueuedTxReturn(queuedTx, status.ErrQueuedTxDiscarded)
		return queuedTx.Hash, queuedTx.Err
	case <-time.After(status.DefaultTxSendCompletionTimeout * time.Second):
		b.NotifyOnQueuedTxReturn(queuedTx, status.ErrQueuedTxTimedOut)
		return common.Hash{}, status.ErrQueuedTxTimedOut
	}

	return queuedTx.Hash, nil
}

// CompleteQueuedTransaction wraps call to PublicTransactionPoolAPI.CompleteQueuedTransaction
func (b *StatusBackend) CompleteQueuedTransaction(ctx context.Context, id status.QueuedTxId, passphrase string) (common.Hash, error) {
	queuedTx, err := b.txQueue.Get(id)
	if err != nil {
		return common.Hash{}, err
	}

	hash, err := b.txapi.CompleteQueuedTransaction(ctx, SendTxArgs(queuedTx.Args), passphrase)

	// on password error, notify the app, and keep tx in queue (so that CompleteQueuedTransaction() can be resent)
	if err == keystore.ErrDecrypt {
		b.NotifyOnQueuedTxReturn(queuedTx, err)
		return hash, err // SendTransaction is still blocked
	}

	// when incorrect sender tries to complete the account, notify and keep tx in queue (so that correct sender can complete)
	if err == status.ErrInvalidCompleteTxSender {
		b.NotifyOnQueuedTxReturn(queuedTx, err)
		return hash, err // SendTransaction is still blocked
	}

	// allow SendTransaction to return
	queuedTx.Hash = hash
	queuedTx.Err = err
	queuedTx.Done <- struct{}{} // sendTransaction() waits on this, notify so that it can return

	return hash, err
}

// DiscardQueuedTransaction discards queued transaction forcing SendTransaction to return
func (b *StatusBackend) DiscardQueuedTransaction(id status.QueuedTxId) error {
	queuedTx, err := b.txQueue.Get(id)
	if err != nil {
		return err
	}

	// remove from queue, before notifying SendTransaction
	b.TransactionQueue().Remove(queuedTx.Id)

	// allow SendTransaction to return
	queuedTx.Err = status.ErrQueuedTxDiscarded
	queuedTx.Discard <- struct{}{} // sendTransaction() waits on this, notify so that it can return

	return nil
}
