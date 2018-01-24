package transactions

import (
	"strconv"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/status-im/status-go/geth/common"
	"github.com/status-im/status-go/geth/signal"
	"github.com/status-im/status-go/geth/transactions/queue"
)

const (
	// EventTransactionQueued is triggered when send transaction request is queued
	EventTransactionQueued = "transaction.queued"
	// EventTransactionFailed is triggered when send transaction request fails
	EventTransactionFailed = "transaction.failed"
)

const (
	// SendTransactionNoErrorCode is sent when no error occurred.
	SendTransactionNoErrorCode = iota
	// SendTransactionDefaultErrorCode is every case when there is no special tx return code.
	SendTransactionDefaultErrorCode
	// SendTransactionPasswordErrorCode is sent when account failed verification.
	SendTransactionPasswordErrorCode
	// SendTransactionTimeoutErrorCode is sent when tx is timed out.
	SendTransactionTimeoutErrorCode
	// SendTransactionDiscardedErrorCode is sent when tx was discarded.
	SendTransactionDiscardedErrorCode
)

var txReturnCodes = map[error]int{
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
	// discard notifications with empty tx
	if queuedTx == nil {
		return
	}
	// we don't want to notify a user if tx sent successfully
	if queuedTx.Err == nil {
		return
	}
	signal.Send(signal.Envelope{
		Type: EventTransactionFailed,
		Event: ReturnSendTransactionEvent{
			ID:           string(queuedTx.ID),
			Args:         queuedTx.Args,
			MessageID:    common.MessageIDFromContext(queuedTx.Context),
			ErrorMessage: queuedTx.Err.Error(),
			ErrorCode:    strconv.Itoa(sendTransactionErrorCode(queuedTx.Err)),
		},
	})
}

func sendTransactionErrorCode(err error) int {
	if code, ok := txReturnCodes[err]; ok {
		return code
	}
	return SendTxDefaultErrorCode
}
