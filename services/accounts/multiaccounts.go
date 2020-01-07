package accounts

import (
	"errors"

	"github.com/status-im/status-go/account"
	"github.com/status-im/status-go/multiaccounts"
)

var (
	// ErrUpdatingWrongAccount raised if caller tries to update any other account except one used for login.
	ErrUpdatingWrongAccount = errors.New("failed to update wrong account. Please login with that account first")
)

func NewMultiAccountsAPI(db *multiaccounts.Database, manager *account.Manager) *MultiAccountsAPI {
	return &MultiAccountsAPI{db: db, manager: manager}
}

// MultiAccountsAPI is class with methods available over RPC.
type MultiAccountsAPI struct {
	db      *multiaccounts.Database
	manager *account.Manager
}

func (api *MultiAccountsAPI) UpdateAccount(account multiaccounts.Account) error {
	return api.db.UpdateAccount(account)
}
