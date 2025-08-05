// Package accounts provides account management functionality.
// It is split into subpackages:
//   - core: main account manager implementation
//   - keystore: cryptographic key storage operations
//   - persistence: database operations for key pairs and accounts
//   - generator: account creation and derivation
//   - types: type definitions
//   - common: shared utilities
package accountsmanagement

import (
	"github.com/status-im/status-go/accounts-management/core"
	"github.com/status-im/status-go/accounts-management/keystore"
	"github.com/status-im/status-go/accounts-management/persistence"
	"github.com/status-im/status-go/accounts-management/types"

	"go.uber.org/zap"
)

// Re-export main types and interfaces for backward compatibility
type AccountsManager = core.AccountsManager
type KeyStore = keystore.KeyStore
type Persistence = persistence.Persistence

// Re-export error types
var (
	ErrLoggerIsMissing                        = core.ErrLoggerIsMissing
	ErrAccountKeyStoreMissing                 = core.ErrAccountKeyStoreMissing
	ErrNoAccountSelected                      = core.ErrNoAccountSelected
	ErrPersistenceIsMissing                   = core.ErrPersistenceIsMissing
	ErrAccountDoesNotExist                    = core.ErrAccountDoesNotExist
	ErrAddressAndPasswordOrPrivateKeyRequired = core.ErrAddressAndPasswordOrPrivateKeyRequired
	ErrAccountIsNil                           = core.ErrAccountIsNil
	ErrKeypairIsNil                           = core.ErrKeypairIsNil
	ErrCannotRemoveChatAccount                = core.ErrCannotRemoveChatAccount
	ErrCannotRemoveDefaultWalletAccount       = core.ErrCannotRemoveDefaultWalletAccount
	ErrCannotRemoveProfileKeypair             = core.ErrCannotRemoveProfileKeypair
)

// NewAccountsManager creates a new accounts manager instance
func NewAccountsManager(logger *zap.Logger) (*AccountsManager, error) {
	return core.NewAccountsManager(logger)
}

// Re-export types for convenience
type (
	Account                = types.Account
	Keypair                = types.Keypair
	Keycard                = types.Keycard
	KeystoreAccount        = types.KeystoreAccount
	AccountCreationDetails = types.AccountCreationDetails
	KeypairType            = types.KeypairType
	AccountType            = types.AccountType
	AccountOperable        = types.AccountOperable
)
