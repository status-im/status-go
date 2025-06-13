package account

import (
	"errors"
)

var (
	ErrNoAccountSelected      = errors.New("no account has been selected, please login")
	ErrAccountKeyStoreMissing = errors.New("account key store is not set")
)
