# Multi-Standard Balance

A service for fetching and managing cryptocurrency balances across multiple token standards (Native, ERC-20, ERC-721, ERC-1155) on various blockchain networks.

## Overview

This package provides a unified interface for:
- **Balance Fetching**: Efficiently fetch balances using Multicall3 contracts
- **Multi-Standard Support**: Native tokens, ERC-20, ERC-721, and ERC-1155 tokens
- **Multi-Chain Support**: Works across different blockchain networks
- **Caching & State Management**: Tracks balance states and fetch timestamps
- **Event-Driven**: Publishes events for balance changes and fetch operations

## Key Components

### Controller
The main orchestrator that manages balance fetching operations:
- Monitors account and network changes, token list updates and pending transactions
- Leaves when and what to fetch to `refresh.Policy` from go-wallet-sdk (`pkg/balance/refresh`) with the `HeldEveryTick` schedule: the held set every 2 min on screen and every 15 min hidden, every balance on start and hourly on screen, other reads after a 10s debounce, a confirmed transaction at once
- The held set of an account on a chain is its native coin, its tokens with a balance, the tokens of `HeldTokensProvider` (custom tokens, pending ERC20 transfers) and its known collectibles
- A full refresh of a chain that fails is retried at the next period
- Coordinates between storage, fetcher, and providers

### Fetcher
Handles the actual balance fetching using Multicall3:
- Supports configurable batch operations (default: 10,000 calls per batch)
- Works with any chain that supports Multicall3
- Returns results via channels for async processing

```go
// Create a fetcher with default batch size
fetcher := multistandardbalance.NewFetcher(ethClientGetter, multistandardbalance.DefaultBatchSize)

// Or with custom batch size
fetcher := multistandardbalance.NewFetcher(ethClientGetter, 5000)
```

### Storage
Abstract interface for persisting balance data:
- Tracks balance values and metadata (block number, hash, fetch time)
- Keeps the block each token was read at: a read takes a token only when it is newer, and keeps the tokens it did not cover
- Supports all token standards
- Handles cleanup of missing accounts/chains

## Usage

```go
// Create a new controller with default configuration
controller := multistandardbalance.NewController(
    multistandardbalance.DefaultControllerConfig(),
    storage,
    fetcher,
    accountsProvider,
    accountsPublisher,
    networksProvider,
    tokenListProvider,
    collectibleListProvider,
    heldTokensProvider,
    lastBlockManager,
    walletFeed,
    logger,
)

// Or with custom configuration
config := multistandardbalance.ControllerConfig{
    FetchDebounceTime: 5 * time.Second,
    FetchPeriod:       1 * time.Minute,
}
controller := multistandardbalance.NewController(
    config,
    storage,
    fetcher,
    // ... other parameters
)

// Start the controller
controller.Start()

// Trigger balance fetches
controller.TriggerFullFetch() // Fetch all accounts/networks/types

// Or trigger selective fetches
fetchConfig := multistandardbalance.FetchConfig{
    multistandardbalance.BalancesKey{Account: addr, ChainID: 1}: {
        multistandardfetcher.ResultTypeNative,
        multistandardfetcher.ResultTypeERC20,
    },
}
controller.TriggerFetchWithConfig(fetchConfig)

// Listen for events
publisher := controller.GetPublisher()
// Subscribe to balance change events...
```

## Events

- `EventBalanceFetchStarted`: Fetches initiated
- `EventBalanceFetchFinished`: Successful fetch completion
- `EventBalanceFetchError`: Fetch errors
- `EventBalanceFetchFailedToStart`: Failed to initiate fetch

## Configuration

### Controller Configuration
- **FetchDebounceTime**: 10 seconds (default) - debounce time for fetch requests
- **FetchPeriod**: 2 minutes (default) - minimum time between fetches

### Fetcher Configuration
- **BatchSize**: 10,000 (default) - number of calls per Multicall3 batch

Both configurations can be customized when creating the respective components.
