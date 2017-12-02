package common

import (
	"context"
	"errors"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

const (
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
)

// QueuedTx holds enough information to complete the queued transaction.
type QueuedTx struct {
	id         QueuedTxID
	hash       common.Hash
	ctx        context.Context
	args       SendTxArgs
	inProgress bool // true if transaction is being sent
	done       chan error
	doneOnce   sync.Once
	err        error
	sync.RWMutex
}

// NewQueuedTx QueuedTx constructor.
func NewQueuedTx(ctx context.Context, id QueuedTxID, args SendTxArgs) *QueuedTx {
	return &QueuedTx{
		id:   id,
		ctx:  ctx,
		args: args,
		done: make(chan error, 1),
	}
}

// ID gets queued transaction ID.
func (tx *QueuedTx) ID() QueuedTxID {
	tx.RLock()
	defer tx.RUnlock()

	return tx.id
}

// Hash gets queued transaction hash.
func (tx *QueuedTx) Hash() common.Hash {
	tx.RLock()
	defer tx.RUnlock()

	return tx.hash
}

// MessageID gets message ID from ctx.
func (tx *QueuedTx) MessageID() string {
	tx.RLock()
	defer tx.RUnlock()

	if tx.ctx == nil {
		return ""
	}

	if messageID, ok := tx.ctx.Value(MessageIDKey).(string); ok {
		return messageID
	}

	return ""
}

// Args gets queued transaction args.
func (tx *QueuedTx) Args() SendTxArgs {
	tx.RLock()
	defer tx.RUnlock()

	return tx.args
}

// UpdateGasPrice updates gas price if not set.
func (tx *QueuedTx) UpdateGasPrice(gasGetter func() (*hexutil.Big, error)) (*hexutil.Big, error) {
	tx.Lock()
	defer tx.Unlock()

	if tx.args.GasPrice == nil {
		value, err := gasGetter()
		if err != nil {
			return tx.args.GasPrice, err
		}

		tx.args.GasPrice = value
	}

	return tx.args.GasPrice, nil
}

// Start marks transaction as started.
func (tx *QueuedTx) Start() error {
	tx.Lock()
	defer tx.Unlock()

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
func (tx *QueuedTx) Done(hash common.Hash, err ...error) {
	tx.doneOnce.Do(func() {
		tx.hash = hash

		if len(err) != 0 {
			tx.setError(err[0])
			tx.done <- tx.err
		}
		close(tx.done)
	})
}

// Wait returns read only channel that signals either success or error transaction finish.
func (tx *QueuedTx) Wait() <-chan error {
	return tx.done
}

// Discard done transaction with discard error.
func (tx *QueuedTx) Discard() {
	tx.doneOnce.Do(func() {
		tx.setError(ErrQueuedTxDiscarded)
		tx.done <- ErrQueuedTxDiscarded
	})
}

// Error gets queued transaction error.
func (tx *QueuedTx) Error() error {
	tx.RLock()
	defer tx.RUnlock()
	return tx.err
}

func (tx *QueuedTx) setError(err error) {
	tx.Lock()
	defer tx.Unlock()
	tx.err = err
}
