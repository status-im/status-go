package sign

import (
	"context"

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

const (
	// MessageIDKey is a key for message ID
	// This ID is required to track from which chat a given send transaction request is coming.
	MessageIDKey = contextKey("message_id")
)

type contextKey string // in order to make sure that our context key does not collide with keys from other packages

// messageIDFromContext returns message id from context (if exists)
func messageIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if messageID, ok := ctx.Value(MessageIDKey).(string); ok {
		return messageID
	}

	return ""
}

var txReturnCodes = map[error]int{
	nil:                 SendTransactionNoErrorCode,
	keystore.ErrDecrypt: SendTransactionPasswordErrorCode,
	ErrSignReqTimedOut:  SendTransactionTimeoutErrorCode,
	ErrSignReqDiscarded: SendTransactionDiscardedErrorCode,
}

// SendTransactionEvent is a signal sent on a send transaction request
type SendTransactionEvent struct {
	ID        string      `json:"id"`
	Args      interface{} `json:"args"`
	MessageID string      `json:"message_id"`
}

// NotifyOnEnqueue returns handler that processes incoming tx queue requests
func NotifyOnEnqueue(request *Request) {
	signal.Send(signal.Envelope{
		Type: EventTransactionQueued,
		Event: SendTransactionEvent{
			ID:        request.ID,
			Args:      request.Meta,
			MessageID: messageIDFromContext(request.context),
		},
	})
}

// ReturnSendTransactionEvent is a JSON returned whenever transaction send is returned
type ReturnSendTransactionEvent struct {
	ID           string      `json:"id"`
	Args         interface{} `json:"args"`
	MessageID    string      `json:"message_id"`
	ErrorMessage string      `json:"error_message"`
	ErrorCode    int         `json:"error_code,string"`
}

// NotifyOnReturn returns handler that processes responses from internal tx manager
func NotifyOnReturn(request *Request, err error) {
	// we don't want to notify a user if tx was sent successfully
	if err == nil {
		return
	}
	signal.Send(signal.Envelope{
		Type: EventTransactionFailed,
		Event: ReturnSendTransactionEvent{
			ID:           request.ID,
			Args:         request.Meta,
			MessageID:    messageIDFromContext(request.context),
			ErrorMessage: err.Error(),
			ErrorCode:    sendTransactionErrorCode(err),
		},
	})
}

func sendTransactionErrorCode(err error) int {
	if code, ok := txReturnCodes[err]; ok {
		return code
	}
	return SendTransactionDefaultErrorCode
}
