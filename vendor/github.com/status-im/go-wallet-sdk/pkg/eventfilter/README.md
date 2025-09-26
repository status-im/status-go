# EventFilter

Efficient filtering for Ethereum transfer events across ERC20, ERC721, and ERC1155 tokens. Minimizes `eth_getLogs` API calls while capturing all relevant transfers involving specified addresses with concurrent processing capabilities.

## Features

- **Multi-Token Support**: ERC20, ERC721, and ERC1155 transfers
- **Direction Filtering**: Send, receive, or both directions
- **Concurrent Processing**: Parallel execution of multiple filter queries for improved performance
- **Optimized Queries**: Uses FilterQuery OR operations to minimize API calls
- **Address-Based Filtering**: Capture transfers involving any specified addresses
- **Contract Filtering**: Optional filtering by specific contract addresses
- **Clean API**: Simple switch-based implementation for easy maintenance

## Usage

### Basic Usage with FilterTransfers

```go
import (
    "context"
    "math/big"
    "github.com/ethereum/go-ethereum/common"
    "github.com/status-im/go-wallet-sdk/pkg/eventfilter"
)

// Create filter configuration
config := eventfilter.TransferQueryConfig{
    FromBlock:     big.NewInt(18000000),
    ToBlock:       big.NewInt(18001000),
    Accounts:      []common.Address{common.HexToAddress("0x1234...")},
    TransferTypes: []eventfilter.TransferType{
        eventfilter.TransferTypeERC20,
        eventfilter.TransferTypeERC721,
        eventfilter.TransferTypeERC1155,
    },
    Direction: eventfilter.Both, // Send, Receive, or Both
}

// Filter and parse events with concurrent processing
events, err := eventfilter.FilterTransfers(ctx, client, config)
if err != nil {
    log.Fatal(err)
}

// Process events...
for _, event := range events {
    // Access parsed event data via event.Unpacked
}
```

### Manual Query Generation

```go
// Generate optimized filter queries manually
queries := config.ToFilterQueries()

// Execute queries manually
for _, query := range queries {
    logs, err := client.FilterLogs(ctx, query)
    // Process logs...
}
```

## Configuration Options

### TransferQueryConfig

```go
type TransferQueryConfig struct {
    FromBlock         *big.Int           // Start block number
    ToBlock           *big.Int           // End block number  
    ContractAddresses []common.Address   // Optional contract addresses to filter
    Accounts          []common.Address   // Addresses to filter for
    TransferTypes     []TransferType     // Token types to include
    Direction         Direction          // Transfer direction filter
}
```

### Direction
- **`Send`**: Only transfers where specified addresses are the sender
- **`Receive`**: Only transfers where specified addresses are the recipient  
- **`Both`**: Transfers in both directions

### Transfer Types
- **`TransferTypeERC20`**: ERC20 token transfers
- **`TransferTypeERC721`**: ERC721 NFT transfers
- **`TransferTypeERC1155`**: ERC1155 multi-token transfers

### Contract Filtering
- **`ContractAddresses`**: Optional slice of contract addresses to filter by
- If empty, searches all contracts
- If specified, only events from these contracts are returned

## Query Efficiency

The package minimizes API calls through intelligent query merging:

### Single Transfer Types
- **ERC20/ERC721 only**: 1-2 queries (Send + Receive)
- **ERC1155 only**: 1-2 queries (Send + Receive)

### Mixed Transfer Types
- **ERC20 + ERC721**: 1-2 queries (shared event signature)
- **ERC20/ERC721 + ERC1155**: 2-3 queries (optimized with merging)
- **All types**: 2-3 queries maximum

### Optimization Techniques
- **Event Signature Merging**: Multiple event types in single query using OR operations
- **Topic Structure Optimization**: Merges compatible queries by omitting empty trailing topics
- **Smart Grouping**: ERC20/ERC721 Receive + ERC1155 Send merged when Direction = Both

## Query Structure Examples

### Send Direction
- **ERC20/ERC721**: `[eventSignature, address]` (2 topics)
- **ERC1155**: `[eventSignature, {}, address]` (3 topics)

### Receive Direction  
- **ERC20/ERC721**: `[eventSignature, {}, address]` (3 topics)
- **ERC1155**: `[eventSignature, {}, {}, address]` (4 topics)

### Both Direction (Optimized)
- **ERC20/ERC721 Send**: `[eventSignature, address]` (2 topics)
- **Merged Receive**: `[eventSignature, {}, address]` (3 topics) - combines ERC20/ERC721 Receive + ERC1155 Send
- **ERC1155 Receive**: `[eventSignature, {}, {}, address]` (4 topics)

## Integration

### With go-ethereum client

```go
import (
    "github.com/ethereum/go-ethereum/ethclient"
    "github.com/status-im/go-wallet-sdk/pkg/eventfilter"
)

// Create client
client, _ := ethclient.Dial("https://mainnet.infura.io/v3/YOUR_KEY")

// Use FilterTransfers for concurrent processing
events, err := eventfilter.FilterTransfers(ctx, client, config)
if err != nil {
    log.Fatal(err)
}

// Process events
for _, event := range events {
    // Access parsed event data via event.Unpacked
}
```

### Manual query execution

```go
// Generate queries manually
queries := config.ToFilterQueries()

// Execute queries sequentially
for _, query := range queries {
    logs, err := client.FilterLogs(context.Background(), query)
    // Process logs...
}
```

## Event Signatures

Uses standardized signatures from the `eventlog` package:
- **ERC20/ERC721 Transfer**: `0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef`
- **ERC1155 TransferSingle**: `0xc3d58168c5ae7397731d063d5bbf3d657854427343f4c083240f7aacaa2d0f62`
- **ERC1155 TransferBatch**: `0x4a39dc06d4c0dbc64b70af90fd698a233a518aa5d07e595d983b8c0526c8f7fb`

## Testing

Comprehensive test suite ensures correct query generation:
- Query count validation for all configurations
- Topic structure verification
- Merged query functionality testing

Run tests with:
```bash
go test ./pkg/transfereventfilter
```