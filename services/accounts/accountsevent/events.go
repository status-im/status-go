package accountsevent

import (
	"github.com/ethereum/go-ethereum/common"
)

// AccountsAddedEvent is emitted when a new account is added.
type AccountsAddedEvent struct {
	Accounts []common.Address
}

// AccountsRemovedEvent is emitted when a new account is added.
type AccountsRemovedEvent struct {
	Accounts []common.Address
}
