package status

import (
	"github.com/teslapatrick/go-ethereum/accounts"
	"github.com/teslapatrick/go-ethereum/common"
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

type AccountsFilterHandler func([]common.Address) []common.Address

// Accounts returns accounts' addresses of currently logged in user.
// Since status supports HD keys, the following list is returned:
// [addressCDK#1, addressCKD#2->Child1, addressCKD#2->Child2, .. addressCKD#2->ChildN]
func (d *AccountManager) Accounts() []common.Address {
	var addresses []common.Address
	for _, wallet := range d.am.Wallets() {
		for _, account := range wallet.Accounts() {
			addresses = append(addresses, account.Address)
		}
	}

	if d.accountsFilterHandler != nil {
		return d.accountsFilterHandler(addresses)
	}

	return addresses
}

func (d *AccountManager) SetAccountsFilterHandler(fn AccountsFilterHandler) {
	d.accountsFilterHandler = fn
}
