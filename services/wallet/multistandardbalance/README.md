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
- Monitors account and network changes
- Debounces fetch requests (10s debounce, 2min fetch period)
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
    lastBlockManager,
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
controller.TriggerFetch(false) // false = not forced

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
