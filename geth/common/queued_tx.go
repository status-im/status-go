package common

import (
	"context"
	"sync"

	"github.com/ethereum/go-ethereum/common"
)

// QueuedTx holds enough information to complete the queued transaction.
type QueuedTx struct {
	id         QueuedTxID
	hash       common.Hash
	context    context.Context
	args       SendTxArgs
	inProgress bool // true if transaction is being sent
	done       chan struct{}
	discard    chan struct{}
	err        error
	sync.RWMutex
}

// NewQueuedTx QueuedTx constructor.
func NewQueuedTx(id QueuedTxID, ctx context.Context, args SendTxArgs) *QueuedTx {
	return &QueuedTx{
		id:      id,
		context: ctx,
		args:    args,
		done:    make(chan struct{}, 1),
		discard: make(chan struct{}, 1),
	}
}

// ID gets queued transaction ID.
func (tx *QueuedTx) ID() QueuedTxID {
	tx.RLock()
	defer tx.RUnlock()

	return tx.id
}

// SetID sets queued transaction ID.
func (tx *QueuedTx) SetID(id QueuedTxID) {
	tx.Lock()
	defer tx.Unlock()

	tx.id = id
}

// Hash gets queued transaction hash.
func (tx *QueuedTx) Hash() common.Hash {
	tx.RLock()
	defer tx.RUnlock()

	return tx.hash
}

// SetHash sets queued transaction hash.
func (tx *QueuedTx) SetHash(hash common.Hash) {
	tx.Lock()
	defer tx.Unlock()

	tx.hash = hash
}

// Context gets queued transaction context.
func (tx *QueuedTx) Context() context.Context {
	tx.RLock()
	defer tx.RUnlock()

	return tx.context
}

// SetContext sets queued transaction context.
func (tx *QueuedTx) SetContext(ctx context.Context) {
	tx.Lock()
	defer tx.Unlock()

	tx.context = ctx
}

// Args gets queued transaction args.
func (tx *QueuedTx) Args() SendTxArgs {
	tx.RLock()
	defer tx.RUnlock()

	return tx.args
}

// SetArgs sets queued transaction args.
func (tx *QueuedTx) SetArgs(args SendTxArgs) {
	tx.Lock()
	defer tx.Unlock()

	tx.args = args
}

// InProgress gets queued transaction progress state.
func (tx *QueuedTx) InProgress() bool {
	tx.RLock()
	defer tx.RUnlock()

	return tx.inProgress
}

// SetInProgress sets queued transaction progress state.
func (tx *QueuedTx) SetInProgress(p bool) {
	tx.Lock()
	defer tx.Unlock()

	tx.inProgress = p
}

// Done gets queued transaction done channel.
func (tx *QueuedTx) Done() chan struct{} {
	tx.RLock()
	defer tx.RUnlock()

	return tx.done
}

// Discard gets queued transaction discard channel.
func (tx *QueuedTx) Discard() chan struct{} {
	tx.RLock()
	defer tx.RUnlock()

	return tx.discard
}

// Err gets queued transaction error.
func (tx *QueuedTx) Err() error {
	tx.RLock()
	defer tx.RUnlock()

	return tx.err
}

// SetErr sets queued transaction error.
func (tx *QueuedTx) SetErr(err error) {
	tx.Lock()
	defer tx.Unlock()

	tx.err = err
}
