package transactions

import (
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/status-im/status-go/geth/common"
	"github.com/status-im/status-go/geth/log"
	"github.com/status-im/status-go/geth/signal"
	"github.com/status-im/status-go/geth/transactions/queue"
)

const (
	// EventTransactionQueued is triggered when send transaction request is queued
	EventTransactionQueued = "transaction.queued"
	// EventTransactionFailed is triggered when send transaction request fails
	EventTransactionFailed = "transaction.failed"

	SendTransactionNoErrorCode        = "0"
	SendTransactionDefaultErrorCode   = "1"
	SendTransactionPasswordErrorCode  = "2"
	SendTransactionTimeoutErrorCode   = "3"
	SendTransactionDiscardedErrorCode = "4"
)

var txReturnCodes = map[error]string{ // deliberately strings, in case more meaningful codes are to be returned
	nil:                        SendTransactionNoErrorCode,
	keystore.ErrDecrypt:        SendTransactionPasswordErrorCode,
	queue.ErrQueuedTxTimedOut:  SendTransactionTimeoutErrorCode,
	queue.ErrQueuedTxDiscarded: SendTransactionDiscardedErrorCode,
}

// SendTransactionEvent is a signal sent on a send transaction request
type SendTransactionEvent struct {
	ID        string            `json:"id"`
	Args      common.SendTxArgs `json:"args"`
	MessageID string            `json:"message_id"`
}

// NotifyOnEnqueue returns handler that processes incoming tx queue requests
func NotifyOnEnqueue(queuedTx *common.QueuedTx) {
	log.Info("calling TransactionQueueHandler")
	signal.Send(signal.Envelope{
		Type: EventTransactionQueued,
		Event: SendTransactionEvent{
			ID:        string(queuedTx.ID),
			Args:      queuedTx.Args,
			MessageID: common.MessageIDFromContext(queuedTx.Context),
		},
	})
}

// ReturnSendTransactionEvent is a JSON returned whenever transaction send is returned
type ReturnSendTransactionEvent struct {
	ID           string            `json:"id"`
	Args         common.SendTxArgs `json:"args"`
	MessageID    string            `json:"message_id"`
	ErrorMessage string            `json:"error_message"`
	ErrorCode    string            `json:"error_code"`
}

// NotifyOnReturn returns handler that processes responses from internal tx manager
func NotifyOnReturn(queuedTx *common.QueuedTx) {
	if queuedTx.Err == nil {
		return
	}

	// discard notifications with empty tx
	if queuedTx == nil {
		return
	}

	// error occurred, signal up to application
	signal.Send(signal.Envelope{
		Type: EventTransactionFailed,
		Event: ReturnSendTransactionEvent{
			ID:           string(queuedTx.ID),
			Args:         queuedTx.Args,
			MessageID:    common.MessageIDFromContext(queuedTx.Context),
			ErrorMessage: queuedTx.Err.Error(),
			ErrorCode:    sendTransactionErrorCode(queuedTx.Err),
		},
	})
}

func sendTransactionErrorCode(err error) string {
	if code, ok := txReturnCodes[err]; ok {
		return code
	}
	return SendTxDefaultErrorCode
}
