package status

import (
	"github.com/ethereum/go-ethereum/accounts"
)

type AccountManager struct {
	am                    *accounts.Manager
	accountsFilterHandler AccountsFilterHandler
}

// NewAccountManager creates a new AccountManager
func NewAccountManager(am *accounts.Manager) *AccountManager {
	return &AccountManager{
		am: am,
	}
}

type AccountsFilterHandler func([]accounts.Account) []accounts.Account

// Accounts returns accounts of currently logged in user.
// Since status supports HD keys, the following list is returned:
// [addressCDK#1, addressCKD#2->Child1, addressCKD#2->Child2, .. addressCKD#2->ChildN]
func (d *AccountManager) Accounts() []accounts.Account {
	accounts := d.am.Accounts()
	if d.accountsFilterHandler != nil {
		accounts = d.accountsFilterHandler(accounts)
	}

	return accounts
}

func (d *AccountManager) SetAccountsFilterHandler(fn AccountsFilterHandler) {
	d.accountsFilterHandler = fn
}
