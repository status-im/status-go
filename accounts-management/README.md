# Accounts Management Package

The Accounts Management package provides a comprehensive solution for Status chat and wallet accounts creation, storage, and management within the Status application. It handles account generation, keystore operations, and secure storage of private keys.

## Overview

This package is responsible for:
- Creating and managing Ethereum accounts
- Secure storage and retrieval of private keys
- Account derivation from mnemonics and private keys
- Keystore operations (encryption, decryption, migration)
- Account selection and authentication

## Package Structure

```
accounts-management/
├── accounts.go          # Main account manager implementation
├── errors.go            # Package-specific error definitions
├── accounts_test.go     # Test suite for accounts
├── common/              # Common utilities and constants
├── generator/           # Account generation and derivation
├── keystore/            # Keystore operations
└── types/               # Type definitions
```

## Key Components

### AccountsManager

The main interface for account management operations. It provides methods for:

- **Account Creation**: Create accounts from mnemonics or private keys
- **Account Storage**: Securely store accounts in the keystore
- **Account Loading**: Load and decrypt accounts from the keystore
- **Account Selection**: Select and manage the currently active account
- **Account Derivation**: Derive child accounts from existing accounts

#### Constructor
```go
// Create a new AccountsManager instance
manager, err := NewAccountsManager(logger)
```

The constructor requires:
- `logger`: A zap.Logger instance for logging operations

After creation, you need to set up the persistence and keystore:
```go
// Set the persistence layer for account data storage
manager.SetPersistence(persistence)

// Set the keystore for secure key storage
manager.SetKeystore(keystore)
```

### Key Features

#### Account Creation
```go
// Create a new account with random mnemonic
account, mnemonic, err := manager.CreateAndStoreAccount(password)

// Create account from existing mnemonic
account, err := manager.CreateFromMnemonicAndStoreAccount(mnemonic, password)

// Create account from private key
account, err := manager.CreateFromPrivateKeyAndStoreAccount(privateKeyHex, password)
```

#### Account Management
```go
// Load an account from keystore
account, err := manager.LoadAccount(address, password)

// Verify account password
valid, err := manager.VerifyAccountPassword(address, password)

// Sets chat account for use
err := manager.SetChatAccount(chatAddress, password)

// Get currently selected chat account
account, err := manager.SelectedChatAccount()
```

#### Account Derivation
```go
// Derive a single child account
childAccount, err := manager.DeriveChildAccountForPathAndStore(
    parentAddress,
    "m/44'/60'/0'/0/1",
    password,
)

// Derive multiple child accounts
childAccounts, err := manager.DeriveChildrenAccountsForPathsAndStore(
    parentAddress,
    []string{"m/44'/60'/0'/0/1", "m/44'/60'/0'/0/2"},
    password,
)
```

#### Keystore Operations
```go
// Migrate keystore to new directory
err := manager.MigrateKeyStoreDir(newDir)

// Re-encrypt keystore with new password
err := manager.ReEncryptKeyStoreDir(oldPassword, newPassword)

// Delete an account - this one currently doesn't require a password because of Status' fork of go-ethereum, but it will as soon as we break that dependency
err := manager.DeleteAccount(address)
```

## Usage Example

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

    // Create persistence instance aligning with `Persistence` interface
    persistence := yourPersistenceImplementation()

    // Create account manager with persistence and logger
    manager, err := accountsmanagement.NewAccountsManager(logger)
    if err != nil {
        log.Fatal(err)
    }

    // Set the persistence layer for account data storage
    manager.SetPersistence(persistence)

    // Set up keystore instance aligning with `KeyStore` interface
    keystore := yourKeystoreImplementation()
    manager.SetKeystore(keystore)

    // Create a new account
    account, mnemonic, err := manager.CreateAndStoreAccount("my-password")
    if err != nil {
        log.Fatal(err)
    }

    log.Printf("Created account: %s", account.Address().Hex())
    log.Printf("Mnemonic: %s", mnemonic)

    // Select the account
    err = manager.SetChatAccount(account.Address(), "my-password")
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

## Security Considerations

- **Password Protection**: All private keys are encrypted using the provided password
- **Secure Storage**: Keys are stored in a Geth-compatible keystore format
- **Memory Management**: Private keys are cleared from memory when not in use
- **Account Isolation**: Each account is stored separately with its own encryption

## Error Handling

The package defines several custom errors:

- `ErrNoAccountSelected`: No account has been selected (login required)
- `ErrAccountKeyStoreMissing`: Keystore is not properly initialized
- `ErrCannotLocateKeyFile`: Account file cannot be found for the given address

## Dependencies

- `go.uber.org/zap`: Logging
- `github.com/status-im/status-go/eth-node/types`: Ethereum types
- `github.com/status-im/status-go/multiaccounts`: Multi-account support
- `Persistence` interface: Required for account data storage operations
- `KeyStore` interface: Required for secure cryptographic key storage, encryption/decryption, and keystore management operations

## Testing

Run the test suite:

```bash
go test ./accounts-management/...
```

The package includes comprehensive tests for all major functionality including account creation, derivation, keystore operations, and error handling.

## Contributing

When contributing to this package:

1. Ensure all new functionality includes appropriate tests
2. Follow the existing code style and patterns
3. Update documentation for any new public APIs
4. Consider security implications of any changes
5. Test keystore compatibility with existing accounts

## Related Packages

- **Generator Package**: For account creation and derivation logic
- **Keystore Package**: For secure key storage operations
- **Types Package**: For common type definitions
- **Common Package**: For utility functions and constants