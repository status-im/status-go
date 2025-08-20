// Package accounts provides account management functionality.
// It is split into subpackages:
//   - core: main account manager implementation
//   - errors: structured error handling system
//   - keystore: cryptographic key storage operations
//   - persistence: database operations for key pairs and accounts
//   - generator: account creation and derivation
//   - types: type definitions
//   - common: shared utilities
package accountsmanagement

import (
	"github.com/status-im/status-go/accounts-management/core"
	"github.com/status-im/status-go/accounts-management/keystore"
	keystoretypes "github.com/status-im/status-go/accounts-management/keystore/types"
	"github.com/status-im/status-go/accounts-management/persistence"
	"github.com/status-im/status-go/accounts-management/types"

	"go.uber.org/zap"
)

// Re-export main types and interfaces for backward compatibility
type AccountsManager = core.AccountsManager
type KeyStore = keystore.KeyStore
type Persistence = persistence.Persistence

// Re-export errors for convenience
var (
	ErrNoAccountSelected   = core.ErrNoAccountSelected
	ErrKeystoreFileMissing = core.ErrKeystoreFileMissing
	ErrAccountDoesNotExist = core.ErrAccountDoesNotExist
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
	KeystoreAccount        = keystoretypes.KeystoreAccount
	AccountCreationDetails = types.AccountCreationDetails
	KeypairType            = types.KeypairType
	AccountType            = types.AccountType
	AccountOperable        = types.AccountOperable
)
