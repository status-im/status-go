package node

import (
	"context"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/les/status"
	"github.com/status-im/status-go/geth/common"
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
	nil:                         SendTransactionNoErrorCode,
	keystore.ErrDecrypt:         SendTransactionPasswordErrorCode,
	status.ErrQueuedTxTimedOut:  SendTransactionTimeoutErrorCode,
	status.ErrQueuedTxDiscarded: SendTransactionDiscardedErrorCode,
}

// TxQueueManager provides means to manage internal Status Backend (injected into LES)
type TxQueueManager struct {
	nodeManager    common.NodeManager
	accountManager common.AccountManager
}

func NewTxQueueManager(nodeManager common.NodeManager, accountManager common.AccountManager) *TxQueueManager {
	return &TxQueueManager{
		nodeManager:    nodeManager,
		accountManager: accountManager,
	}
}

// CompleteTransaction instructs backend to complete sending of a given transaction
func (m *TxQueueManager) CompleteTransaction(id, password string) (gethcommon.Hash, error) {
	lightEthereum, err := m.nodeManager.LightEthereumService()
	if err != nil {
		return gethcommon.Hash{}, err
	}

	backend := lightEthereum.StatusBackend

	selectedAccount, err := m.accountManager.SelectedAccount()
	if err != nil {
		return gethcommon.Hash{}, err
	}

	ctx := context.Background()
	ctx = context.WithValue(ctx, status.SelectedAccountKey, selectedAccount.Hex())

	return backend.CompleteQueuedTransaction(ctx, status.QueuedTxID(id), password)
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
	lightEthereum, err := m.nodeManager.LightEthereumService()
	if err != nil {
		return err
	}

	backend := lightEthereum.StatusBackend

	return backend.DiscardQueuedTransaction(status.QueuedTxID(id))
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
	Args      status.SendTxArgs `json:"args"`
	MessageID string            `json:"message_id"`
}

// TransactionQueueHandler returns handler that processes incoming tx queue requests
func (m *TxQueueManager) TransactionQueueHandler() func(queuedTx status.QueuedTx) {
	return func(queuedTx status.QueuedTx) {
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

// ReturnSendTransactionEvent is a JSON returned whenever transaction send is returned
type ReturnSendTransactionEvent struct {
	ID           string            `json:"id"`
	Args         status.SendTxArgs `json:"args"`
	MessageID    string            `json:"message_id"`
	ErrorMessage string            `json:"error_message"`
	ErrorCode    string            `json:"error_code"`
}

// TransactionReturnHandler returns handler that processes responses from internal tx manager
func (m *TxQueueManager) TransactionReturnHandler() func(queuedTx *status.QueuedTx, err error) {
	return func(queuedTx *status.QueuedTx, err error) {
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
