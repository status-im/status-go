package txqueue

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/status-im/status-go/geth/common"
	"github.com/status-im/status-go/geth/log"
)

const (
	// DefaultTxQueueCap defines how many items can be queued.
	DefaultTxQueueCap = int(35)
	// DefaultTxSendQueueCap defines how many items can be passed to sendTransaction() w/o blocking.
	DefaultTxSendQueueCap = int(70)
	// DefaultTxSendCompletionTimeout defines how many seconds to wait before returning result in sentTransaction().
	DefaultTxSendCompletionTimeout = 300
)

var (
	//ErrQueuedTxIDNotFound - error transaction hash not found
	ErrQueuedTxIDNotFound = errors.New("transaction hash not found")
	//ErrQueuedTxTimedOut - error transaction sending timed out
	ErrQueuedTxTimedOut = errors.New("transaction sending timed out")
	//ErrInvalidCompleteTxSender - error transaction with invalid sender
	ErrInvalidCompleteTxSender = errors.New("transaction can only be completed by the same account which created it")
)

// TxQueue is capped container that holds pending transactions
type TxQueue struct {
	transactions  *transactions
	evictableIDs  chan common.QueuedTxID
	enqueueTicker chan struct{}
	incomingPool  chan *common.QueuedTx

	// when this channel is closed, all queue channels processing must cease (incoming queue, processing queued items etc)
	stopped      chan struct{}
	stoppedGroup sync.WaitGroup // to make sure that all routines are stopped

	// when items are enqueued notify subscriber
	txEnqueueHandler common.EnqueuedTxHandler

	// when tx is returned (either successfully or with error) notify subscriber
	txReturnHandler common.EnqueuedTxReturnHandler
}

// NewTransactionQueue make new transaction queue
func NewTransactionQueue() *TxQueue {
	log.Info("initializing transaction queue")
	return &TxQueue{
		transactions:  newTransactions(),
		evictableIDs:  make(chan common.QueuedTxID, DefaultTxQueueCap), // will be used to evict in FIFO
		enqueueTicker: make(chan struct{}),
		incomingPool:  make(chan *common.QueuedTx, DefaultTxSendQueueCap),
	}
}

// Start starts enqueue and eviction loops
func (q *TxQueue) Start() {
	log.Info("starting transaction queue")

	if q.stopped != nil {
		return
	}

	q.stopped = make(chan struct{})
	q.stoppedGroup.Add(2)

	go q.evictionLoop()
	go q.enqueueLoop()
}

// Stop stops transaction enqueue and eviction loops
func (q *TxQueue) Stop() {
	log.Info("stopping transaction queue")

	if q.stopped == nil {
		return
	}

	close(q.stopped) // stops all processing loops (enqueue, eviction etc)
	q.stoppedGroup.Wait()
	q.stopped = nil

	log.Info("finally stopped transaction queue")
}

// evictionLoop frees up queue to accommodate another transaction item
func (q *TxQueue) evictionLoop() {
	defer HaltOnPanic()
	evict := func() {
		if q.transactions.len() >= DefaultTxQueueCap { // eviction is required to accommodate another/last item
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
			log.Info("transaction queue's eviction loop stopped")
			q.stoppedGroup.Done()
			return
		}
	}
}

// enqueueLoop process incoming enqueue requests
func (q *TxQueue) enqueueLoop() {
	defer HaltOnPanic()

	// enqueue incoming transactions
	for {
		select {
		case queuedTx := <-q.incomingPool:
			log.Info("transaction enqueued requested", "tx", queuedTx.ID())
			err := q.Enqueue(queuedTx)
			log.Warn("transaction enqueued error", "tx", err)
			log.Info("transaction enqueued", "tx", queuedTx.ID())
		case <-q.stopped:
			log.Info("transaction queue's enqueue loop stopped")
			q.stoppedGroup.Done()
			return
		}
	}
}

// Reset is to be used in tests only, as it simply creates new transaction map, w/o any cleanup of the previous one
func (q *TxQueue) Reset() {
	q.transactions.reset()
	q.evictableIDs = make(chan common.QueuedTxID, DefaultTxQueueCap)
}

// EnqueueAsync enqueues incoming transaction in async manner, returns as soon as possible
func (q *TxQueue) EnqueueAsync(tx *common.QueuedTx) error {
	q.incomingPool <- tx

	return nil
}

// Enqueue enqueues incoming transaction
func (q *TxQueue) Enqueue(tx *common.QueuedTx) error {
	log.Info(fmt.Sprintf("enqueue transaction: %s", tx.ID()))

	if q.txEnqueueHandler == nil { //discard, until handler is provided
		log.Info("there is no txEnqueueHandler")
		return nil
	}

	log.Info("before enqueueTicker")
	q.enqueueTicker <- struct{}{} // notify eviction loop that we are trying to insert new item
	log.Info("before evictableIDs")
	q.evictableIDs <- tx.ID() // this will block when we hit DefaultTxQueueCap
	log.Info("after evictableIDs")

	q.transactions.add(tx.ID(), tx)

	// notify handler
	log.Info("calling txEnqueueHandler")
	q.txEnqueueHandler(tx)

	return nil
}

// Get returns transaction by transaction identifier
func (q *TxQueue) Get(id common.QueuedTxID) (*common.QueuedTx, error) {
	tx, ok := q.transactions.get(id)
	if !ok {
		return nil, ErrQueuedTxIDNotFound
	}

	return tx, nil
}

// Set transaction by transaction identifier
func (q *TxQueue) Set(id common.QueuedTxID, tx *common.QueuedTx) {
	q.transactions.add(id, tx)
}

// Remove removes transaction by transaction identifier
func (q *TxQueue) Remove(id common.QueuedTxID) {
	q.transactions.delete(id)
}

// Count returns number of currently queued transactions
func (q *TxQueue) Count() int {
	return q.transactions.len()
}

// Has checks whether transaction with a given identifier exists in queue
func (q *TxQueue) Has(id common.QueuedTxID) bool {
	_, ok := q.transactions.get(id)
	return ok
}

// SetEnqueueHandler sets callback handler, that is triggered on enqueue operation
func (q *TxQueue) SetEnqueueHandler(fn common.EnqueuedTxHandler) {
	q.txEnqueueHandler = fn
}

// SetTxReturnHandler sets callback handler, that is triggered when transaction is finished executing
func (q *TxQueue) SetTxReturnHandler(fn common.EnqueuedTxReturnHandler) {
	q.txReturnHandler = fn
}

// NotifyOnQueuedTxReturn is invoked when transaction is ready to return
// Transaction can be in error state, or executed successfully at this point.
func (q *TxQueue) NotifyOnQueuedTxReturn(queuedTx *common.QueuedTx, err error) {
	if q == nil {
		return
	}

	// discard, if transaction is not found
	if queuedTx == nil {
		return
	}

	// on success, remove item from the queue and stop propagating
	if err == nil {
		q.Remove(queuedTx.ID())
		return
	}

	// error occurred, send upward notification
	if q.txReturnHandler == nil { // discard, until handler is provided
		return
	}

	// remove from queue on any error (except for transient ones) and propagate
	transientErrs := map[error]bool{
		keystore.ErrDecrypt:        true, // wrong password
		ErrInvalidCompleteTxSender: true, // completing tx create from another account
	}
	if !transientErrs[err] { // remove only on unrecoverable errors
		q.Remove(queuedTx.ID())
	}

	// notify handler
	q.txReturnHandler(queuedTx, err)
}
