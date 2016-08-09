package les

import (
	"golang.org/x/net/context"
	"sync"

	"errors"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/internal/ethapi"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/rpc"
)

const (
	defaultTxQueueCap                  = int(5)  // how many items can be passed to sendTransaction() w/o blocking
	defaultEvictingTxQueueCap          = int(20) // how many items can be queued
	defaultEvictingTxQueueEvictionStep = int(5)  // how many item to evict in a single run
)

var (
	ErrQueuedTxHashNotFound = errors.New("Transaction hash not found")
)

// StatusBackend implements les.StatusBackend with direct calls to Ethereum
// internals to support calls from status-go bindings (to internal packages e.g. ethapi)
type StatusBackend struct {
	eapi  *ethapi.PublicEthereumAPI        // Wrapper around the Ethereum object to access metadata
	bcapi *ethapi.PublicBlockChainAPI      // Wrapper around the blockchain to access chain data
	txapi *ethapi.PublicTransactionPoolAPI // Wrapper around the transaction pool to access transaction data

	txQueue          chan QueuedTx
	txQueueHandler   QueuedTxHandler
	muTxQueueHanlder sync.Mutex

	txEvictingQueue evictingTxQueue
}

type QueuedTxHash string

type evictingTxQueue struct {
	transactions  map[QueuedTxHash]*QueuedTx
	evictionQueue chan QueuedTxHash
	cap           int
	mu            sync.Mutex
}

type QueuedTxHandler func(QueuedTx)

type QueuedTx struct {
	Hash    common.Hash
	Context context.Context
	Args    SendTxArgs
}

// SendTxArgs represents the arguments to sumbit a new transaction into the transaction pool.
type SendTxArgs struct {
	From     common.Address  `json:"from"`
	To       *common.Address `json:"to"`
	Gas      *rpc.HexNumber  `json:"gas"`
	GasPrice *rpc.HexNumber  `json:"gasPrice"`
	Value    *rpc.HexNumber  `json:"value"`
	Data     string          `json:"data"`
	Nonce    *rpc.HexNumber  `json:"nonce"`
}

// NewStatusBackend creates a new backend using an existing Ethereum object.
func NewStatusBackend(apiBackend ethapi.Backend) *StatusBackend {
	glog.V(logger.Debug).Infof("Status service started")
	backend := &StatusBackend{
		eapi:    ethapi.NewPublicEthereumAPI(apiBackend, nil, nil),
		bcapi:   ethapi.NewPublicBlockChainAPI(apiBackend),
		txapi:   ethapi.NewPublicTransactionPoolAPI(apiBackend),
		txQueue: make(chan QueuedTx, defaultTxQueueCap),
		txEvictingQueue: evictingTxQueue{
			transactions:  make(map[QueuedTxHash]*QueuedTx),
			evictionQueue: make(chan QueuedTxHash, defaultEvictingTxQueueCap), // will be used to evict in FIFO
			cap:           defaultEvictingTxQueueCap,
		},
	}

	go backend.transactionQueueForwardingLoop()

	return backend
}

func (b *StatusBackend) SetTransactionQueueHandler(fn QueuedTxHandler) {
	b.muTxQueueHanlder.Lock()
	defer b.muTxQueueHanlder.Unlock()

	b.txQueueHandler = fn
}

// SendTransaction wraps call to PublicTransactionPoolAPI.SendTransaction
func (b *StatusBackend) SendTransaction(ctx context.Context, args SendTxArgs) error {
	if ctx == nil {
		ctx = context.Background()
	}

	_, err := b.txapi.SendTransaction(ctx, ethapi.SendTxArgs(args))
	return err
}

// CompleteQueuedTransaction wraps call to PublicTransactionPoolAPI.CompleteQueuedTransaction
func (b *StatusBackend) CompleteQueuedTransaction(hash QueuedTxHash) (common.Hash, error) {
	queuedTx, err := b.txEvictingQueue.getQueuedTransaction(hash)
	if err != nil {
		return common.Hash{}, err
	}

	return b.txapi.CompleteQueuedTransaction(context.Background(), ethapi.SendTxArgs(queuedTx.Args))
}

// GetTransactionQueue wraps call to PublicTransactionPoolAPI.GetTransactionQueue
func (b *StatusBackend) GetTransactionQueue() (chan QueuedTx, error) {
	return b.txQueue, nil
}

func (b *StatusBackend) transactionQueueForwardingLoop() {
	txQueue, err := b.txapi.GetTransactionQueue()
	if err != nil {
		glog.V(logger.Error).Infof("cannot read from transaction queue")
		return
	}

	// forward internal ethapi transactions
	for queuedTx := range txQueue {
		if b.txQueueHandler == nil { //discard, until handler is provided
			continue
		}
		tx := QueuedTx{
			Hash:    queuedTx.Hash,
			Context: queuedTx.Context,
			Args:    SendTxArgs(queuedTx.Args),
		}
		b.txEvictingQueue.enqueueQueuedTransaction(tx)
		b.txQueueHandler(tx)
	}
}

func (q *evictingTxQueue) enqueueQueuedTransaction(tx QueuedTx) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.cap <= len(q.transactions) { // eviction is required
		for i := 0; i < defaultEvictingTxQueueEvictionStep; i++ {
			hash := <-q.evictionQueue
			delete(q.transactions, hash)
		}
	}

	q.transactions[QueuedTxHash(tx.Hash.Hex())] = &tx
	q.evictionQueue <- QueuedTxHash(tx.Hash.Hex())

	return nil
}

func (q *evictingTxQueue) getQueuedTransaction(hash QueuedTxHash) (*QueuedTx, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if tx, ok := q.transactions[hash]; ok {
		delete(q.transactions, hash)
		return tx, nil
	}

	return nil, ErrQueuedTxHashNotFound
}
