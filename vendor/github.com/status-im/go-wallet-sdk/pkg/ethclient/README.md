# ethclient

Ethereum JSON-RPC client with two method sets:

- **Custom implementations**: Chain-agnostic, works with any EVM chain
- **Go-ethereum compatible**: Same interface as `ethclient`

## Quick Start

```go
rpcClient, _ := rpc.Dial("https://mainnet.infura.io/v3/YOUR-PROJECT-ID")
client := ethclient.NewClient(rpcClient)

blockNumber, _ := client.BlockNumber(ctx)
balance, _ := client.BalanceAt(ctx, address, nil)
```

## Key Benefits

- **Chain-agnostic**: Works with Arbitrum, Optimism, Polygon, etc.
- **47 JSON-RPC methods**: Complete Ethereum API coverage
- **Drop-in replacement**: Same method names as go-ethereum's ethclient (not chain-agnostic)
- **Custom types**: Clean, exportable response types

## Method Types

```go
// Go-ethereum compatible methods (requires go-ethereum RPC client, some methods only work on L1)
client.BlockNumber(ctx)
client.BalanceAt(ctx, address, nil)

// Custom implementations (works with any RPC client and any EVM-compatible chain)
client.EthBlockNumber(ctx)
client.EthGetBalance(ctx, address, nil)
```

## Migration

```go
// Before
ethClient, _ := ethclient.Dial(url)
balance, _ := ethClient.BalanceAt(ctx, address, nil)

// After
rpcClient, _ := rpc.Dial(url)
client := ethclient.NewClient(rpcClient)
balance, _ := client.BalanceAt(ctx, address, nil) // Same API!
```

## Examples

```bash
cd examples/ethclient-usage && go run main.go
``` 