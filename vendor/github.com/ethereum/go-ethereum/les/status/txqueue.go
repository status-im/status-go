package status

import (
	"errors"
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
)

// TxQueue is capped container that holds pending transactions
type TxQueue struct {
	transactions  map[QueuedTxId]*QueuedTx
	evictableIds  chan QueuedTxId
	enqueueTicker chan struct{}

	// when items are enqueued notify handlers
	txEnqueueHandler EnqueuedTxHandler
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

// QueuedTxHandler is a function that receives queued/pending transactions, when they get queued
type EnqueuedTxHandler func(QueuedTx)

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
			delete(q.transactions, <-q.evictableIds)
		}
	}
}

func (q *TxQueue) Enqueue(tx *QueuedTx) error {
	if q.txEnqueueHandler == nil { //discard, until handler is provided
		return nil
	}

	q.enqueueTicker <- struct{}{} // notify eviction loop that we are trying to insert new item
	q.evictableIds <- tx.Id       // this will block when we hit DefaultTxQueueCap

	q.transactions[tx.Id] = tx

	// notify handler
	q.txEnqueueHandler(*tx)

	return nil
}

func (q *TxQueue) Get(id QueuedTxId) (*QueuedTx, error) {
	if tx, ok := q.transactions[id]; ok {
		delete(q.transactions, id)
		return tx, nil
	}

	return nil, ErrQueuedTxIdNotFound
}

func (q *TxQueue) Count() int {
	return len(q.transactions)
}

func (q *TxQueue) Has(id QueuedTxId) bool {
	_, ok := q.transactions[id]

	return ok
}

func (q *TxQueue) SetEnqueueHandler(fn EnqueuedTxHandler) {
	q.txEnqueueHandler = fn
}
