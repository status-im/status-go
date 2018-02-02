package transactions

import "errors"

var (
	//ErrQueuedTxTimedOut - error transaction sending timed out
	ErrQueuedTxTimedOut = errors.New("transaction sending timed out")
	//ErrQueuedTxDiscarded - error transaction discarded
	ErrQueuedTxDiscarded = errors.New("transaction has been discarded")
)
