# Transfer Detector

A service for detecting and monitoring cryptocurrency transfers across multiple token standards (ERC-20, ERC-721, ERC-1155) on various blockchain networks.

## Overview

This package provides a unified interface for:
- **Transfer Detection**: Efficiently detect transfers using event filtering
- **Multi-Standard Support**: ERC-20, ERC-721, and ERC-1155 token transfers
- **Multi-Chain Support**: Works across different blockchain networks
- **Block Range Management**: Tracks last fetched blocks to avoid duplicates
- **Event-Driven**: Publishes events for transfer detection operations

## Key Components

### Controller
The main orchestrator that manages transfer detection operations:
- Monitors account and network changes
- Debounces fetch requests (10s debounce, 10min fetch period)
- Coordinates between fetcher and providers
- Manages block number tracking to prevent duplicate detection

### Fetcher
Handles the actual transfer filtering using event logs:
- Uses `eventfilter.FilterTransfers` for efficient event detection
- Supports all transfer types (ERC-20, ERC-721, ERC-1155)
- Works with any Ethereum-compatible chain
- Returns structured event log data

```go
// Create a fetcher
fetcher := transferdetector.NewFetcher(ethClientGetter)
```

### Providers
Required interfaces for integration:
- **AccountsProvider**: Provides wallet addresses to monitor
- **NetworksProvider**: Provides active blockchain networks
- **LastBlockProvider**: Provides latest block numbers for each chain
- **FilterProvider**: Handles the actual transfer filtering

## Usage

```go
// Create a new controller with default configuration
controller := transferdetector.NewController(
    transferdetector.DefaultControllerConfig(),
    fetcher,
    accountsProvider,
    accountsPublisher,
    networksProvider,
    lastBlockProvider,
    logger,
)

// Or with custom configuration
config := transferdetector.ControllerConfig{
    FetchDebounceTime: 5 * time.Second,
    FetchPeriod:       5 * time.Minute,
}
controller := transferdetector.NewController(
    config,
    fetcher,
    accountsProvider,
    accountsPublisher,
    networksProvider,
    lastBlockProvider,
    logger,
)

// Start the controller
controller.Start()

// Trigger transfer detection
controller.TriggerFetch()

// Listen for events
publisher := controller.GetPublisher()
// Subscribe to transfer detection events...
```

## Events

- `EventTransferDetectionStarted`: Transfer detection initiated
- `EventTransferDetectionFinished`: Successful detection completion with found events
- `EventTransferDetectionError`: Detection errors

Each event contains:
- **ChainID**: The blockchain network ID
- **Accounts**: List of monitored wallet addresses
- **FromBlock/ToBlock**: Block range that was scanned
- **Events**: (Finished only) Array of detected transfer events
- **Error**: (Error only) Error details

## Configuration

### Controller Configuration
- **FetchDebounceTime**: 10 seconds (default) - debounce time for detection requests
- **FetchPeriod**: 10 minutes (default) - minimum time between detections

The controller automatically tracks the last fetched block number for each chain to ensure no transfers are missed and no duplicates are detected.

## Transfer Types

The detector supports all major token standards:
- **ERC-20**: Fungible token transfers
- **ERC-721**: NFT transfers (single token)
- **ERC-1155**: Multi-token transfers (batch operations)

## Event Data Structure

Detected transfers include:
- **ContractKey**: The contract address where the transfer occurred
- **EventKey**: The event name (e.g., "Transfer")
- **Unpacked**: Structured data containing transfer details (from, to, value, tokenId, etc.)

## Integration

The transfer detector integrates with:
- **Account Management**: Monitors specified wallet addresses
- **Network Management**: Works across multiple blockchain networks
- **Block Tracking**: Maintains state of last scanned blocks
- **Event Publishing**: Notifies other services of detected transfers

This makes it ideal for building comprehensive wallet applications that need to track all user activity across multiple chains and token standards.
