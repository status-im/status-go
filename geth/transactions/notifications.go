package transactions

import (
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/status-im/status-go/geth/signal"
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
	nil:                  SendTransactionNoErrorCode,
	keystore.ErrDecrypt:  SendTransactionPasswordErrorCode,
	ErrQueuedTxTimedOut:  SendTransactionTimeoutErrorCode,
	ErrQueuedTxDiscarded: SendTransactionDiscardedErrorCode,
}

// SendTransactionEvent is a signal sent on a send transaction request
type SendTransactionEvent struct {
	ID        string     `json:"id"`
	Args      SendTxArgs `json:"args"`
	MessageID string     `json:"message_id"`
}

// NotifyOnEnqueue returns handler that processes incoming tx queue requests
func NotifyOnEnqueue(queuedTx *QueuedTx) {
	signal.Send(signal.Envelope{
		Type: EventTransactionQueued,
		Event: SendTransactionEvent{
			ID:        queuedTx.ID,
			Args:      queuedTx.Args,
			MessageID: messageIDFromContext(queuedTx.Context),
		},
	})
}

// ReturnSendTransactionEvent is a JSON returned whenever transaction send is returned
type ReturnSendTransactionEvent struct {
	ID           string     `json:"id"`
	Args         SendTxArgs `json:"args"`
	MessageID    string     `json:"message_id"`
	ErrorMessage string     `json:"error_message"`
	ErrorCode    int        `json:"error_code,string"`
}

// NotifyOnReturn returns handler that processes responses from internal tx manager
func NotifyOnReturn(queuedTx *QueuedTx, err error) {
	// we don't want to notify a user if tx was sent successfully
	if err == nil {
		return
	}
	// discard notifications with empty tx
	if queuedTx == nil {
		return
	}
	signal.Send(signal.Envelope{
		Type: EventTransactionFailed,
		Event: ReturnSendTransactionEvent{
			ID:           queuedTx.ID,
			Args:         queuedTx.Args,
			MessageID:    messageIDFromContext(queuedTx.Context),
			ErrorMessage: err.Error(),
			ErrorCode:    sendTransactionErrorCode(err),
		},
	})
}

func sendTransactionErrorCode(err error) int {
	if code, ok := txReturnCodes[err]; ok {
		return code
	}
	return SendTxDefaultErrorCode
}
