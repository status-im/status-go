# Crypto Package

The crypto package provides a comprehensive set of cryptographic operations for the Status Go application. It offers a clean interface for cryptographic functions with support for multiple providers, primarily focused on Ethereum-compatible operations.

## Overview

This package provides:
- **Cryptographic primitives**: Key generation, signing, hashing, and address derivation
- **Provider abstraction**: Pluggable crypto providers for different implementations
- **Ethereum compatibility**: Full support for Ethereum cryptographic standards
- **Type safety**: Strongly typed cryptographic types (Address, Hash, etc.)

## Package Structure

```
crypto/
├── README.md              # This file
├── crypto.go              # Main crypto functions and provider management
├── interface.go           # CryptoProvider interface definition
├── ethereum_crypto.go     # Ethereum-specific crypto operations
├── types/                 # Cryptographic type definitions
│   ├── address.go         # Ethereum address type and operations
│   ├── hash.go            # Hash type and operations
│   ├── hex.go             # Hex encoding/decoding utilities
│   ├── utils.go           # General crypto utilities
│   └── errors.go          # Crypto-specific error types
└── geth/                  # Geth crypto provider implementation
    └── provider.go        # Go-ethereum based crypto provider
```

## Quick Start

### Basic Usage

```go
import "github.com/status-im/status-go/crypto"

// Generate a new private key
privateKey, err := crypto.GenerateKey()
if err != nil {
    log.Fatal(err)
}

// Derive address from public key
address := crypto.PubkeyToAddress(privateKey.PublicKey)

// Compute Keccak256 hash
hash := crypto.Keccak256([]byte("Hello, World!"))

// Sign data
signature, err := crypto.Sign(hash, privateKey)
if err != nil {
    log.Fatal(err)
}
```

### Using Crypto Types

```go
import (
    "github.com/status-im/status-go/crypto"
    cryptotypes "github.com/status-im/status-go/crypto/types"
)

// Create address from hex string
addr := cryptotypes.Address("0x742d35Cc6634C0532925a3b8D4C9db96590c6d0b")

// Create hash from hex string
hash := cryptotypes.Hash("0x1234567890abcdef...")

// Convert to hex string
hexAddr := addr.Hex()
hexHash := hash.Hex()
```

## Core Functions

### Key Management

- `GenerateKey()` - Generate new ECDSA private key
- `HexToECDSA(hexkey)` - Convert hex string to private key
- `FromECDSA(prv)` - Convert private key to bytes
- `FromECDSAPub(pub)` - Convert public key to bytes

### Address Operations

- `PubkeyToAddress(pub)` - Derive Ethereum address from public key
- `CreateAddress(b, nonce)` - Create contract address from address and nonce

### Hashing

- `Keccak256(data...)` - Compute Keccak256 hash
- `Keccak256Hash(data...)` - Compute hash and return Hash type
- `TextHash(data)` - Hash text with Ethereum message prefix
- `TextAndHash(data)` - Hash text and return both hash and hex string

### Signing

- `Sign(digestHash, prv)` - Sign hash with private key
- `SignBytes(data, prv)` - Sign arbitrary data (hashes first)
- `SignBytesAsHex(data, prv)` - Sign data and return hex signature
- `SignStringAsHex(data, prv)` - Sign string and return hex signature
- `SigToPub(hash, sig)` - Recover public key from signature

### Key Conversion

- `ToECDSA(data)` - Convert bytes to private key
- `ToECDSAUnsafe(data)` - Convert bytes to private key without validation
- `DecompressPubkey(pubkey)` - Decompress compressed public key
- `CompressPubkey(pubkey)` - Compress public key to 33-byte format

### Advanced Operations

- `GenerateSharedKey(myKey, theirKey, sskLen)` - Generate shared secret key
- `S256()` - Get secp256k1 curve instance

## Crypto Provider Interface

The package uses a provider pattern to allow different cryptographic implementations:

```go
type CryptoProvider interface {
    S256() elliptic.Curve
    GenerateKey() (*ecdsa.PrivateKey, error)
    HexToECDSA(hexkey string) (*ecdsa.PrivateKey, error)
    FromECDSA(prv *ecdsa.PrivateKey) []byte
    FromECDSAPub(pub *ecdsa.PublicKey) []byte
    PubkeyToAddress(p ecdsa.PublicKey) types.Address
    Keccak256(data ...[]byte) []byte
    ToECDSAUnsafe(data []byte) *ecdsa.PrivateKey
    ToECDSA(d []byte) (*ecdsa.PrivateKey, error)
    TextHash(data []byte) []byte
    TextAndHash(data []byte) ([]byte, string)
    Keccak256Hash(data ...[]byte) (h types.Hash)
    Sign(digestHash []byte, prv *ecdsa.PrivateKey) (sig []byte, err error)
    SigToPub(hash, sig []byte) (*ecdsa.PublicKey, error)
    UnmarshalPubkey(pub []byte) (*ecdsa.PublicKey, error)
    CreateAddress(b types.Address, nonce uint64) types.Address
    DecompressPubkey(pubkey []byte) (*ecdsa.PublicKey, error)
    CompressPubkey(pubkey *ecdsa.PublicKey) []byte
    GenerateSharedKey(myIdentityKey *ecdsa.PrivateKey, theirEphemeralKey *ecdsa.PublicKey, sskLen int) ([]byte, error)
}
```

### Setting Custom Provider

```go
import "github.com/status-im/status-go/crypto"

// Create custom provider
customProvider := &MyCryptoProvider{}

// Set as default provider
crypto.SetProvider(customProvider)
```

## Types

### Address

Represents an Ethereum address (20 bytes):

```go
type Address [20]byte

// Methods:
func (a Address) Hex() string
func (a Address) String() string
func (a Address) Bytes() []byte
func (a Address) IsZero() bool
```

### Hash

Represents a cryptographic hash (32 bytes):

```go
type Hash [32]byte

// Methods:
func (h Hash) Hex() string
func (h Hash) String() string
func (h Hash) Bytes() []byte
func (h Hash) IsZero() bool
```

## Error Handling

The package provides structured error handling:

```go
import "github.com/status-im/status-go/crypto/types"

// Check for specific error types
if err != nil {
    var cryptoErr *types.CryptoError
    if errors.As(err, &cryptoErr) {
        switch cryptoErr.Category {
        case types.ErrorCategoryKey:
            log.Printf("Key error: %s", cryptoErr.Message)
        case types.ErrorCategoryHash:
            log.Printf("Hash error: %s", cryptoErr.Message)
        }
    }
}
```

## Examples

### Complete Key Generation and Address Derivation

```go
package main

import (
    "fmt"
    "log"

    "github.com/status-im/status-go/crypto"
    cryptotypes "github.com/status-im/status-go/crypto/types"
)

func main() {
    // Generate new private key
    privateKey, err := crypto.GenerateKey()
    if err != nil {
        log.Fatal("Failed to generate key:", err)
    }

    // Derive address
    address := crypto.PubkeyToAddress(privateKey.PublicKey)

    fmt.Printf("Generated address: %s\n", address.Hex())
    fmt.Printf("Private key (hex): %x\n", crypto.FromECDSA(privateKey))
}
```

### Message Signing and Verification

```go
package main

import (
    "fmt"
    "log"

    "github.com/status-im/status-go/crypto"
)

func main() {
    // Generate key
    privateKey, err := crypto.GenerateKey()
    if err != nil {
        log.Fatal(err)
    }

    // Message to sign
    message := "Hello, Status!"

    // Sign message
    signature, err := crypto.SignStringAsHex(message, privateKey)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Message: %s\n", message)
    fmt.Printf("Signature: %s\n", signature)

    // Verify signature (recover public key)
    hash := crypto.TextHash([]byte(message))
    sigBytes, _ := hex.DecodeString(signature)
    recoveredPub, err := crypto.SigToPub(hash, sigBytes)
    if err != nil {
        log.Fatal("Signature verification failed:", err)
    }

    fmt.Printf("Signature verified! Recovered public key: %x\n", crypto.FromECDSAPub(recoveredPub))
}
```

## Dependencies

- **Go** - For modern Go features
- **go-ethereum** - For Geth crypto provider implementation
- **crypto/ecdsa** - Standard Go ECDSA implementation
- **crypto/elliptic** - Standard Go elliptic curve operations

## Contributing

When contributing to this package:

1. Follow Go best practices and conventions
2. Add tests for new functionality
3. Update this README for new features
4. Ensure compatibility with existing providers
5. Use the existing error types and patterns

## License

This package is part of the Status Go project and follows the same licensing terms.
