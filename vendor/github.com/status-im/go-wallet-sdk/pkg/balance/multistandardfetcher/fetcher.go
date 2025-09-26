package multistandardfetcher

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/common"

	"github.com/status-im/go-wallet-sdk/pkg/multicall"
)

type FetchConfig struct {
	Native  []AccountAddress
	ERC20   map[AccountAddress][]ContractAddress
	ERC721  map[AccountAddress][]ContractAddress
	ERC1155 map[AccountAddress][]CollectibleID
}

type ResultType string

const (
	ResultTypeNative  ResultType = "native"
	ResultTypeERC20   ResultType = "erc20"
	ResultTypeERC721  ResultType = "erc721"
	ResultTypeERC1155 ResultType = "erc1155"
)

type Result struct {
	Account       AccountAddress
	Result        *big.Int
	Err           error
	AtBlockNumber *big.Int
	AtBlockHash   common.Hash
}

type Results[T comparable] struct {
	Account       AccountAddress
	Results       map[T]*big.Int
	Err           error
	AtBlockNumber *big.Int
	AtBlockHash   common.Hash
}

type NativeResult = Result
type ERC20Result = Results[ContractAddress]
type ERC721Result = Results[ContractAddress]
type ERC1155Result = Results[HashableCollectibleID]

type FetchResult struct {
	ResultType ResultType
	Result     any
}

// Fetches balances asynchronously using Multicall3 batched calls.
// Returns a channel where a FetchResult will be sent for each account address specified in the FetchConfig.
// The channel is closed when all results have been sent.
func FetchBalances(ctx context.Context, multicall3Address common.Address, caller multicall.Caller, config FetchConfig, batchSize int) <-chan FetchResult {
	// One job per account
	nativeJobCount := len(config.Native)
	erc20JobCount := len(config.ERC20)
	erc721JobCount := len(config.ERC721)
	erc1155JobCount := len(config.ERC1155)
	jobCount := nativeJobCount + erc20JobCount + erc721JobCount + erc1155JobCount

	resultsCh := make(chan FetchResult, jobCount)
	jobs := make([]multicall.Job, 0, jobCount)

	type jobResultProcessor func(multicall.JobResult) FetchResult
	jobResultProcessors := make([]jobResultProcessor, 0, jobCount)

	for _, account := range config.Native {
		job := buildNativeJob(account, multicall3Address)
		jobs = append(jobs, job)
		jobResultProcessors = append(jobResultProcessors, func(jobResult multicall.JobResult) FetchResult {
			return FetchResult{
				ResultType: ResultTypeNative,
				Result:     processNativeJobResult(account, jobResult),
			}
		})
	}

	for account, contractAddresses := range config.ERC20 {
		job := buildERC20Job(account, contractAddresses)
		jobs = append(jobs, job)
		jobResultProcessors = append(jobResultProcessors, func(jobResult multicall.JobResult) FetchResult {
			return FetchResult{
				ResultType: ResultTypeERC20,
				Result:     processERC20JobResult(account, contractAddresses, jobResult),
			}
		})
	}

	for account, contractAddresses := range config.ERC721 {
		job := buildERC721Job(account, contractAddresses)
		jobs = append(jobs, job)
		jobResultProcessors = append(jobResultProcessors, func(jobResult multicall.JobResult) FetchResult {
			return FetchResult{
				ResultType: ResultTypeERC721,
				Result:     processERC721JobResult(account, contractAddresses, jobResult),
			}
		})
	}

	for account, tokens := range config.ERC1155 {
		job := buildERC1155JobRunner(account, tokens)
		jobs = append(jobs, job)
		jobResultProcessors = append(jobResultProcessors, func(jobResult multicall.JobResult) FetchResult {
			return FetchResult{
				ResultType: ResultTypeERC1155,
				Result:     processERC1155JobResult(account, tokens, jobResult),
			}
		})
	}

	jobResultsCh := multicall.RunAsync(ctx, jobs, nil, caller, batchSize)

	go func() {
		defer close(resultsCh)
		// jobResultsCh is closed when all results have been sent, causing the
		// loop to exit and the goroutine to end.
		for jobResult := range jobResultsCh {
			resultsCh <- jobResultProcessors[jobResult.JobIdx](jobResult.JobResult)
		}
	}()

	return resultsCh
}
