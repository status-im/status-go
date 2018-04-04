package transactions

import (
	"errors"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/status-im/status-go/geth/account"
)

const (
	// DefaultTxQueueCap defines how many items can be queued.
	DefaultTxQueueCap = int(35)
)

var (
	// ErrQueuedTxExist - transaction was already enqueued
	ErrQueuedTxExist = errors.New("transaction already exist in queue")
	//ErrQueuedTxIDNotFound - error transaction hash not found
	ErrQueuedTxIDNotFound = errors.New("transaction hash not found")
	//ErrQueuedTxInProgress - error transaction is in progress
	ErrQueuedTxInProgress = errors.New("transaction is in progress")
	//ErrInvalidCompleteTxSender - error transaction with invalid sender
	ErrInvalidCompleteTxSender = errors.New("transaction can only be completed by the same account which created it")
)

// remove from queue on any error (except for transient ones) and propagate
// defined as map[string]bool because errors from ethclient returned wrapped as jsonError
var transientErrs = map[string]bool{
	keystore.ErrDecrypt.Error():          true, // wrong password
	ErrInvalidCompleteTxSender.Error():   true, // completing tx create from another account
	account.ErrNoAccountSelected.Error(): true, // account not selected
}

type empty struct{}

// TxQueue is capped container that holds pending transactions
type TxQueue struct {
	mu           sync.RWMutex // to guard transactions map
	transactions map[string]*QueuedTx
	inprogress   map[string]empty

	// TODO(dshulyak) research why eviction is done in separate goroutine
	evictableIDs  chan string
	enqueueTicker chan struct{}

	// when this channel is closed, all queue channels processing must cease (incoming queue, processing queued items etc)
	stopped      chan struct{}
	stoppedGroup sync.WaitGroup // to make sure that all routines are stopped
	log          log.Logger
}

// newQueue creates a transaction queue.
func newQueue() *TxQueue {

	logger := log.New("package", "status-go/geth/transactions.TxQueue")

	logger.Info("initializing transaction queue")
	return &TxQueue{
		transactions:  make(map[string]*QueuedTx),
		inprogress:    make(map[string]empty),
		evictableIDs:  make(chan string, DefaultTxQueueCap), // will be used to evict in FIFO
		enqueueTicker: make(chan struct{}),
		log:           logger,
	}
}

// Start starts enqueue and eviction loops
func (q *TxQueue) Start() {
	q.log.Info("starting transaction queue")

	if q.stopped != nil {
		return
	}

	q.stopped = make(chan struct{})
	q.stoppedGroup.Add(1)
	go q.evictionLoop()
}

// Stop stops transaction enqueue and eviction loops
func (q *TxQueue) Stop() {
	q.log.Info("stopping transaction queue")

	if q.stopped == nil {
		return
	}

	close(q.stopped) // stops all processing loops (enqueue, eviction etc)
	q.stoppedGroup.Wait()
	q.stopped = nil

	q.log.Info("finally stopped transaction queue")
}

// evictionLoop frees up queue to accommodate another transaction item
func (q *TxQueue) evictionLoop() {
	defer haltOnPanic()
	evict := func() {
		if q.Count() >= DefaultTxQueueCap { // eviction is required to accommodate another/last item
			q.Remove(<-q.evictableIDs)
		}
	}

	for {
		select {
		case <-time.After(250 * time.Millisecond): // do not wait for manual ticks, check queue regularly
			evict()
		case <-q.enqueueTicker: // when manually requested
			evict()
		case <-q.stopped:
			q.log.Info("transaction queue's eviction loop stopped")
			q.stoppedGroup.Done()
			return
		}
	}
}

// Reset is to be used in tests only, as it simply creates new transaction map, w/o any cleanup of the previous one
func (q *TxQueue) Reset() {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.transactions = make(map[string]*QueuedTx)
	q.evictableIDs = make(chan string, DefaultTxQueueCap)
	q.inprogress = make(map[string]empty)
}

// Enqueue enqueues incoming transaction
func (q *TxQueue) Enqueue(tx *QueuedTx) error {
	q.log.Info("enqueue transaction", "ID", tx.ID)
	q.mu.RLock()
	if _, ok := q.transactions[tx.ID]; ok {
		q.mu.RUnlock()
		return ErrQueuedTxExist
	}
	q.mu.RUnlock()

	// we can't hold a lock in this part
	q.log.Debug("notifying eviction loop")
	q.enqueueTicker <- struct{}{} // notify eviction loop that we are trying to insert new item
	q.evictableIDs <- tx.ID       // this will block when we hit DefaultTxQueueCap
	q.log.Debug("notified eviction loop")

	q.mu.Lock()
	q.transactions[tx.ID] = tx
	q.mu.Unlock()

	// notify handler
	q.log.Info("calling txEnqueueHandler")
	return nil
}

// Get returns transaction by transaction identifier
func (q *TxQueue) Get(id string) (*QueuedTx, error) {
	q.mu.RLock()
	defer q.mu.RUnlock()

	if tx, ok := q.transactions[id]; ok {
		return tx, nil
	}
	return nil, ErrQueuedTxIDNotFound
}

// LockInprogress returns error if transaction is already inprogress.
func (q *TxQueue) LockInprogress(id string) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	if _, ok := q.transactions[id]; ok {
		if _, inprogress := q.inprogress[id]; inprogress {
			return ErrQueuedTxInProgress
		}
		q.inprogress[id] = empty{}
		return nil
	}
	return ErrQueuedTxIDNotFound
}

// Remove removes transaction by transaction identifier
func (q *TxQueue) Remove(id string) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.remove(id)
}

func (q *TxQueue) remove(id string) {
	delete(q.transactions, id)
	delete(q.inprogress, id)
}

// Done removes transaction from queue if no error or error is not transient
// and notify subscribers
func (q *TxQueue) Done(id string, hash gethcommon.Hash, err error) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	tx, ok := q.transactions[id]
	if !ok {
		return ErrQueuedTxIDNotFound
	}
	q.done(tx, hash, err)
	return nil
}

func (q *TxQueue) done(tx *QueuedTx, hash gethcommon.Hash, err error) {
	delete(q.inprogress, tx.ID)
	// hash is updated only if err is nil, but transaction is not removed from a queue
	if err == nil {
		q.transactions[tx.ID].Result <- Result{Hash: hash, Error: err}
		q.remove(tx.ID)
		return
	}
	if _, transient := transientErrs[err.Error()]; !transient {
		q.transactions[tx.ID].Result <- Result{Error: err}
		q.remove(tx.ID)
	}
}

// Count returns number of currently queued transactions
func (q *TxQueue) Count() int {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return len(q.transactions)
}

// Has checks whether transaction with a given identifier exists in queue
func (q *TxQueue) Has(id string) bool {
	q.mu.RLock()
	defer q.mu.RUnlock()
	_, ok := q.transactions[id]
	return ok
}
