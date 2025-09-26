# EventLog

Ethereum event log parser for ERC20, ERC721, and ERC1155 events. Automatically detects and parses token events with type-safe access to event data.

## Supported Events

- **ERC20**: Transfer, Approval
- **ERC721**: Transfer, Approval, ApprovalForAll  
- **ERC1155**: TransferSingle, TransferBatch, ApprovalForAll, URI

## Usage

```go
import (
    "github.com/ethereum/go-ethereum/core/types"
    "github.com/status-im/go-wallet-sdk/pkg/eventlog"
)

// Parse events from a log
log := types.Log{...}
events := eventlog.ParseLog(log)

// Handle each event
for _, event := range events {
    switch event.EventKey {
    case eventlog.ERC20Transfer:
        transfer := event.Unpacked.(erc20.Erc20Transfer)
        fmt.Printf("Transfer: %s -> %s, Amount: %s\n", 
            transfer.From.Hex(), transfer.To.Hex(), transfer.Value.String())
    case eventlog.ERC721Transfer:
        transfer := event.Unpacked.(erc721.Erc721Transfer)
        fmt.Printf("NFT Transfer: %s -> %s, TokenID: %s\n",
            transfer.From.Hex(), transfer.To.Hex(), transfer.TokenId.String())
    case eventlog.ERC1155TransferSingle:
        transfer := event.Unpacked.(erc1155.Erc1155TransferSingle)
        fmt.Printf("ERC1155: %s -> %s, ID: %s, Value: %s\n",
            transfer.From.Hex(), transfer.To.Hex(), transfer.Id.String(), transfer.Value.String())
    }
}
```

## Event Structure

```go
type Event struct {
    ContractKey ContractKey  // "erc20", "erc721", or "erc1155"
    ContractABI *abi.ABI     // Full contract ABI
    EventKey    EventKey     // Specific event type
    ABIEvent    *abi.Event   // ABI event definition
    Unpacked    any          // Type-safe parsed event data
}
```

## Integration with EventFilter

```go
import (
    "github.com/status-im/go-wallet-sdk/pkg/eventfilter"
    "github.com/status-im/go-wallet-sdk/pkg/eventlog"
)

// Filter and parse events
events, err := eventfilter.FilterTransfers(client, config)
if err != nil {
    log.Fatal(err)
}

for _, event := range events {
    parsedEvents := eventlog.ParseLog(event.Raw)
    for _, parsedEvent := range parsedEvents {
        // Process parsed event data
        handleEvent(parsedEvent)
    }
}
```

## Error Handling

Returns empty slice for unknown events or malformed data. Safe to use with any log data.

## Testing

```bash
go test -v ./pkg/eventlog/...
```

See `examples/eventfilter-example` for complete usage example.
