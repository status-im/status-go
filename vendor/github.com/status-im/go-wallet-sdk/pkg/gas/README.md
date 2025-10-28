# Gas Package

A comprehensive gas estimation and fee suggestion package for Ethereum and L2 networks.

## Features

- **Multi-chain support**: Ethereum L1, Arbitrum Stack, Optimism Stack, Linea Stack
- **Smart fee estimation**: Priority fees, base fees, and max fees with inclusion time estimates

## Quick Start

```go
import "github.com/status-im/go-wallet-sdk/pkg/gas"

// Define chain parameters
params := gas.ChainParameters{
    ChainClass:       gas.ChainClassL1,
    NetworkBlockTime: 12.0, // seconds
}

// Get default config for the chain class
config := gas.DefaultConfig(params.ChainClass)

// Create a call message for the transaction
callMsg := &ethereum.CallMsg{
    To:    &common.Address{},
    Data:  []byte{},
    Value: big.NewInt(0),
}

// Get fee suggestions
suggestions, err := gas.GetTxSuggestions(ctx, ethClient, params, config, callMsg)
if err != nil {
    return err
}

// Access fee suggestions
lowFee := suggestions.FeeSuggestions.Low.MaxFeePerGas
lowMaxFee := suggestions.FeeSuggestions.Low.MaxFeePerGas
lowMinTime := suggestions.FeeSuggestions.LowInclusion.MinTimeUntilInclusion
lowMaxTime := suggestions.FeeSuggestions.LowInclusion.MaxTimeUntilInclusion
```

## Chain Classes

- **L1**: Ethereum mainnet, Polygon, BSC
- **ArbStack**: Arbitrum One, Arbitrum Nova
- **OPStack**: Optimism, Base, OP Sepolia
- **LineaStack**: Linea mainnet, Linea testnet

## Configuration

```go
// Chain parameters (required)
params := gas.ChainParameters{
    ChainClass:       gas.ChainClassL1,
    NetworkBlockTime: 12.0, // Average block time in seconds
}

// Suggestions config (use DefaultConfig or customize)
config := gas.DefaultConfig(params.ChainClass)
// Or customize:
config := gas.SuggestionsConfig{
    NetworkCongestionBlocks:           10,    // blocks to analyze for congestion
    GasPriceEstimationBlocks:          10,    // blocks for gas price estimation
    LowRewardPercentile:               10,    // %
    MediumRewardPercentile:            45,    // %
    HighRewardPercentile:              90,    // %
    LowBaseFeeMultiplier:              1.025, // multiplier for low level base fee
    MediumBaseFeeMultiplier:           1.025, // multiplier for medium level base fee
    HighBaseFeeMultiplier:             1.025, // multiplier for high level base fee
    LowBaseFeeCongestionMultiplier:    0.0,   // congestion factor for low (L1 only)
    MediumBaseFeeCongestionMultiplier: 10.0,  // congestion factor for medium (L1 only)
    HighBaseFeeCongestionMultiplier:   10.0,  // congestion factor for high (L1 only)
}
```

## API Methods

### GetChainSuggestions

Get fee suggestions for a specific account without requiring a transaction call message. This method provides general fee recommendations based on network conditions and account-specific factors.

```go
func GetChainSuggestions(
    ctx context.Context,
    ethClient EthClient,
    params ChainParameters,
    config SuggestionsConfig,
    account common.Address,
) (*FeeSuggestions, error)
```

**Parameters:**
- `ctx`: Context for cancellation and timeout
- `ethClient`: Ethereum client implementing `EthClient` interface
- `params`: Chain parameters (class and block time)
- `config`: Configuration for estimation (use `DefaultConfig()` or customize)
- `account`: Account address for account-specific fee suggestions (required for LineaStack)

**Returns:**
- `FeeSuggestions`: Contains fee suggestions for three priority levels with inclusion time estimates
- `error`: Error if fee history retrieval or estimation fails

**Example:**
```go
// Get general fee suggestions for an account
account := common.HexToAddress("0x...")
suggestions, err := gas.GetChainSuggestions(ctx, ethClient, params, config, account)
if err != nil {
    return err
}

// Use medium priority fees
maxPriorityFee := suggestions.Medium.MaxPriorityFeePerGas
maxFee := suggestions.Medium.MaxFeePerGas

// Check estimated wait time
minWait := suggestions.MediumInclusion.MinTimeUntilInclusion
maxWait := suggestions.MediumInclusion.MaxTimeUntilInclusion
```

### GetTxSuggestions

Get comprehensive fee suggestions and gas limit estimation for a transaction.

```go
func GetTxSuggestions(
    ctx context.Context,
    ethClient EthClient,
    params ChainParameters,
    config SuggestionsConfig,
    callMsg *ethereum.CallMsg,
) (*TxSuggestions, error)
```

**Parameters:**
- `ctx`: Context for cancellation and timeout
- `ethClient`: Ethereum client implementing `EthClient` interface
- `params`: Chain parameters (class and block time)
- `config`: Configuration for estimation (use `DefaultConfig()` or customize)
- `callMsg`: Transaction call message (can be `nil` to skip gas limit estimation)

**Returns:**
- `TxSuggestions`: Contains gas limit and fee suggestions for three priority levels
- `error`: Error if fee history retrieval or estimation fails

**Example:**
```go
suggestions, err := gas.GetTxSuggestions(ctx, ethClient, params, config, callMsg)
if err != nil {
    return err
}

// Use medium priority fees
tx := types.NewTx(&types.DynamicFeeTx{
    ChainID:   chainID,
    Nonce:     nonce,
    To:        to,
    Value:     value,
    Gas:       suggestions.GasLimit.Uint64(),
    GasFeeCap: suggestions.FeeSuggestions.Medium.MaxFeePerGas,
    GasTipCap: suggestions.FeeSuggestions.Medium.MaxPriorityFeePerGas,
    Data:      data,
})
```

### EstimateInclusion

Estimate transaction inclusion time for a custom fee configuration.

```go
func EstimateInclusion(
    ctx context.Context,
    ethClient EthClient,
    params ChainParameters,
    config SuggestionsConfig,
    fee Fee,
) (*Inclusion, error)
```

**Parameters:**
- `ctx`: Context for cancellation and timeout
- `ethClient`: Ethereum client implementing `EthClient` interface
- `params`: Chain parameters (class and block time)
- `config`: Configuration for estimation
- `fee`: Custom fee with `MaxPriorityFeePerGas` and `MaxFeePerGas`

**Returns:**
- `Inclusion`: Estimated min/max blocks and time until inclusion
- `error`: Error if fee history retrieval or estimation fails

**Example:**
```go
// Estimate inclusion time for a custom fee
customFee := gas.Fee{
    MaxPriorityFeePerGas: big.NewInt(2000000000), // 2 gwei
    MaxFeePerGas:         big.NewInt(30000000000), // 30 gwei
}

inclusion, err := gas.EstimateInclusion(ctx, ethClient, params, config, customFee)
if err != nil {
    return err
}

fmt.Printf("Estimated inclusion time: %.1f-%.1f seconds\n",
    inclusion.MinTimeUntilInclusion,
    inclusion.MaxTimeUntilInclusion)
fmt.Printf("Estimated blocks: %d-%d blocks\n",
    inclusion.MinBlocksUntilInclusion,
    inclusion.MaxBlocksUntilInclusion)
```

### DefaultConfig

Get default configuration optimized for a specific chain class.

```go
func DefaultConfig(chainClass ChainClass) SuggestionsConfig
```

**Parameters:**
- `chainClass`: Chain class (`ChainClassL1`, `ChainClassArbStack`, `ChainClassOPStack`, `ChainClassLineaStack`)

**Returns:**
- `SuggestionsConfig`: Optimized configuration with appropriate block counts and percentiles

**Example:**
```go
// Get default config for Ethereum L1
config := gas.DefaultConfig(gas.ChainClassL1)
// Returns:
//   - 10 blocks for congestion, 10 blocks for estimation
//   - Base fee multipliers: 1.025x for all levels
//   - Congestion multipliers: 0.0x (low), 10.0x (medium), 10.0x (high)

// Get default config for L2 chains (OPStack, ArbStack)
l2Config := gas.DefaultConfig(gas.ChainClassOPStack)
// Returns:
//   - 10 blocks for congestion, 50 blocks for estimation
//   - Base fee multipliers: 1.025x (low), 4.1x (medium), 10.25x (high)
//   - Congestion multipliers: 0.0x for all levels (no congestion on L2)
```

## Output Types

### TxSuggestions

```go
type TxSuggestions struct {
    GasLimit       *big.Int        // Estimated gas limit
    FeeSuggestions *FeeSuggestions // Fee suggestions
}
```

### FeeSuggestions

```go
type FeeSuggestions struct {
    // Fee suggestions for three priority levels
    Low    Fee // Low priority fees
    Medium Fee // Medium priority fees
    High   Fee // High priority fees
    
    // Inclusion time estimates
    LowInclusion    Inclusion // Low priority inclusion estimate
    MediumInclusion Inclusion // Medium priority inclusion estimate
    HighInclusion   Inclusion // High priority inclusion estimate
    
    // Network state
    EstimatedBaseFee      *big.Int // Next block's estimated base fee (wei)
    PriorityFeeLowerBound *big.Int // Recommended min priority fee (wei)
    PriorityFeeUpperBound *big.Int // Recommended max priority fee (wei)
    NetworkCongestion     float64  // Congestion score 0-1 (L1 only)
}
```

### Fee

```go
type Fee struct {
    MaxPriorityFeePerGas *big.Int // Max priority fee per gas (wei)
    MaxFeePerGas         *big.Int // Max fee per gas (wei)
}
```

### Inclusion

```go
type Inclusion struct {
    MinBlocksUntilInclusion int     // Minimum blocks until inclusion
    MaxBlocksUntilInclusion int     // Maximum blocks until inclusion
    MinTimeUntilInclusion   float64 // Minimum time in seconds
    MaxTimeUntilInclusion   float64 // Maximum time in seconds
}
```
