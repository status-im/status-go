package ethapi

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/les/status"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"golang.org/x/net/context"
)

// StatusBackend implements les.StatusBackend with direct calls to Ethereum
// internals to support calls from status-go bindings (to internal packages e.g. ethapi)
type StatusBackend struct {
	eapi  *PublicEthereumAPI        // Wrapper around the Ethereum object to access metadata
	bcapi *PublicBlockChainAPI      // Wrapper around the blockchain to access chain data
	txapi *PublicTransactionPoolAPI // Wrapper around the transaction pool to access transaction data

	txQueue *status.TxQueue
}

// NewStatusBackend creates a new backend using an existing Ethereum object.
func NewStatusBackend(apiBackend Backend) *StatusBackend {
	glog.V(logger.Info).Infof("Status backend service started")
	backend := &StatusBackend{
		eapi:    NewPublicEthereumAPI(apiBackend),
		bcapi:   NewPublicBlockChainAPI(apiBackend),
		txapi:   NewPublicTransactionPoolAPI(apiBackend),
		txQueue: status.NewTransactionQueue(),
	}

	go backend.transactionQueueForwardingLoop()

	return backend
}

func (b *StatusBackend) SetTransactionQueueHandler(fn status.EnqueuedTxHandler) {
	b.txQueue.SetEnqueueHandler(fn)
}

func (b *StatusBackend) TransactionQueue() *status.TxQueue {
	return b.txQueue
}

// SendTransaction wraps call to PublicTransactionPoolAPI.SendTransaction
func (b *StatusBackend) SendTransaction(ctx context.Context, args status.SendTxArgs) (common.Hash, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	return b.txapi.SendTransaction(ctx, SendTxArgs(args))
}

// CompleteQueuedTransaction wraps call to PublicTransactionPoolAPI.CompleteQueuedTransaction
func (b *StatusBackend) CompleteQueuedTransaction(id status.QueuedTxId, passphrase string) (common.Hash, error) {
	queuedTx, err := b.txQueue.Get(id)
	if err != nil {
		return common.Hash{}, err
	}

	hash, err := b.txapi.CompleteQueuedTransaction(context.Background(), SendTxArgs(queuedTx.Args), passphrase)
	queuedTx.Hash = hash
	queuedTx.Err = err
	queuedTx.Done <- struct{}{} // sendTransaction() waits on this, notify so that it can return

	return hash, err
}

func (b *StatusBackend) transactionQueueForwardingLoop() {
	txQueue, err := b.txapi.GetTransactionQueue()
	if err != nil {
		glog.V(logger.Error).Infof("cannot read from transaction queue")
		return
	}

	// forward internal ethapi transactions to status backend
	for queuedTx := range txQueue {
		b.txQueue.Enqueue(queuedTx)
	}
}
