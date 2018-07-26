package sign

import (
	"errors"
)

var (
	//ErrSignReqNotFound - error sign request hash not found
	ErrSignReqNotFound = errors.New("sign request not found")
	//ErrSignReqInProgress - error sign request is in progress
	ErrSignReqInProgress = errors.New("sign request is in progress")
	//ErrSignReqTimedOut - error sign request sending timed out
	ErrSignReqTimedOut = errors.New("sign request sending timed out")
	//ErrSignReqDiscarded - error sign request discarded
	ErrSignReqDiscarded = errors.New("sign request has been discarded")
)

// TransientError means that the sign request won't be removed from the list of
// pending if it happens. There are a few built-in transient errors, and this
// struct can be used to wrap any error to be transient.
type TransientError struct {
	Reason error
}

// Error returns the string representation of the underlying error.
func (e TransientError) Error() string {
	return e.Reason.Error()
}

// NewTransientError wraps an error into a TransientError structure.
func NewTransientError(reason error) TransientError {
	return TransientError{reason}
}
