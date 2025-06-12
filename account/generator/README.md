# Account Generator

The Account Generator package provides functionality for creating and deriving Ethereum accounts. It offers several ways to create accounts and derive child keys from them.

## Account Creation

The generator provides three main ways to create accounts:

1. **From Mnemonic**: Using `CreateAccountFromMnemonic(mnemonic, bip39Passphrase)` to create an account from a mnemonic phrase.
2. **From Private Key**: Using `CreateAccountFromPrivateKey(privateKeyHex)` to create an account from a private key.
3. **From Key**: Using `CreateAccountFromKey(key)` to create an account from an existing key.

For batch account creation, you can use `CreateAccountsOfMnemonicLength(mnemonicPhraseLength, n, bip39Passphrase)` which returns both the created accounts and their corresponding mnemonic phrases.

## Account Derivation

The package provides functionality to derive child accounts from existing accounts:

1. **Single Derivation**: Using `DeriveChildFromAccount(account, pathString)` to derive a single child account at a specified path.
2. **Multiple Derivations**: Using `DeriveChildrenFromAccount(account, pathStrings)` to derive multiple child accounts at different paths.

The derivation paths follow the BIP44 standard format (e.g., "m/44'/60'/0'/0/0").

## Account Types

The package defines several types for working with accounts:

- `Account`: Represents an Ethereum account with a private key and optional extended key.
- `AccountInfo`: Contains basic account information (private key, public key, address).
- `IdentifiedAccountInfo`: Extends AccountInfo with KeyUID.
- `GeneratedAccountInfo`: Extends IdentifiedAccountInfo with a mnemonic phrase.
- `GeneratedAndDerivedAccountInfo`: Extends GeneratedAccountInfo with derived accounts.

Note: This package is focused on account creation and derivation. For account storage and management, please refer to the main account package.