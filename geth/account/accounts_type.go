package account

import (
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
)

// Accounter defines expected methods for managing Status accounts
type Accounter interface {
	// CreateAccount creates an internal geth account
	// BIP44-compatible keys are generated: CKD#1 is stored as account key, CKD#2 stored as sub-account root
	// Public key of CKD#1 is returned, with CKD#2 securely encoded into account key file (to be used for
	// sub-account derivations)
	CreateAccount(password string) (address, pubKey, mnemonic string, err error)

	// CreateChildAccount creates sub-account for an account identified by parent address.
	// CKD#2 is used as root for master accounts (when parentAddress is "").
	// Otherwise (when parentAddress != ""), child is derived directly from parent.
	CreateChildAccount(parentAddress, password string) (address, pubKey string, err error)

	// RecoverAccount re-creates master key using given details.
	// Once master key is re-generated, it is inserted into keystore (if not already there).
	RecoverAccount(password, mnemonic string) (address, pubKey string, err error)

	// VerifyAccountPassword tries to decrypt a given account key file, with a provided password.
	// If no error is returned, then account is considered verified.
	VerifyAccountPassword(keyStoreDir, address, password string) (*keystore.Key, error)

	// SelectAccount selects current account, by verifying that address has corresponding account which can be decrypted
	// using provided password. Once verification is done, decrypted key is injected into Whisper (as a single identity,
	// all previous identities are removed).
	SelectAccount(address, password string) error

	// SelectedAccount returns currently selected account
	SelectedAccount() (*SelectedExtKey, error)

	// Logout clears whisper identities
	Logout() error

	// Accounts returns handler to process account list request
	Accounts() ([]common.Address, error)

	// AddressToDecryptedAccount tries to load decrypted key for a given account.
	// The running node, has a keystore directory which is loaded on start. Key file
	// for a given address is expected to be in that directory prior to node start.
	AddressToDecryptedAccount(address, password string) (accounts.Account, *keystore.Key, error)
}
