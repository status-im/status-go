package txqueue

import (
	"sync"

	"github.com/status-im/status-go/geth/common"
)

// transactions holds
type transactions struct {
	m map[common.QueuedTxID]*common.QueuedTx
	l sync.RWMutex
}

// newTransactions is a transaction constructor.
func newTransactions() *transactions {
	return &transactions{m: make(map[common.QueuedTxID]*common.QueuedTx)}
}

// reset transactions state.
func (tr *transactions) reset() {
	tr.l.Lock()
	defer tr.l.Unlock()

	tr.m = make(map[common.QueuedTxID]*common.QueuedTx)
}

// add transaction with key ID.
func (tr *transactions) add(key common.QueuedTxID, value *common.QueuedTx) {
	tr.l.Lock()
	defer tr.l.Unlock()

	tr.m[key] = value
}

// get transaction by a key.
func (tr *transactions) get(key common.QueuedTxID) (value *common.QueuedTx, ok bool) {
	tr.l.RLock()
	defer tr.l.RUnlock()

	value, ok = tr.m[key]
	return
}

// delete transaction by a key.
func (tr *transactions) delete(key common.QueuedTxID) {
	tr.l.Lock()
	defer tr.l.Unlock()

	delete(tr.m, key)
}

// len counts transactions.
func (tr *transactions) len() int {
	tr.l.RLock()
	defer tr.l.RUnlock()

	return len(tr.m)
}
