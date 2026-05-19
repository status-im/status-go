# Accounts Management Package

The Accounts Management package provides a comprehensive solution for Status chat and wallet accounts creation, storage, and management within the Status application. It handles account generation, keystore operations, and secure storage of private keys. For cryptographic operations, see the separate `crypto` package.

## Package Structure

The package is organized into focused subpackages:

```
accounts-management/
├── manager.go                 # AccountsManager and core business logic
├── keystore_operations.go     # Keystore-related operations
├── persistence_operations.go  # Database operations
├── interface_keystore.go      # KeyStore interface definition
├── interface_persistence.go   # Persistence interface definition
├── manager_test.go            # Test suite for core functionality
├── errors.go                  # Top-level error definitions for the package
├── errors/                    # Structured error type and helpers
│   ├── errors.go              # AccountsError, ErrorCode, ErrorCategory
│   └── *_test.go              # Error tests
├── keystore/                  # Cryptographic key storage operations
│   ├── account.go             # KeystoreAccount type and helpers
│   ├── adapter.go             # KeyStore adapter wiring
│   ├── helper.go              # Helper functions
│   ├── errors.go              # Keystore-specific errors
│   └── internal/geth/         # Geth-compatible keystore implementation
│       ├── encryption.go      # Encryption operations
│       ├── decryption.go      # Decryption operations
│       ├── reencryption.go    # Re-encryption operations
│       ├── migration.go       # Migration utilities
│       └── const.go           # Constants
├── generator/                 # Account creation and derivation
│   ├── generator.go           # Account generation and derivation functions
│   ├── types.go               # Account types and methods
│   ├── path_decoder.go        # BIP32 path parsing
│   ├── errors.go              # Generator-specific errors
│   ├── README.md              # Generator-specific documentation
│   └── *_test.go              # Generator tests
├── types/                     # Core type definitions
│   ├── types.go               # Shared types (PublicKeyData, SelectedExtKey)
│   ├── key.go                 # Key type structure
│   ├── keypair.go             # Keypair, Account, Keycard, AccountCreationDetails, enums
│   └── *_test.go              # Type tests
├── common/                    # Shared utilities
│   ├── const.go               # Constants and BIP32/BIP44 paths
│   ├── mnemonic.go            # Mnemonic generation utilities
│   ├── address.go             # Address utilities
│   ├── publickey.go           # Public key utilities
│   ├── hdkey.go               # HD key helpers
│   ├── utils.go               # General utilities
│   └── *_test.go              # Common tests
├── mock/                      # Mock implementations for testing
│   └── persistence.go         # Mock persistence implementation
└── testdata/                  # Fixture data for tests
```

## Key Components

### Main Package

The main account management operations are directly in the root package:

- **AccountsManager**: Main interface for account management operations including account creation, selection, and verification
- **Keystore Operations**: Methods for keystore management and account loading
- **Persistence Operations**: Methods for database operations including keypair and account management
- **Interface Definitions**: KeyStore and Persistence interfaces for implementing custom backends

### Errors Package

The `errors/` subpackage defines the structured error machinery; the package-root `errors.go` declares the concrete error variables and codes used across `accounts-management`:

- **Structured Errors**: `AccountsError` with code, category, message, wrapped error, and context map
- **Error Codes**: Numeric `ErrorCode` constants for programmatic error handling
- **Error Categories**: `ErrorCategory` values (`account`, `keystore`, `validation`, `database`, `system`, `unknown`)
- **Context Support**: `(*AccountsError).WithContext(key, value)` adds key-value metadata
- **Helper Constructors**: `errors.NewError(...)` and `errors.WrapError(...)` for creation and wrapping

### Keystore Package

Handles secure cryptographic key storage:

- **KeyStore Interface**: Defined in `interface_keystore.go` (root package) for keystore implementations
- **`keystore/` package**: `account.go` defines `KeystoreAccount`; `adapter.go` wires the geth-backed implementation; `helper.go` provides helpers; `errors.go` declares keystore-specific errors
- **Geth Implementation**: `keystore/internal/geth/` — geth-compatible adapter with encryption, decryption, re-encryption, and migration

### Persistence Package

Manages account data storage:

- **Persistence Interface**: Defines the contract for data persistence operations including keypair, account, and keycard management
- **Database Operations**: Account, keypair, and keycard management with comprehensive CRUD operations

### Generator Package

Account creation and derivation logic:

- **Account Creation**: From mnemonics, private keys, and extended keys
- **Account Derivation**: BIP32/BIP44 path-based derivation with support for single and multiple derivations
- **Path Decoding**: BIP32 path parsing and validation
- **Batch Operations**: Support for creating multiple accounts from a single mnemonic

### Types Package

Core data structures:

- **Account**: Represents an Ethereum account with address, keyUID, wallet/chat flags, and metadata
- **Keypair**: Represents a collection of related accounts with type, derivation info, and keycard support
- **Keycard**: Represents a hardware keycard with associated accounts
- **KeystoreAccount**: Represents a keystore entry with address and URL
- **AccountCreationDetails**: Contains account creation information including path and name
- **AccountType/KeypairType**: Type enumerations for accounts and keypairs
- **AccountOperable**: Enumeration for account operability status (fully, partially, non-operable)



### Common Package

Shared utilities and constants:

- **Constants**: BIP32/BIP44 path constants and other shared constants
- **Mnemonic Utilities**: Mnemonic generation and validation
- **Address Utilities**: Address validation and formatting
- **Public Key Utilities**: Public key operations and conversions
- **General Utilities**: Common helper functions

## Usage

### Basic Usage

```go
package main

import (
    "log"

    "github.com/status-im/status-go/internal/accounts-management"
    "go.uber.org/zap"
)

func main() {
    // Initialize logger
    logger, _ := zap.NewDevelopment()

    // Create account manager
    manager, err := accountsmanagement.NewAccountsManager(logger)
    if err != nil {
        log.Fatal(err)
    }

    // Set up persistence and root data directory
    persistence := yourPersistenceImplementation()
    manager.SetPersistence(persistence)

    // Set the root data directory for keystore management
    manager.SetRootDataDir("/path/to/root/data/dir")

    // Create a new keypair from mnemonic
    walletAccount := &types.AccountCreationDetails{
        Path: "m/44'/60'/0'/0/0",
        Name: "Wallet Account",
    }

    keypair, err := manager.CreateKeypairFromMnemonicAndStore(
        mnemonic,
        "my-password",
        "My Keypair",
        types.ColdWalletTypeNone, // cold wallet type (none for hot wallets)
        walletAccount,
        true, // profile keypair
        0,    // clock
    )
    if err != nil {
        log.Fatal(err)
    }

    log.Printf("Created keypair: %s", keypair.Name)
    log.Printf("Keypair UID: %s", keypair.KeyUID)

    // Select the account (this will also switch the keystore internally)
    err = manager.SetChatAccount(account.Address(), "my-password", nil)
    if err != nil {
        log.Fatal(err)
    }

    // Get selected account
    selectedAccount, err := manager.SelectedChatAccount()
    if err != nil {
        log.Fatal(err)
    }

    log.Printf("Selected chat account: %s", selectedAccount.Address().Hex())
}
```

### Advanced Usage

For more advanced usage, you can import specific subpackages:

```go
import (
    "github.com/status-im/status-go/internal/accounts-management"
    "github.com/status-im/status-go/internal/accounts-management/generator"
    "github.com/status-im/status-go/internal/accounts-management/types"
    "github.com/status-im/status-go/internal/accounts-management/errors"
)

// Use specific types and functions
account, err := generator.CreateAccountFromMnemonic(mnemonic, passphrase)
keypair := &types.Keypair{ /* ... */ }

// Use structured error handling
if err != nil {
    var accountsErr *errors.AccountsError
    if errors.As(err, &accountsErr) {
        switch accountsErr.Category {
        case errors.ErrorCategoryAccount:
            log.Printf("Account error: %s", accountsErr.Message)
        case errors.ErrorCategoryValidation:
            log.Printf("Validation error: %s", accountsErr.Message)
        }
    }
}
```



### Error Handling

The errors package provides a structured error handling system:

```go
import (
    accmgmterrors "github.com/status-im/status-go/internal/accounts-management/errors"
    accmgmt "github.com/status-im/status-go/internal/accounts-management"
)

// Use a pre-declared package error
err := accmgmt.ErrKeypairAlreadyAdded.WithContext("keyuid", keyUID)

// Wrap an existing error
wrappedErr := accmgmterrors.WrapError(accmgmt.ErrCodeWrongPasswordProvided, "wrong password", originalErr, nil)

// Pattern-match on category
var accountsErr *accmgmterrors.AccountsError
if errors.As(err, &accountsErr) {
    switch accountsErr.Category {
    case accmgmterrors.ErrorCategoryValidation:
        // ...
    case accmgmterrors.ErrorCategoryDatabase:
        // ...
    }
}
```

### Account Generation

The generator package provides flexible account creation:

```go
import "github.com/status-im/status-go/internal/accounts-management/generator"

// Create account from mnemonic
account, err := generator.CreateAccountFromMnemonic(mnemonic, passphrase)

// Create account from private key
account, err := generator.CreateAccountFromPrivateKey(privateKeyHex)

// Derive child account
childAccount, err := generator.DeriveChildFromAccount(account, "m/44'/60'/0'/0/1")

// Derive multiple children in one call
children, err := generator.DeriveChildrenFromAccount(account, []string{
    "m/44'/60'/0'/0/0",
    "m/44'/60'/0'/0/1",
})

// Create a single account and derive multiple paths from a mnemonic
masterAcc, derived, err := generator.CreateAndDeriveAccountsFromMnemonic(mnemonic, paths, passphrase)

// Create multiple new accounts (each backed by its own freshly generated mnemonic)
accounts, mnemonics, err := generator.CreateAccountsOfMnemonicLength(12, 5, passphrase)
```

### Keypair Creation

The main package provides keypair creation methods:

```go
// Create keypair from mnemonic
walletAccount := &types.AccountCreationDetails{
    Path: "m/44'/60'/0'/0/0",
    Name: "Wallet Account",
}
keypair, err := manager.CreateKeypairFromMnemonicAndStore(
    mnemonic,
    password,
    "My Keypair",
    types.ColdWalletTypeNone, // cold wallet type
    walletAccount,
    true, // profile keypair
    clock,
)

// Create keypair from private key
keypair, err := manager.CreateKeypairFromPrivateKeyAndStore(
    privateKeyHex,
    password,
    "Private Key Keypair",
    walletAccount,
    clock,
)
```

## Main Types and Interfaces

### AccountsManager

The main interface for account management:

```go
// Selected methods on AccountsManager (see manager.go, keystore_operations.go, persistence_operations.go for the full surface):
func (m *AccountsManager) CreateKeypairFromMnemonicAndStore(mnemonic, password, keypairName string,
    coldWallet types.ColdWalletType, walletAccount *types.AccountCreationDetails, profile bool, clock uint64) (*types.Keypair, error)
func (m *AccountsManager) CreateKeypairFromPrivateKeyAndStore(privateKey, password, keypairName string,
    walletAccount *types.AccountCreationDetails, clock uint64) (*types.Keypair, error)
func (m *AccountsManager) AddKeypairStoredToColdWallet(keyUID, masterAddress, name, walletXPub string,
    coldWallet types.ColdWalletType, walletAccounts []*types.Account, clock uint64) (*types.Keypair, error)
func (m *AccountsManager) SetChatAccount(address cryptotypes.Address, password string, privateKey *ecdsa.PrivateKey) error
func (m *AccountsManager) SelectedChatAccount() (*generator.Account, error)
func (m *AccountsManager) LoadAccount(address cryptotypes.Address, password string) (*generator.Account, error)
func (m *AccountsManager) VerifyAccountPassword(address cryptotypes.Address, password string) (bool, error)
func (m *AccountsManager) GetVerifiedWalletAccount(address cryptotypes.Address, password string) (*generator.Account, error)
func (m *AccountsManager) Logout()
func (m *AccountsManager) Accounts() ([]cryptotypes.Address, error)
```


### KeyStore Interface

Defines keystore operations:

```go
type KeyStore interface {
    ImportECDSA(priv *ecdsa.PrivateKey, passphrase string) (keystore.KeystoreAccount, error)
    ImportSingleExtendedKey(extKey *extkeys.ExtendedKey, passphrase string) (keystore.KeystoreAccount, error)
    AccountDecryptedKey(address cryptotypes.Address, passphrase string) (keystore.KeystoreAccount, *ecdsa.PrivateKey, *extkeys.ExtendedKey, error)
    Delete(address cryptotypes.Address, passphrase string) error
    Find(address cryptotypes.Address) (keystore.KeystoreAccount, error)
    Accounts() []keystore.KeystoreAccount
    ReEncryptKeyStoreDir(oldPass, newPass string) error
    MigrateKeyStoreDir(newDir string) error
    KeystorePath() string
}
```

### Persistence Interface

Defines data persistence operations:

```go
type Persistence interface {
    AddressExists(address cryptotypes.Address) (bool, error)
    GetProfileKeypair() (*types.Keypair, error)
    GetWalletRootAddress() (cryptotypes.Address, error)
    GetPath(address cryptotypes.Address) (string, error)
    GetKeypairByKeyUID(keyUID string) (*types.Keypair, error)
    GetActiveKeypairs() ([]*types.Keypair, error)
    GetAllKeypairs() ([]*types.Keypair, error)
    SaveOrUpdateKeypair(keypair *types.Keypair) error
    SaveOrUpdateAccounts(accounts []*types.Account, updateKeypairClock bool) error
    SaveOrUpdateKeycard(keycard types.Keycard, clock uint64, updateKeypairClock bool) error
    MarkKeypairFullyOperable(keyUID string, clock uint64, updateKeypairClock bool) error
    MarkAccountFullyOperable(address cryptotypes.Address) error
    DeleteAllKeycardsWithKeyUID(keyUID string, clock uint64) error
    GetPositionForNextNewAccount() (int64, error)
    GetAccountByAddress(address cryptotypes.Address) (*types.Account, error)
    RemoveAccount(address cryptotypes.Address, clock uint64) error
    RemoveKeypair(keyUID string, clock uint64) error
}
```

## Account Types and Operability

The package defines different account types and operability levels:

### Account Types
- **Generated**: Account created from mnemonic
- **Key**: Account created from private key
- **Seed**: Account derived from seed
- **Watch**: Watch-only account

### Keypair Types
- **Profile**: Keypair used for chat profile
- **Key**: Keypair created from private key
- **Seed**: Keypair created from seed/mnemonic

### Account Operability
- **Fully Operable**: Account has keystore file and can sign transactions
- **Partially Operable**: Account has keystore file for derived address but not for itself
- **Non-Operable**: Account has no keystore file and cannot sign transactions

## Error Handling

The package provides comprehensive error handling with specific error types:

```go
var (
    ErrLoggerIsMissing                        = errors.NewError(ErrCodeLoggerMissing, "logger is missing", getErrorCategory)
    ErrKeystoreMissing                        = errors.NewError(ErrCodeKeystoreMissing, "keystore is missing", getErrorCategory)
    ErrPersistenceMissing                     = errors.NewError(ErrCodePersistenceMissing, "persistence is missing", getErrorCategory)
    ErrNoAccountSelected                      = errors.NewError(ErrCodeNoAccountSelected, "no account selected", getErrorCategory)
    ErrAccountDoesNotExist                    = errors.NewError(ErrCodeAccountDoesNotExist, "account does not exist", getErrorCategory)
    ErrAddressAndPasswordOrPrivateKeyRequired = errors.NewError(ErrCodeAddressAndPasswordOrPrivateKeyRequired, "address and password or private key are required", getErrorCategory)
    ErrAccountIsNil                           = errors.NewError(ErrCodeAccountIsNil, "account is nil", getErrorCategory)
    ErrKeypairIsNil                           = errors.NewError(ErrCodeKeypairIsNil, "keypair is nil", getErrorCategory)
    ErrCannotRemoveDefaultChatAccount         = errors.NewError(ErrCodeCannotRemoveDefaultChatAccount, "cannot remove default chat account", getErrorCategory)
    ErrCannotRemoveDefaultWalletAccount       = errors.NewError(ErrCodeCannotRemoveDefaultWalletAccount, "cannot remove default wallet account", getErrorCategory)
    ErrCannotRemoveProfileKeypair             = errors.NewError(ErrCodeCannotRemoveProfileKeypair, "cannot remove profile keypair", getErrorCategory)
    // ... see errors.go for the full list
)
```

## Testing

The package includes comprehensive test coverage:

- **Unit Tests**: Each subpackage includes its own test suite
- **Mock Implementations**: Mock persistence and keystore implementations for testing
- **Integration Tests**: End-to-end testing of account management workflows

## Contributing

When contributing to this package:

1. Ensure all new functionality includes appropriate tests
2. Follow the existing code style and patterns
3. Update documentation for any new public APIs
4. Consider security implications of any changes
5. Test keystore compatibility with existing accounts
6. Place new code in the appropriate subpackage based on its responsibility
7. Maintain backward compatibility for the main package exports
8. Use the mock implementations for testing new functionality
9. Use interfaces for external dependencies