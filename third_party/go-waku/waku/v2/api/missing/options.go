package missing

import (
	"time"

	"github.com/waku-org/go-waku/waku/v2/api/common"
)

type missingMessageVerifierParams struct {
	delay                        time.Duration
	interval                     time.Duration
	maxAttemptsToRetrieveHistory int
	storeQueryTimeout            time.Duration
}

// MissingMessageVerifierOption is an option that can be used to customize the MissingMessageVerifier behavior
type MissingMessageVerifierOption func(*missingMessageVerifierParams)

// WithVerificationInterval is an option used to setup the verification interval
func WithVerificationInterval(t time.Duration) MissingMessageVerifierOption {
	return func(params *missingMessageVerifierParams) {
		params.interval = t
	}
}

// WithDelay is an option used to indicate the delay to apply for verifying messages
func WithDelay(t time.Duration) MissingMessageVerifierOption {
	return func(params *missingMessageVerifierParams) {
		params.delay = t
	}
}

// WithMaxAttempts indicates how many times will the message verifier retry a failed storenode request
func WithMaxRetryAttempts(max int) MissingMessageVerifierOption {
	return func(params *missingMessageVerifierParams) {
		params.maxAttemptsToRetrieveHistory = max
	}
}

// WithStoreQueryTimeout sets the timeout for store query
func WithStoreQueryTimeout(timeout time.Duration) MissingMessageVerifierOption {
	return func(params *missingMessageVerifierParams) {
		params.storeQueryTimeout = timeout
	}
}

var defaultMissingMessagesVerifierOptions = []MissingMessageVerifierOption{
	WithVerificationInterval(time.Minute),
	WithDelay(20 * time.Second),
	WithMaxRetryAttempts(3),
	WithStoreQueryTimeout(common.DefaultStoreQueryTimeout),
}
