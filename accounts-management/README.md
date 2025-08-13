# Accounts Management Package

The Accounts Management package provides a comprehensive solution for Status chat and wallet accounts creation, storage, and management within the Status application. It handles account generation, keystore operations, and secure storage of private keys. For cryptographic operations, see the separate `crypto` package.

## Package Structure

The package is organized into focused subpackages:

```
accounts-management/
├── accounts.go           # Main package with public API and re-exports
├── core/                 # Main account manager implementation
│   ├── manager.go        # AccountsManager and core business logic
│   ├── keystore_operations.go  # Keystore-related operations
│   ├── persistence_operations.go  # Database operations
│   └── manager_test.go   # Test suite for core functionality

├── errors/               # Structured error handling system
│   ├── errors.go         # Error definitions, codes, and categories
│   ├── errors_test.go    # Error system tests
│   ├── errors_test_data.go # Test data for error tests
│   └── README.md         # Error handling documentation
├── keystore/             # Cryptographic key storage operations
│   ├── interface.go      # KeyStore interface definition
│   └── geth/             # Geth-compatible keystore implementation
│       ├── adapter.go    # Geth keystore adapter
│       ├── encryption.go # Encryption operations
│       ├── decryption.go # Decryption operations
│       ├── reencryption.go # Re-encryption operations
│       ├── migration.go  # Migration utilities
│       ├── helper.go     # Helper functions
│       └── const.go      # Constants
├── persistence/          # Database operations for account data
│   └── interface.go      # Persistence interface definition
├── generator/            # Account creation and derivation
│   ├── generator.go      # Account generation functions
│   ├── types.go          # Account types and methods
│   ├── path_decoder.go   # BIP32 path parsing
│   ├── README.md         # Generator-specific documentation
│   └── *_test.go         # Generator tests
├── types/                # Core type definitions
│   ├── types.go          # Core type definitions
│   ├── key.go            # Key type structure
│   ├── account.go        # Account-related types
│   ├── keypair.go        # Keypair-related types
│   └── *_test.go         # Type tests
├── common/               # Shared utilities
│   ├── const.go          # Constants and paths
│   ├── mnemonic.go       # Mnemonic generation utilities
│   ├── address.go        # Address utilities
│   ├── publickey.go      # Public key utilities
│   ├── utils.go          # General utilities
│   └── *_test.go         # Common tests
└── mock/                 # Mock implementations for testing
    └── persistence.go    # Mock persistence implementation
```

## Key Components

### Core Package

The main account management operations are in the `core` package:

- **AccountsManager**: Main interface for account management operations including account creation, selection, and verification
- **Keystore Operations**: Methods for keystore management and account loading
- **Persistence Operations**: Methods for database operations including keypair and account management

### Errors Package

A structured error handling system in the `errors` package:

- **Structured Errors**: Rich error types with codes, categories, and context
- **Error Codes**: Numeric error codes for programmatic error handling
- **Error Categories**: Logical grouping for different error types
- **Context Support**: Add key-value metadata to errors for debugging
- **Helper Functions**: Pre-built error creation functions for common patterns

### Keystore Package

Handles secure cryptographic key storage:

- **KeyStore Interface**: Defines the contract for keystore implementations
- **Geth Implementation**: Complete Geth-compatible keystore adapter with encryption, decryption, re-encryption, and migration capabilities

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

    "github.com/status-im/status-go/accounts-management"
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
        walletAccount,
        true,  // profile keypair
        false, // not keycard
        0,     // clock
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

    log.Printf("Selected account: %s", selectedAccount.Address().Hex())
}
```

### Advanced Usage

For more advanced usage, you can import specific subpackages:

```go
import (
    "github.com/status-im/status-go/accounts-management/core"
    "github.com/status-im/status-go/accounts-management/generator"
    "github.com/status-im/status-go/accounts-management/types"
    "github.com/status-im/status-go/accounts-management/errors"
)

// Use specific types and functions
account := generator.CreateAccountFromMnemonic(mnemonic, passphrase)
keypair := &types.Keypair{...}

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

// Use account management functions
account := generator.CreateAccountFromMnemonic(mnemonic, passphrase)
keypair := &types.Keypair{...}
```



### Error Handling

The errors package provides a structured error handling system:

```go
import "github.com/status-im/status-go/accounts-management/errors"

// Create structured errors
err := errors.NewError(errors.ErrCodeLoggerMissing, "logger is missing", errors.getErrorCategory)

// Add context to errors
err = err.WithContext("function", "LoadAccount")

// Wrap existing errors
wrappedErr := errors.WrapError(errors.ErrCodeWrongPasswordProvided, "wrong password", originalErr, errors.getErrorCategory)

// Use helper functions
func ErrDerivingAddress(keyUID string, path string) *AccountsError {
	return NewError(ErrCodeErrorDerivingAddress, "error deriving address from keypair", getErrorCategory).
        WithContext("keyuid", keyUID).
        WithContext("path", path)
}

err := errors.ErrDerivingAddress("0x123", "m/1'")
```

### Account Generation

The generator package provides flexible account creation:

```go
import "github.com/status-im/status-go/accounts-management/generator"

// Create account from mnemonic
account := generator.CreateAccountFromMnemonic(mnemonic, passphrase)

// Create account from private key
account := generator.CreateAccountFromPrivateKey(privateKeyHex)

// Derive child account
childAccount := generator.DeriveChildFromAccount(account, "m/44'/60'/0'/0/1")

// Create multiple accounts
accounts, mnemonics := generator.CreateAccountsOfMnemonicLength(12, 5, passphrase)
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
    walletAccount,
    true,  // profile keypair
    false, // not keycard
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
type AccountsManager struct {
    // Core account management operations
    CreateKeypairFromMnemonicAndStore(mnemonic string, password string, keypairName string, walletAccount *types.AccountCreationDetails, profile bool, keycard bool, clock uint64) (*types.Keypair, error)
    CreateKeypairFromPrivateKeyAndStore(privateKey string, password string, keypairName string, walletAccount *types.AccountCreationDetails, clock uint64) (*types.Keypair, error)
    SetChatAccount(address ethtypes.Address, password string, privateKey *ecdsa.PrivateKey) error
    SelectedChatAccount() (*generator.Account, error)
    LoadAccount(address ethtypes.Address, password string) (*generator.Account, error)
    VerifyAccountPassword(address ethtypes.Address, password string) (bool, error)
    GetVerifiedWalletAccount(address ethtypes.Address, password string) (*generator.Account, error)
    Logout()
    Accounts() ([]ethtypes.Address, error)
}
```



### KeyStore Interface

Defines keystore operations:

```go
type KeyStore interface {
    AccountDecryptedKey(address types.Address, password string) (types.KeystoreAccount, *ecdsa.PrivateKey, *extkeys.ExtendedKey, error)
    ImportECDSA(priv *ecdsa.PrivateKey, password string) (types.KeystoreAccount, error)
    // ... other keystore operations
}
```

### Persistence Interface

Defines data persistence operations:

```go
type Persistence interface {
    AddressExists(address ethtypes.Address) (bool, error)
    GetProfileKeypair() (*types.Keypair, error)
    GetWalletRootAddress() (ethtypes.Address, error)
    GetPath(address ethtypes.Address) (string, error)
    GetKeypairByKeyUID(keyUID string) (*types.Keypair, error)
    GetActiveKeypairs() ([]*types.Keypair, error)
    SaveOrUpdateKeypair(keypair *types.Keypair) error
    SaveOrUpdateAccounts(accounts []*types.Account, updateKeypairClock bool) error
    SaveOrUpdateKeycard(keycard *types.Keycard, clock uint64, updateKeypairClock bool) error
    MarkKeypairFullyOperable(keyUID string, clock uint64, updateKeypairClock bool) error
    MarkAccountFullyOperable(address ethtypes.Address) error
    DeleteAllKeycardsWithKeyUID(keyUID string, clock uint64) error
    GetPositionForNextNewAccount() (int64, error)
    GetAccountByAddress(address ethtypes.Address) (*types.Account, error)
    RemoveAccount(address ethtypes.Address, clock uint64) error
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