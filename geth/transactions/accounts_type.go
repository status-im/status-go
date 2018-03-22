package transactions

import (
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/status-im/status-go/geth/account"
)

// Accounter defines expected methods for managing Status accounts.
type Accounter interface {
	// SelectedAccount returns currently selected account
	SelectedAccount() (*account.SelectedExtKey, error)

	// VerifyAccountPassword tries to decrypt a given account key file, with a provided password.
	// If no error is returned, then account is considered verified.
	VerifyAccountPassword(keyStoreDir, address, password string) (*keystore.Key, error)
}
