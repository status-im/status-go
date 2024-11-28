package activity

import "errors"

var (
	ErrNoAddressesProvided = errors.New("no addresses provided")
	ErrNoChainIDsProvided  = errors.New("no chainIDs provided")
	ErrSessionNotFound     = errors.New("session not found")
)
