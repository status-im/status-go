package accounts

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/status-im/status-go/multiaccounts"
)

var (
	// ErrUpdatingWrongAccount raised if caller tries to update any other account except one used for login.
	ErrUpdatingWrongAccount = errors.New("failed to updating wrong account. please login with that account first")
)

func NewMultiAccountsAPI(db *multiaccounts.Database, address common.Address) *MultiAccountsAPI {
	return &MultiAccountsAPI{db: db, login: address}
}

// MultiAccountsAPI is class with methods available over RPC.
type MultiAccountsAPI struct {
	db    *multiaccounts.Database
	login common.Address
}

func (api *MultiAccountsAPI) UpdateAccount(account multiaccounts.Account) error {
	if account.Address != api.login {
		return ErrUpdatingWrongAccount
	}
	return api.db.UpdateAccount(account)
}
