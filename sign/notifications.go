package sign

import (
	"context"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/status-im/status-go/signal"
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

// SendSignRequestAdded sends a signal when a sign request is added.
func SendSignRequestAdded(request *Request) {
	signal.SendSignRequestAdded(
		signal.PendingRequestEvent{
			ID:        request.ID,
			Args:      request.Meta,
			Method:    request.Method,
			MessageID: messageIDFromContext(request.context),
		})
}

// SendSignRequestFailed sends a signal only if error had happened
func SendSignRequestFailed(request *Request, err error) {
	signal.SendSignRequestFailed(
		signal.PendingRequestEvent{
			ID:        request.ID,
			Args:      request.Meta,
			Method:    request.Method,
			MessageID: messageIDFromContext(request.context),
		},
		err, sendTransactionErrorCode(err))
}

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

var txReturnCodes = map[error]int{
	nil:                 SignRequestNoErrorCode,
	keystore.ErrDecrypt: SignRequestPasswordErrorCode,
	ErrSignReqTimedOut:  SignRequestTimeoutErrorCode,
	ErrSignReqDiscarded: SignRequestDiscardedErrorCode,
}

func sendTransactionErrorCode(err error) int {
	if code, ok := txReturnCodes[err]; ok {
		return code
	}
	return SignRequestDefaultErrorCode
}
