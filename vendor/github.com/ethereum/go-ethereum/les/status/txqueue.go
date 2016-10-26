package status

import (
	"errors"
	"sync"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"
	"golang.org/x/net/context"
)

const (
	DefaultTxQueueCap              = int(35) // how many items can be queued
	DefaultTxSendQueueCap          = int(70) // how many items can be passed to sendTransaction() w/o blocking
	DefaultTxSendCompletionTimeout = 300     // how many seconds to wait before returning result in sentTransaction()
)

var (
	ErrQueuedTxIdNotFound = errors.New("transaction hash not found")
	ErrQueuedTxTimedOut   = errors.New("transaction sending timed out")
)

// TxQueue is capped container that holds pending transactions
type TxQueue struct {
	transactions  map[QueuedTxId]*QueuedTx
	mu            sync.RWMutex // to guard trasactions map
	evictableIds  chan QueuedTxId
	enqueueTicker chan struct{}

	// when items are enqueued notify subscriber
	txEnqueueHandler EnqueuedTxHandler

	// when tx is returned (either successfully or with error) notify subscriber
	txReturnHandler EnqueuedTxReturnHandler
}

// QueuedTx holds enough information to complete the queued transaction.
type QueuedTx struct {
	Id      QueuedTxId
	Hash    common.Hash
	Context context.Context
	Args    SendTxArgs
	Done    chan struct{}
	Err     error
}

type QueuedTxId string

// EnqueuedTxHandler is a function that receives queued/pending transactions, when they get queued
type EnqueuedTxHandler func(QueuedTx)

// EnqueuedTxReturnHandler is a function that receives response when tx is complete (both on success and error)
type EnqueuedTxReturnHandler func(queuedTx QueuedTx, err error)

// SendTxArgs represents the arguments to submbit a new transaction into the transaction pool.
type SendTxArgs struct {
	From     common.Address  `json:"from"`
	To       *common.Address `json:"to"`
	Gas      *rpc.HexNumber  `json:"gas"`
	GasPrice *rpc.HexNumber  `json:"gasPrice"`
	Value    *rpc.HexNumber  `json:"value"`
	Data     string          `json:"data"`
	Nonce    *rpc.HexNumber  `json:"nonce"`
}

func NewTransactionQueue() *TxQueue {
	txQueue := &TxQueue{
		transactions:  make(map[QueuedTxId]*QueuedTx),
		evictableIds:  make(chan QueuedTxId, DefaultTxQueueCap), // will be used to evict in FIFO
		enqueueTicker: make(chan struct{}),
	}

	go txQueue.evictionLoop()

	return txQueue
}

func (q *TxQueue) evictionLoop() {
	for range q.enqueueTicker {
		if len(q.transactions) >= (DefaultTxQueueCap - 1) { // eviction is required to accommodate another/last item
			q.Remove(<-q.evictableIds)
			q.enqueueTicker <- struct{}{} // in case we pulled already removed item
		}
	}
}

func (q *TxQueue) Enqueue(tx *QueuedTx) error {
	if q.txEnqueueHandler == nil { //discard, until handler is provided
		return nil
	}

	q.enqueueTicker <- struct{}{} // notify eviction loop that we are trying to insert new item
	q.evictableIds <- tx.Id       // this will block when we hit DefaultTxQueueCap

	q.mu.Lock()
	q.transactions[tx.Id] = tx
	q.mu.Unlock()

	// notify handler
	q.txEnqueueHandler(*tx)

	return nil
}

func (q *TxQueue) Get(id QueuedTxId) (*QueuedTx, error) {
	q.mu.RLock()
	defer q.mu.RUnlock()

	if tx, ok := q.transactions[id]; ok {
		return tx, nil
	}

	return nil, ErrQueuedTxIdNotFound
}

func (q *TxQueue) Remove(id QueuedTxId) {
	q.mu.Lock()
	defer q.mu.Unlock()

	delete(q.transactions, id)
}

func (q *TxQueue) Count() int {
	q.mu.RLock()
	defer q.mu.RUnlock()

	return len(q.transactions)
}

func (q *TxQueue) Has(id QueuedTxId) bool {
	q.mu.RLock()
	defer q.mu.RUnlock()

	_, ok := q.transactions[id]

	return ok
}

func (q *TxQueue) SetEnqueueHandler(fn EnqueuedTxHandler) {
	q.txEnqueueHandler = fn
}

func (q *TxQueue) SetTxReturnHandler(fn EnqueuedTxReturnHandler) {
	q.txReturnHandler = fn
}

func (q *TxQueue) NotifyOnQueuedTxReturn(id QueuedTxId, err error) {
	if q == nil {
		return
	}

	// on success, remove item from the queue and stop propagating
	if err == nil {
		q.Remove(id)
		return
	}

	// error occurred, send upward notification
	if q.txReturnHandler == nil { // discard, until handler is provided
		return
	}

	// discard, if transaction is not found
	tx, _ := q.Get(id)
	if tx == nil {
		return
	}

	// remove from queue on any error (except for password related one) and propagate
	if err != accounts.ErrDecrypt {
		q.Remove(id)
	}

	// notify handler
	q.txReturnHandler(*tx, err)
}
