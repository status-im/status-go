package accountsmanagement

import (
	"errors"
)

var (
	ErrPersistenceIsMissing   = errors.New("persistence is not set")
	ErrLoggerIsMissing        = errors.New("logger is not set")
	ErrAccountDoesNotExist    = errors.New("account doesn't exist")
	ErrNoAccountSelected      = errors.New("no account has been selected, please login")
	ErrAccountKeyStoreMissing = errors.New("account key store is not set")
	ErrAccountIsNil           = errors.New("account is nil")
)
