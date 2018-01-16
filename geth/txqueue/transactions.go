package txqueue

import (
	"sync"

	"github.com/status-im/status-go/geth/common"
)

// transactions safely holds queued transactions.
type transactions struct {
	mu sync.RWMutex
	m  map[common.QueuedTxID]*common.QueuedTx
}

// newTransactions is a transaction constructor.
func newTransactions() *transactions {
	return &transactions{m: make(map[common.QueuedTxID]*common.QueuedTx)}
}

// reset transactions state.
func (t *transactions) reset() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.m = make(map[common.QueuedTxID]*common.QueuedTx)
}

// add transaction with key ID.
func (t *transactions) add(key common.QueuedTxID, value *common.QueuedTx) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.m[key] = value
}

// get transaction by a key.
func (t *transactions) get(key common.QueuedTxID) (value *common.QueuedTx, ok bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	value, ok = t.m[key]
	return
}

// delete transaction by a key.
func (t *transactions) delete(key common.QueuedTxID) {
	t.mu.Lock()
	defer t.mu.Unlock()

	delete(t.m, key)
}

// len counts transactions.
func (t *transactions) len() int {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return len(t.m)
}
