package common

import (
	"context"
	"errors"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

const (
	// fixme(@jekamas): does it used anywhere
	// MessageIDKey is a key for message ID
	// This ID is required to track from which chat a given send transaction request is coming.
	MessageIDKey = contextKey("message_id")
)

type contextKey string // in order to make sure that our ctx key does not collide with keys from other packages

var (
	//ErrQueuedTxInProgress - error transaction in progress
	ErrQueuedTxInProgress = errors.New("transaction is in progress")
	//ErrQueuedTxDiscarded - error transaction discarded
	ErrQueuedTxDiscarded = errors.New("transaction has been discarded")
	//ErrQueuedTxAlreadyProcessed - error transaction has already processed
	ErrQueuedTxAlreadyProcessed = errors.New("transaction has been already processed")
)

// QueuedTx holds enough information to complete the queued transaction.
type QueuedTx struct {
	id         QueuedTxID
	hash       common.Hash
	msgID      string
	args       SendTxArgs
	inProgress bool
	done       chan error
	err        error
	sync.RWMutex
}

// NewQueuedTx QueuedTx constructor.
func NewQueuedTx(ctx context.Context, id QueuedTxID, args SendTxArgs) *QueuedTx {
	return &QueuedTx{
		id:    id,
		msgID: getMessageID(ctx),
		args:  args,
		done:  make(chan error, 1),
	}
}

// ID gets queued transaction ID.
func (tx *QueuedTx) ID() QueuedTxID {
	tx.RLock()
	defer tx.RUnlock()

	return tx.id
}

// MessageID gets queued transaction message ID.
func (tx *QueuedTx) MessageID() string {
	tx.RLock()
	defer tx.RUnlock()

	return tx.msgID
}

// Hash gets queued transaction hash.
func (tx *QueuedTx) Hash() common.Hash {
	tx.RLock()
	defer tx.RUnlock()

	return tx.hash
}

// Args gets queued transaction args.
func (tx *QueuedTx) Args() SendTxArgs {
	tx.RLock()
	defer tx.RUnlock()

	return tx.args
}

// UpdateGasPrice updates gas price if not set.
func (tx *QueuedTx) UpdateGasPrice(gasGetter func() (*hexutil.Big, error)) (*hexutil.Big, error) {
	var gasPrice hexutil.Big

	tx.RLock()
	if tx.args.GasPrice != nil {
		gasPrice = *tx.args.GasPrice
		tx.RUnlock()
		return &gasPrice, nil
	}
	tx.RUnlock()

	value, err := gasGetter()
	if err != nil {
		return &gasPrice, err
	}

	tx.Lock()
	tx.args.GasPrice = value
	tx.Unlock()

	return value, nil
}

// Start marks transaction as started.
func (tx *QueuedTx) Start() error {
	tx.Lock()
	defer tx.Unlock()

	if tx.hash != (common.Hash{}) || tx.err != nil {
		return ErrQueuedTxAlreadyProcessed
	}

	if tx.inProgress {
		return ErrQueuedTxInProgress
	}

	tx.inProgress = true

	return nil
}

// Stop marks transaction as stopped.
func (tx *QueuedTx) Stop() {
	tx.Lock()
	defer tx.Unlock()

	tx.inProgress = false
}

// Done transaction with success or error if given.
func (tx *QueuedTx) Done(hash common.Hash, err error) {
	tx.Lock()
	defer tx.Unlock()

	tx.hash = hash

	if err != nil {
		tx.setError(err)
	}
	tx.done <- err
}

// Discard done transaction with discard error.
func (tx *QueuedTx) Discard() {
	tx.Lock()
	defer tx.Unlock()

	tx.setError(ErrQueuedTxDiscarded)
	tx.done <- ErrQueuedTxDiscarded
}

// Wait returns read only channel that signals either success or error transaction finish.
func (tx *QueuedTx) Wait() <-chan error {
	return tx.done
}

// Error gets queued transaction error.
func (tx *QueuedTx) Error() error {
	tx.RLock()
	defer tx.RUnlock()
	return tx.err
}

// setError sets transaction error.
func (tx *QueuedTx) setError(err error) {
	tx.err = err
}

// getMessageID gets message ID from ctx.
func getMessageID(ctx context.Context) (msgID string) {
	if ctx == nil {
		return
	}

	var ok bool
	if msgID, ok = ctx.Value(MessageIDKey).(string); ok {
		return
	}

	return
}
