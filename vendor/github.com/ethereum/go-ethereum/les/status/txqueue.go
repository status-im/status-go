package status

import (
	"errors"
	"sync"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"golang.org/x/net/context"
)

const (
	DefaultTxQueueCap              = int(35) // how many items can be queued
	DefaultTxSendQueueCap          = int(70) // how many items can be passed to sendTransaction() w/o blocking
	DefaultTxSendCompletionTimeout = 300     // how many seconds to wait before returning result in sentTransaction()
	SelectedAccountKey             = "selected_account"
)

var (
	ErrQueuedTxIdNotFound      = errors.New("transaction hash not found")
	ErrQueuedTxTimedOut        = errors.New("transaction sending timed out")
	ErrQueuedTxDiscarded       = errors.New("transaction has been discarded")
	ErrInvalidCompleteTxSender = errors.New("transaction can only be completed by the same account which created it")
)

// TxQueue is capped container that holds pending transactions
type TxQueue struct {
	transactions  map[QueuedTxId]*QueuedTx
	mu            sync.RWMutex // to guard transactions map
	evictableIds  chan QueuedTxId
	enqueueTicker chan struct{}
	incomingPool  chan *QueuedTx

	// when this channel is closed, all queue channels processing must cease (incoming queue, processing queued items etc)
	stopped      chan struct{}
	stoppedGroup sync.WaitGroup // to make sure that all routines are stopped

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
	Discard chan struct{}
	Err     error
}

type QueuedTxId string

// EnqueuedTxHandler is a function that receives queued/pending transactions, when they get queued
type EnqueuedTxHandler func(QueuedTx)

// EnqueuedTxReturnHandler is a function that receives response when tx is complete (both on success and error)
type EnqueuedTxReturnHandler func(queuedTx *QueuedTx, err error)

// SendTxArgs represents the arguments to submit a new transaction into the transaction pool.
type SendTxArgs struct {
	From     common.Address  `json:"from"`
	To       *common.Address `json:"to"`
	Gas      *hexutil.Big    `json:"gas"`
	GasPrice *hexutil.Big    `json:"gasPrice"`
	Value    *hexutil.Big    `json:"value"`
	Data     hexutil.Bytes   `json:"data"`
	Nonce    *hexutil.Uint64 `json:"nonce"`
}

func NewTransactionQueue() *TxQueue {
	glog.V(logger.Info).Infof("StatusIM: initializing transaction queue")
	return &TxQueue{
		transactions:  make(map[QueuedTxId]*QueuedTx),
		evictableIds:  make(chan QueuedTxId, DefaultTxQueueCap), // will be used to evict in FIFO
		enqueueTicker: make(chan struct{}),
		incomingPool:  make(chan *QueuedTx, DefaultTxSendQueueCap),
	}
}

func (q *TxQueue) Start() {
	glog.V(logger.Info).Infof("StatusIM: starting transaction queue")

	q.stopped = make(chan struct{})
	q.stoppedGroup.Add(2)

	go q.evictionLoop()
	go q.enqueueLoop()
}

func (q *TxQueue) Stop() {
	glog.V(logger.Info).Infof("StatusIM: stopping transaction queue")
	close(q.stopped) // stops all processing loops (enqueue, eviction etc)
	q.stoppedGroup.Wait()
}

func (q *TxQueue) evictionLoop() {
	for {
		select {
		case <-q.enqueueTicker:
			if len(q.transactions) >= (DefaultTxQueueCap - 1) { // eviction is required to accommodate another/last item
				q.Remove(<-q.evictableIds)
				q.enqueueTicker <- struct{}{} // in case we pulled already removed item
			}
		case <-q.stopped:
			glog.V(logger.Info).Infof("StatusIM: transaction queue's eviction loop stopped")
			q.stoppedGroup.Done()
			return
		}
	}
}

func (q *TxQueue) enqueueLoop() {
	// enqueue incoming transactions
	for {
		select {
		case queuedTx := <-q.incomingPool:
			glog.V(logger.Info).Infof("StatusIM: transaction enqueued %v", queuedTx.Id)
			q.Enqueue(queuedTx)
		case <-q.stopped:
			glog.V(logger.Info).Infof("StatusIM: transaction queue's enqueue loop stopped")
			q.stoppedGroup.Done()
			return
		}
	}
}

// Reset is to be used in tests only, as it simply creates new transaction map, w/o any cleanup of the previous one
func (q *TxQueue) Reset() {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.transactions = make(map[QueuedTxId]*QueuedTx)
	q.evictableIds = make(chan QueuedTxId, DefaultTxQueueCap)
}

func (q *TxQueue) EnqueueAsync(tx *QueuedTx) error {
	q.incomingPool <- tx

	return nil
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

func (q *TxQueue) NotifyOnQueuedTxReturn(queuedTx *QueuedTx, err error) {
	if q == nil {
		return
	}

	// discard, if transaction is not found
	if queuedTx == nil {
		return
	}

	// on success, remove item from the queue and stop propagating
	if err == nil {
		q.Remove(queuedTx.Id)
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
		q.Remove(queuedTx.Id)
	}

	// notify handler
	q.txReturnHandler(queuedTx, err)
}
