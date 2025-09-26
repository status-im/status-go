# Multicall Package

Efficiently batch multiple Ethereum contract calls into single transactions using Multicall3.

## Quick Start

```go
import "github.com/status-im/go-wallet-sdk/pkg/multicall"

// Build calls
calls := []multicall3.IMulticall3Call{
    multicall.BuildNativeBalanceCall(account, multicall3Addr),
    multicall.BuildERC20BalanceCall(account, tokenAddr),
    multicall.BuildERC721BalanceCall(account, nftAddr),
}

// Create job with call result function
job := multicall.Job{
    Calls: calls,
    CallResultFn: func(result multicall3.IMulticall3Result) (any, error) {
        return multicall.ProcessNativeBalanceResult(result)
    },
}

// Execute synchronously
results := multicall.RunSync(ctx, []multicall.Job{job}, blockNumber, caller, batchSize)

// Process results
if len(results) > 0 && len(results[0].Results) > 0 {
    balance, err := results[0].Results[0].Value.(*big.Int), results[0].Results[0].Err
}
```

## Features

- **Batch Execution**: Combine multiple calls into single transactions
- **Token Support**: Native ETH, ERC20, ERC721, ERC1155 balance queries
- **Chunked Processing**: Automatic batching for large call sets
- **Error Handling**: Graceful failure handling with detailed error reporting
- **Async Support**: Both synchronous and asynchronous execution modes
- **Job-based API**: Flexible job system with custom result processing functions

## API

### Types
- `Job` - Contains calls and a result processing function
- `JobResult` - Contains processed results, block info, and errors
- `CallResult` - Individual call result with value and error
- `Caller` - Interface for executing multicall operations

### Call Builders
- `BuildNativeBalanceCall()` - Get ETH balance
- `BuildERC20BalanceCall()` - Get ERC20 token balance  
- `BuildERC721BalanceCall()` - Get ERC721 NFT balance
- `BuildERC1155BalanceCall()` - Get ERC1155 token balance

### Execution
- `RunSync()` - Execute jobs synchronously, returns `[]JobResult`
- `RunAsync()` - Execute jobs asynchronously, returns channel of `JobsResult`
- `ProcessJobs()` - Internal function for processing jobs

### Result Processing
- `ProcessNativeBalanceResult()` - Parse ETH balance from result
- `ProcessERC20BalanceResult()` - Parse ERC20 balance from result
- `ProcessERC721BalanceResult()` - Parse ERC721 balance from result
- `ProcessERC1155BalanceResult()` - Parse ERC1155 balance from result

## Example

```go
// Multiple token balances in one call
nativeCalls := []multicall3.IMulticall3Call{
    multicall.BuildNativeBalanceCall(account1, multicall3Addr),
    multicall.BuildNativeBalanceCall(account2, multicall3Addr),
    multicall.BuildNativeBalanceCall(account3, multicall3Addr),
}
tokenCalls := []multicall3.IMulticall3Call{
    multicall.BuildERC20BalanceCall(account1, usdcAddr),
    multicall.BuildERC20BalanceCall(account1, daiAddr),
    multicall.BuildERC20BalanceCall(account2, usdcAddr),
    multicall.BuildERC20BalanceCall(account2, daiAddr),
    multicall.BuildERC20BalanceCall(account3, usdcAddr),
    multicall.BuildERC20BalanceCall(account3, daiAddr),
}

// Create jobs with appropriate result processing functions
jobs := []multicall.Job{
    {
        Calls: nativeCalls,
        CallResultFn: func(result multicall3.IMulticall3Result) (any, error) {
            return multicall.ProcessNativeBalanceResult(result)
        },
    },
    {
        Calls: tokenCalls,
        CallResultFn: func(result multicall3.IMulticall3Result) (any, error) {
            return multicall.ProcessERC20BalanceResult(result)
        },
    },
}

results := multicall.RunSync(ctx, jobs, blockNum, caller, 100)

// Process native balances
for _, callResult := range results[0].Results {
    if callResult.Err != nil {
        // Handle error
        continue
    }
    nativeBalance := callResult.Value.(*big.Int)
    // Do something with balance
}

// Process token balances
for _, callResult := range results[1].Results {
    if callResult.Err != nil {
        // Handle error
        continue
    }
    tokenBalance := callResult.Value.(*big.Int)
    // Do something with balance
}

// Access block information
blockNumber := results[0].BlockNumber
blockHash := results[0].BlockHash
```

## Async Example

```go
// Execute jobs asynchronously
resultsCh := multicall.RunAsync(ctx, jobs, blockNum, caller, 100)

// Process results as they come in
for result := range resultsCh {
    jobIdx := result.JobIdx
    jobResult := result.JobResult
    
    if jobResult.Err != nil {
        // Handle job-level error
        continue
    }
    
    // Process individual call results
    for _, callResult := range jobResult.Results {
        if callResult.Err != nil {
            // Handle call-level error
            continue
        }
        
        // Process the result based on job type
        switch jobIdx {
        case 0: // Native balance job
            balance := callResult.Value.(*big.Int)
            // Process native balance
        case 1: // Token balance job
            balance := callResult.Value.(*big.Int)
            // Process token balance
        }
    }
}
```
