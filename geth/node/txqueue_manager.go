package node

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/les/status"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/pborman/uuid"
	"github.com/status-im/status-go/geth/common"
	"github.com/status-im/status-go/geth/log"
)

const (
	// EventTransactionQueued is triggered when send transaction request is queued
	EventTransactionQueued = "transaction.queued"

	// EventTransactionFailed is triggered when send transaction request fails
	EventTransactionFailed = "transaction.failed"

	// SendTxDefaultErrorCode is sent by default, when error is not nil, but type is unknown/unexpected.
	SendTxDefaultErrorCode = SendTransactionDefaultErrorCode
)

// Send transaction response codes
const (
	SendTransactionNoErrorCode        = "0"
	SendTransactionDefaultErrorCode   = "1"
	SendTransactionPasswordErrorCode  = "2"
	SendTransactionTimeoutErrorCode   = "3"
	SendTransactionDiscardedErrorCode = "4"
)

var txReturnCodes = map[error]string{ // deliberately strings, in case more meaningful codes are to be returned
	nil:                  SendTransactionNoErrorCode,
	keystore.ErrDecrypt:  SendTransactionPasswordErrorCode,
	ErrQueuedTxTimedOut:  SendTransactionTimeoutErrorCode,
	ErrQueuedTxDiscarded: SendTransactionDiscardedErrorCode,
}

// TxQueueManager provides means to manage internal Status Backend (injected into LES)
type TxQueueManager struct {
	nodeManager    common.NodeManager
	accountManager common.AccountManager
	txQueue        *TxQueue
}

func NewTxQueueManager(nodeManager common.NodeManager, accountManager common.AccountManager) *TxQueueManager {
	return &TxQueueManager{
		nodeManager:    nodeManager,
		accountManager: accountManager,
		txQueue:        NewTransactionQueue(),
	}
}

func (m *TxQueueManager) Start() {
	log.Info("start TxQueueManager")
	m.txQueue.Start()
}

func (m *TxQueueManager) Stop() {
	log.Info("stop TxQueueManager")
	m.txQueue.Stop()
}

func (m *TxQueueManager) TransactionQueue() common.TxQueue {
	return m.txQueue
}

func (m *TxQueueManager) QueueTransactionAndWait(ctx context.Context, args common.SendTxArgs) (*common.QueuedTx, error) {
	log.Info(fmt.Sprintf("queue a new transaction from %s to %s and wait", args.From.Hex(), args.To.Hex()))

	tx := common.QueuedTx{
		ID:      common.QueuedTxID(uuid.New()),
		Hash:    gethcommon.Hash{},
		Context: ctx,
		Args:    args,
		Done:    make(chan struct{}, 1),
		Discard: make(chan struct{}, 1),
	}

	err := m.txQueue.Enqueue(&tx)
	if err != nil {
		return &tx, err
	}

	log.Info(fmt.Sprintf("queued transaction with ID %s successfully", tx.ID))

	// now wait up until transaction is:
	// - completed (via CompleteQueuedTransaction),
	// - discarded (via DiscardQueuedTransaction)
	// - or times out
	select {
	case <-tx.Done:
		m.NotifyOnQueuedTxReturn(&tx, tx.Err)
		return &tx, tx.Err
	case <-tx.Discard:
		m.NotifyOnQueuedTxReturn(&tx, ErrQueuedTxDiscarded)
		return &tx, ErrQueuedTxDiscarded
	case <-time.After(DefaultTxSendCompletionTimeout * time.Second):
		m.NotifyOnQueuedTxReturn(&tx, ErrQueuedTxTimedOut)
		return &tx, ErrQueuedTxTimedOut
	}
}

func (m *TxQueueManager) NotifyOnQueuedTxReturn(queuedTx *common.QueuedTx, err error) {
	m.txQueue.NotifyOnQueuedTxReturn(queuedTx, err)
}

// CompleteTransaction instructs backend to complete sending of a given transaction
func (m *TxQueueManager) CompleteTransaction(id, password string) (gethcommon.Hash, error) {
	log.Info(fmt.Sprintf("complete transaction with id %s", id))

	queuedTx, err := m.txQueue.Get(common.QueuedTxID(id))
	if err != nil {
		return gethcommon.Hash{}, err
	}

	selectedAccount, err := m.accountManager.SelectedAccount()
	if err != nil {
		return gethcommon.Hash{}, err
	}

	config, err := m.nodeManager.NodeConfig()
	if err != nil {
		return gethcommon.Hash{}, err
	}

	// make sure that only account which created the tx can complete it
	if queuedTx.Args.From.Hex() != selectedAccount.Address.Hex() {
		return gethcommon.Hash{}, ErrInvalidCompleteTxSender
	}

	// Send the transaction finally.
	var hash gethcommon.Hash
	var txErr error

	log.Info(fmt.Sprintf("sending transaction"))

	if config.UpstreamConfig.Enabled {
		hash, txErr = m.completeRemoteTransaction(queuedTx, password)
	} else {
		hash, txErr = m.completeLocalTransaction(queuedTx, password)
	}

	// when incorrect sender tries to complete the account,
	// notify and keep tx in queue (so that correct sender can complete)
	if txErr == keystore.ErrDecrypt {
		m.NotifyOnQueuedTxReturn(queuedTx, txErr)
		return hash, txErr
	}

	queuedTx.Hash = hash
	queuedTx.Err = txErr
	queuedTx.Done <- struct{}{} // sendTransaction() waits on this, notify so that it can return

	return hash, txErr
}

func (m *TxQueueManager) completeLocalTransaction(queuedTx *common.QueuedTx, password string) (gethcommon.Hash, error) {
	log.Info(fmt.Sprintf("completeLocalTransaction with id %s", queuedTx.ID))

	les, err := m.nodeManager.LightEthereumService()
	if err != nil {
		return gethcommon.Hash{}, err
	}

	return les.StatusBackend.SendTransaction(context.Background(), status.SendTxArgs(queuedTx.Args), password)
}

func (m *TxQueueManager) completeRemoteTransaction(queuedTx *common.QueuedTx, password string) (gethcommon.Hash, error) {
	log.Info(fmt.Sprintf("completeRemoteTransaction withd id %s", queuedTx.ID))

	var emptyHash gethcommon.Hash

	config, err := m.nodeManager.NodeConfig()
	if err != nil {
		return emptyHash, err
	}

	selectedAcct, err := m.accountManager.SelectedAccount()
	if err != nil {
		return emptyHash, err
	}

	client, err := m.nodeManager.RPCClient()
	if err != nil {
		return emptyHash, err
	}

	// We need to request a new transaction nounce from upstream node.
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	var txCount hexutil.Uint
	if err := client.CallContext(ctx, &txCount, "eth_getTransactionCount", queuedTx.Args.From, "pending"); err != nil {
		return emptyHash, err
	}

	chainID := big.NewInt(int64(config.NetworkID))
	nonce := uint64(txCount)
	gas := (*big.Int)(queuedTx.Args.Gas)
	gasPrice := (*big.Int)(queuedTx.Args.GasPrice)
	dataVal := []byte(queuedTx.Args.Data)
	priceVal := (*big.Int)(queuedTx.Args.Value)

	tx := types.NewTransaction(nonce, *queuedTx.Args.To, priceVal, gas, gasPrice, dataVal)
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), selectedAcct.AccountKey.PrivateKey)
	if err != nil {
		return emptyHash, err
	}

	txBytes, err := rlp.EncodeToBytes(signedTx)
	if err != nil {
		return emptyHash, err
	}

	ctx2, cancel2 := context.WithTimeout(context.Background(), time.Minute)
	defer cancel2()

	var result json.RawMessage
	if err := client.CallContext(ctx2, &result, "eth_sendRawTransaction", gethcommon.ToHex(txBytes)); err != nil {
		return emptyHash, err
	}

	return signedTx.Hash(), nil
}

// CompleteTransactions instructs backend to complete sending of multiple transactions
func (m *TxQueueManager) CompleteTransactions(ids, password string) map[string]common.RawCompleteTransactionResult {
	results := make(map[string]common.RawCompleteTransactionResult)

	parsedIDs, err := common.ParseJSONArray(ids)
	if err != nil {
		results["none"] = common.RawCompleteTransactionResult{
			Error: err,
		}
		return results
	}

	for _, txID := range parsedIDs {
		txHash, txErr := m.CompleteTransaction(txID, password)
		results[txID] = common.RawCompleteTransactionResult{
			Hash:  txHash,
			Error: txErr,
		}
	}

	return results
}

// DiscardTransaction discards a given transaction from transaction queue
func (m *TxQueueManager) DiscardTransaction(id string) error {
	queuedTx, err := m.txQueue.Get(common.QueuedTxID(id))
	if err != nil {
		return err
	}

	// remove from queue, before notifying SendTransaction
	m.txQueue.Remove(queuedTx.ID)

	// allow SendTransaction to return
	queuedTx.Err = ErrQueuedTxDiscarded
	queuedTx.Discard <- struct{}{} // sendTransaction() waits on this, notify so that it can return

	return nil
}

// DiscardTransactions discards given multiple transactions from transaction queue
func (m *TxQueueManager) DiscardTransactions(ids string) map[string]common.RawDiscardTransactionResult {
	var parsedIDs []string
	results := make(map[string]common.RawDiscardTransactionResult)

	parsedIDs, err := common.ParseJSONArray(ids)
	if err != nil {
		results["none"] = common.RawDiscardTransactionResult{
			Error: err,
		}
		return results
	}

	for _, txID := range parsedIDs {
		err := m.DiscardTransaction(txID)
		if err != nil {
			results[txID] = common.RawDiscardTransactionResult{
				Error: err,
			}
		}
	}

	return results
}

// SendTransactionEvent is a signal sent on a send transaction request
type SendTransactionEvent struct {
	ID        string            `json:"id"`
	Args      common.SendTxArgs `json:"args"`
	MessageID string            `json:"message_id"`
}

// TransactionQueueHandler returns handler that processes incoming tx queue requests
func (m *TxQueueManager) TransactionQueueHandler() func(queuedTx common.QueuedTx) {
	return func(queuedTx common.QueuedTx) {
		log.Info("calling TransactionQueueHandler")
		SendSignal(SignalEnvelope{
			Type: EventTransactionQueued,
			Event: SendTransactionEvent{
				ID:        string(queuedTx.ID),
				Args:      queuedTx.Args,
				MessageID: common.MessageIDFromContext(queuedTx.Context),
			},
		})
	}
}

func (m *TxQueueManager) SetTransactionQueueHandler(fn common.EnqueuedTxHandler) {
	m.txQueue.SetEnqueueHandler(fn)
}

// ReturnSendTransactionEvent is a JSON returned whenever transaction send is returned
type ReturnSendTransactionEvent struct {
	ID           string            `json:"id"`
	Args         common.SendTxArgs `json:"args"`
	MessageID    string            `json:"message_id"`
	ErrorMessage string            `json:"error_message"`
	ErrorCode    string            `json:"error_code"`
}

// TransactionReturnHandler returns handler that processes responses from internal tx manager
func (m *TxQueueManager) TransactionReturnHandler() func(queuedTx *common.QueuedTx, err error) {
	return func(queuedTx *common.QueuedTx, err error) {
		if err == nil {
			return
		}

		// discard notifications with empty tx
		if queuedTx == nil {
			return
		}

		// error occurred, signal up to application
		SendSignal(SignalEnvelope{
			Type: EventTransactionFailed,
			Event: ReturnSendTransactionEvent{
				ID:           string(queuedTx.ID),
				Args:         queuedTx.Args,
				MessageID:    common.MessageIDFromContext(queuedTx.Context),
				ErrorMessage: err.Error(),
				ErrorCode:    m.sendTransactionErrorCode(err),
			},
		})
	}
}

func (m *TxQueueManager) sendTransactionErrorCode(err error) string {
	if code, ok := txReturnCodes[err]; ok {
		return code
	}

	return SendTxDefaultErrorCode
}

func (m *TxQueueManager) SetTransactionReturnHandler(fn common.EnqueuedTxReturnHandler) {
	m.txQueue.SetTxReturnHandler(fn)
}
