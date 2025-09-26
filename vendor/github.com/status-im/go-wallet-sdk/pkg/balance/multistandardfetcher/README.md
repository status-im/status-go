# Multi-Standard Fetcher

A high-level package for fetching balances across multiple token standards (Native ETH, ERC20, ERC721, ERC1155) using Multicall3 batched calls.

## Overview

The `multistandardfetcher` package provides a unified interface for fetching balances across different token standards in a single operation. It leverages the underlying `multicall` package to efficiently batch multiple contract calls and returns results asynchronously through a channel-based API.

## Features

- **Multi-Standard Support**: Native ETH, ERC20, ERC721, and ERC1155 tokens
- **Asynchronous Processing**: Returns results through channels as they become available
- **Efficient Batching**: Uses Multicall3 to minimize RPC calls
- **Type-Safe Results**: Strongly typed result structures for each token standard
- **Error Handling**: Graceful handling of individual call failures
- **Block Information**: Includes block number and hash for all results

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "math/big"
    
    "github.com/ethereum/go-ethereum/common"
    "github.com/ethereum/go-ethereum/ethclient"
    
    "github.com/status-im/go-wallet-sdk/pkg/balance/multistandardfetcher"
    "github.com/status-im/go-wallet-sdk/pkg/contracts/multicall3"
)

func main() {
    // Setup
    rpcURL := "https://eth-mainnet.g.alchemy.com/v2/YOUR_API_KEY"
    client, _ := ethclient.Dial(rpcURL)
    defer client.Close()
    
    // Get Multicall3 address
    multicallAddr, _ := multicall3.GetMulticall3Address(1)
    multicallContract, _ := multicall3.NewMulticall3(multicallAddr, client)
    
    // Configure what to fetch
    config := multistandardfetcher.FetchConfig{
        Native: []multistandardfetcher.AccountAddress{
            common.HexToAddress("0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045"), // vitalik.eth
        },
        ERC20: map[multistandardfetcher.AccountAddress][]multistandardfetcher.ContractAddress{
            common.HexToAddress("0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045"): {
                common.HexToAddress("0xA0b86a33E6441b8C4C8C0C4C0C4C0C4C0C4C0C4C0"), // USDC
                common.HexToAddress("0x6B175474E89094C44Da98b954EedeAC495271d0F"), // DAI
            },
        },
        ERC721: map[multistandardfetcher.AccountAddress][]multistandardfetcher.ContractAddress{
            common.HexToAddress("0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045"): {
                common.HexToAddress("0xBC4CA0EdA7647A8aB7C2061c2E118A18a936f13D"), // BAYC
            },
        },
        ERC1155: map[multistandardfetcher.AccountAddress][]multistandardfetcher.CollectibleID{
            common.HexToAddress("0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045"): {
                {
                    ContractAddress: common.HexToAddress("0x495f947276749Ce646f68AC8c248420045cb7b5e"),
                    TokenID:         big.NewInt(1),
                },
            },
        },
    }
    
    // Fetch balances
    ctx := context.Background()
    resultsCh := multistandardfetcher.FetchBalances(ctx, multicallAddr, multicallContract, config, 100)
    
    // Process results
    for result := range resultsCh {
        switch result.ResultType {
        case multistandardfetcher.ResultTypeNative:
            native := result.Result.(multistandardfetcher.NativeResult)
            fmt.Printf("Native balance for %s: %s wei\n", native.Account.Hex(), native.Result.String())
            
        case multistandardfetcher.ResultTypeERC20:
            erc20 := result.Result.(multistandardfetcher.ERC20Result)
            fmt.Printf("ERC20 balances for %s:\n", erc20.Account.Hex())
            for token, balance := range erc20.Results {
                fmt.Printf("  %s: %s\n", token.Hex(), balance.String())
            }
            
        case multistandardfetcher.ResultTypeERC721:
            erc721 := result.Result.(multistandardfetcher.ERC721Result)
            fmt.Printf("ERC721 balances for %s:\n", erc721.Account.Hex())
            for nft, balance := range erc721.Results {
                fmt.Printf("  %s: %s NFTs\n", nft.Hex(), balance.String())
            }
            
        case multistandardfetcher.ResultTypeERC1155:
            erc1155 := result.Result.(multistandardfetcher.ERC1155Result)
            fmt.Printf("ERC1155 balances for %s:\n", erc1155.Account.Hex())
            for collectibleID, balance := range erc1155.Results {
                originalID := collectibleID.ToCollectibleID()
                fmt.Printf("  %s (token %s): %s\n", 
                    originalID.ContractAddress.Hex(), 
                    originalID.TokenID.String(), 
                    balance.String())
            }
        }
    }
}
```

## API Reference

### Types

#### FetchConfig
Configuration struct for specifying what balances to fetch:

```go
type FetchConfig struct {
    Native  []AccountAddress                    // Accounts to fetch native ETH balances for
    ERC20   map[AccountAddress][]ContractAddress // Account -> ERC20 tokens to check
    ERC721  map[AccountAddress][]ContractAddress // Account -> ERC721 contracts to check
    ERC1155 map[AccountAddress][]CollectibleID   // Account -> ERC1155 collectibles to check
}
```

#### Result Types

```go
type ResultType string

const (
    ResultTypeNative  ResultType = "native"
    ResultTypeERC20   ResultType = "erc20"
    ResultTypeERC721  ResultType = "erc721"
    ResultTypeERC1155 ResultType = "erc1155"
)

type FetchResult struct {
    ResultType ResultType
    Result     any  // One of: NativeResult, ERC20Result, ERC721Result, ERC1155Result
}
```

#### Individual Result Types

```go
// Native ETH balance result
type NativeResult struct {
    Account       AccountAddress
    Result        *big.Int
    Err           error
    AtBlockNumber *big.Int
    AtBlockHash   common.Hash
}

// ERC20/ERC721 balance results (map of contract -> balance)
type ERC20Result = Results[ContractAddress]
type ERC721Result = Results[ContractAddress]

// ERC1155 balance results (map of collectible -> balance)
type ERC1155Result = Results[HashableCollectibleID]

type Results[T comparable] struct {
    Account       AccountAddress
    Results       map[T]*big.Int
    Err           error
    AtBlockNumber *big.Int
    AtBlockHash   common.Hash
}
```

#### Type Aliases

```go
type AccountAddress = common.Address
type ContractAddress = common.Address
type HashableTokenID = [32]byte

type CollectibleID struct {
    ContractAddress ContractAddress
    TokenID         *big.Int
}

type HashableCollectibleID struct {
    ContractAddress ContractAddress
    TokenID         HashableTokenID
}
```

### Functions

#### FetchBalances

```go
func FetchBalances(
    ctx context.Context,
    multicall3Address common.Address,
    caller multicall.Caller,
    config FetchConfig,
    batchSize int,
) <-chan FetchResult
```

Fetches balances asynchronously using Multicall3 batched calls.

**Parameters:**
- `ctx`: Context for cancellation and timeout
- `multicall3Address`: Multicall3 contract address for the target chain
- `caller`: Multicall3 caller interface (usually a contract instance)
- `config`: Configuration specifying what balances to fetch
- `batchSize`: Maximum number of calls per batch

**Returns:**
- `<-chan FetchResult`: Channel that receives results as they become available

## Usage Patterns

### Fetching Only Native Balances

```go
config := multistandardfetcher.FetchConfig{
    Native: []multistandardfetcher.AccountAddress{
        common.HexToAddress("0x1234..."),
        common.HexToAddress("0x5678..."),
    },
}
```

### Fetching ERC20 Balances for Multiple Accounts

```go
config := multistandardfetcher.FetchConfig{
    ERC20: map[multistandardfetcher.AccountAddress][]multistandardfetcher.ContractAddress{
        common.HexToAddress("0x1234..."): {
            common.HexToAddress("0xUSDC..."),
            common.HexToAddress("0xDAI..."),
        },
        common.HexToAddress("0x5678..."): {
            common.HexToAddress("0xUSDC..."),
        },
    },
}
```

### Fetching ERC1155 Collectibles

```go
config := multistandardfetcher.FetchConfig{
    ERC1155: map[multistandardfetcher.AccountAddress][]multistandardfetcher.CollectibleID{
        common.HexToAddress("0x1234..."): {
            {
                ContractAddress: common.HexToAddress("0xContract..."),
                TokenID:         big.NewInt(1),
            },
            {
                ContractAddress: common.HexToAddress("0xContract..."),
                TokenID:         big.NewInt(2),
            },
        },
    },
}
```

### Mixed Balance Types

```go
config := multistandardfetcher.FetchConfig{
    Native: []multistandardfetcher.AccountAddress{account1},
    ERC20: map[multistandardfetcher.AccountAddress][]multistandardfetcher.ContractAddress{
        account1: {token1, token2},
    },
    ERC721: map[multistandardfetcher.AccountAddress][]multistandardfetcher.ContractAddress{
        account2: {nft1},
    },
    ERC1155: map[multistandardfetcher.AccountAddress][]multistandardfetcher.CollectibleID{
        account2: {
            {ContractAddress: contract1, TokenID: big.NewInt(1)},
        },
    },
}
```

## Error Handling

The package provides comprehensive error handling at multiple levels:

1. **Job-level errors**: If the entire job fails (e.g., network error), the `Err` field in the result will be set
2. **Call-level errors**: Individual calls that fail are skipped in the results map
3. **Context cancellation**: Respects context cancellation and timeout

```go
for result := range resultsCh {
    switch result.ResultType {
    case multistandardfetcher.ResultTypeNative:
        native := result.Result.(multistandardfetcher.NativeResult)
        if native.Err != nil {
            log.Printf("Failed to fetch native balance: %v", native.Err)
            continue
        }
        // Process successful result
        fmt.Printf("Balance: %s\n", native.Result.String())
    }
}
```

## Performance Considerations

- **Batch Size**: Adjust `batchSize` based on your RPC provider's limits (typically 100-1000)
- **Concurrent Processing**: Results are processed asynchronously as they become available
- **Memory Usage**: Large result sets are streamed through channels to minimize memory usage
- **RPC Limits**: The package automatically chunks large requests to respect RPC limits

## Dependencies

- `github.com/status-im/go-wallet-sdk/pkg/multicall` - For Multicall3 batching
- `github.com/status-im/go-wallet-sdk/pkg/contracts/multicall3` - For Multicall3 contract bindings
- `github.com/ethereum/go-ethereum` - For Ethereum types and client

## Examples

### Complete Example Application

See `examples/multistandardfetcher-example/` for a complete working example that demonstrates:
- Fetching balances across all token standards (Native ETH, ERC20, ERC721, ERC1155)
- Querying popular tokens and NFT collections for vitalik.eth
- Formatted output with token symbols and readable balances
- Error handling
- Real-world usage patterns

```bash
# Run the complete example
cd examples/multistandardfetcher-example
export RPC_URL="https://eth-mainnet.g.alchemy.com/v2/YOUR_API_KEY"
go run main.go
```
