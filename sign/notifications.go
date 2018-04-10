package sign

import (
	"context"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/status-im/status-go/geth/signal"
)

const (
	// EventSignRequestAdded is triggered when send transaction request is queued
	EventSignRequestAdded = "sign-request.queued"
	// EventSignRequestFailed is triggered when send transaction request fails
	EventSignRequestFailed = "sign-request.failed"
)

const (
	// SignRequestNoErrorCode is sent when no error occurred.
	SignRequestNoErrorCode = iota
	// SignRequestDefaultErrorCode is every case when there is no special tx return code.
	SignRequestDefaultErrorCode
	// SignRequestPasswordErrorCode is sent when account failed verification.
	SignRequestPasswordErrorCode
	// SignRequestTimeoutErrorCode is sent when tx is timed out.
	SignRequestTimeoutErrorCode
	// SignRequestDiscardedErrorCode is sent when tx was discarded.
	SignRequestDiscardedErrorCode
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
	nil:                 SignRequestNoErrorCode,
	keystore.ErrDecrypt: SignRequestPasswordErrorCode,
	ErrSignReqTimedOut:  SignRequestTimeoutErrorCode,
	ErrSignReqDiscarded: SignRequestDiscardedErrorCode,
}

// PendingRequestEvent is a signal sent when a sign request is added
type PendingRequestEvent struct {
	ID        string      `json:"id"`
	Method    string      `json:"method"`
	Args      interface{} `json:"args"`
	MessageID string      `json:"message_id"`
}

// NotifyOnEnqueue sends a signal when a sign request is added
func NotifyOnEnqueue(request *Request) {
	signal.Send(signal.Envelope{
		Type: EventSignRequestAdded,
		Event: PendingRequestEvent{
			ID:        request.ID,
			Args:      request.Meta,
			Method:    request.Method,
			MessageID: messageIDFromContext(request.context),
		},
	})
}

// PendingRequestErrorEvent is a signal sent when sign request has failed
type PendingRequestErrorEvent struct {
	PendingRequestEvent
	ErrorMessage string `json:"error_message"`
	ErrorCode    int    `json:"error_code,string"`
}

// NotifyIfError sends a signal only if error had happened
func NotifyIfError(request *Request, err error) {
	// we don't want to notify a user if tx was sent successfully
	if err == nil {
		return
	}
	signal.Send(signal.Envelope{
		Type: EventSignRequestFailed,
		Event: PendingRequestErrorEvent{
			PendingRequestEvent: PendingRequestEvent{
				ID:        request.ID,
				Args:      request.Meta,
				Method:    request.Method,
				MessageID: messageIDFromContext(request.context),
			},
			ErrorMessage: err.Error(),
			ErrorCode:    sendTransactionErrorCode(err),
		},
	})
}

func sendTransactionErrorCode(err error) int {
	if code, ok := txReturnCodes[err]; ok {
		return code
	}
	return SignRequestDefaultErrorCode
}
