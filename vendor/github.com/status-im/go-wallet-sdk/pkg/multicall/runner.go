package multicall

//go:generate mockgen -destination=mock/caller.go . Caller

import (
	"context"
	"errors"
	"math/big"
	"slices"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"

	"github.com/status-im/go-wallet-sdk/pkg/contracts/multicall3"
)

type Caller interface {
	ViewTryBlockAndAggregate(opts *bind.CallOpts, requireSuccess bool, calls []multicall3.IMulticall3Call) (*big.Int, [32]byte, []multicall3.IMulticall3Result, error)
	ViewTryAggregate(opts *bind.CallOpts, requireSuccess bool, calls []multicall3.IMulticall3Call) ([]multicall3.IMulticall3Result, error)
}

type CallResult struct {
	Value any
	Err   error
}
type JobResult struct {
	Results     []CallResult
	Err         error
	BlockNumber *big.Int
	BlockHash   common.Hash
}

type JobsResult struct {
	JobIdx    int
	JobResult JobResult
}

type Job struct {
	Calls        []multicall3.IMulticall3Call
	CallResultFn func(multicall3.IMulticall3Result) (any, error)
}

// Collects all jobs and runs them in batches in a blocking manner.
// Once finished, returns a JobResult for each job.
// The output JobResult index matches the input Job index.
func RunSync(ctx context.Context, jobs []Job, atBlock *big.Int, caller Caller, batchsize int) []JobResult {
	resultsCh := RunAsync(ctx, jobs, atBlock, caller, batchsize)

	results := make([]JobResult, len(jobs))
	for result := range resultsCh {
		results[result.JobIdx] = result.JobResult
	}

	return results
}

// Collects all jobs and runs them in batches in a non-blocking manner.
// Returns immediately with a channel where a single JobResult will be sent for each job.
// The received JobResult index matches the input Job index.
// The channel is closed when all results have been sent.
func RunAsync(ctx context.Context, jobs []Job, atBlock *big.Int, caller Caller, batchsize int) <-chan JobsResult {
	resultsCh := make(chan JobsResult, len(jobs))

	go func() {
		defer func() {
			close(resultsCh)
		}()

		ProcessJobs(ctx, jobs, resultsCh, atBlock, caller, batchsize)
	}()
	return resultsCh
}

// Collects all jobs and runs them in batches.
// A single JobResult will be sent on each JobRunner's channel,
// as soon as each individual job is finished.
func ProcessJobs(ctx context.Context, jobs []Job, resultsCh chan<- JobsResult, atBlock *big.Int, caller Caller, batchsize int) {
	flatCalls := make([]multicall3.IMulticall3Call, 0, len(jobs))
	for _, job := range jobs {
		flatCalls = append(flatCalls, job.Calls...)
	}

	if len(flatCalls) == 0 {
		// No jobs to run, send empty result for each job
		for i := range jobs {
			resultsCh <- JobsResult{
				JobIdx: i,
				JobResult: JobResult{
					Err: nil,
				},
			}
		}
		return
	}

	rawCallResults := make([]multicall3.IMulticall3Result, 0, len(flatCalls))

	var blockNumber *big.Int
	var blockHash common.Hash
	const requireSuccess = false // Don't revert if any individual call fails
	lastProcessedJobIdx := 0

	// Handle errors
	var err error
	defer func() {
		if err == nil || lastProcessedJobIdx >= len(jobs) {
			return
		}
		// Report error to unprocessed jobs
		for i := range jobs[lastProcessedJobIdx:] {
			resultsCh <- JobsResult{
				JobIdx: i,
				JobResult: JobResult{
					Err: err,
				},
			}
		}
	}()

	first := true
	for chunk := range slices.Chunk(flatCalls, batchsize) {
		var chunkResults []multicall3.IMulticall3Result
		if first {
			first = false
			// First chunk, we need to get the block number and hash
			blockNumber, blockHash, chunkResults, err = caller.ViewTryBlockAndAggregate(&bind.CallOpts{
				Context:     ctx,
				BlockNumber: atBlock,
			}, requireSuccess, chunk)
		} else {
			// Subsequent chunks, we use the block number from the previous chunk
			chunkResults, err = caller.ViewTryAggregate(&bind.CallOpts{
				Context:     ctx,
				BlockNumber: blockNumber,
			}, requireSuccess, chunk)
		}
		if err != nil {
			return
		}
		rawCallResults = append(rawCallResults, chunkResults...)

		// Process results for any finished jobs
		for {
			pendingJob := jobs[lastProcessedJobIdx]
			pendingCallCount := len(pendingJob.Calls)
			if len(rawCallResults) < pendingCallCount {
				break
			}

			jobResult := JobResult{
				BlockNumber: blockNumber,
				BlockHash:   blockHash,
			}

			callResultFn := pendingJob.CallResultFn
			if callResultFn == nil {
				jobResult.Err = errors.New("call result function is nil")
			} else {
				results := make([]CallResult, 0, pendingCallCount)
				for _, rawCallResult := range rawCallResults[:pendingCallCount] {
					result, err := callResultFn(rawCallResult)
					results = append(results, CallResult{
						Value: result,
						Err:   err,
					})
				}
				jobResult.Results = results
			}

			resultsCh <- JobsResult{
				JobIdx:    lastProcessedJobIdx,
				JobResult: jobResult,
			}

			rawCallResults = rawCallResults[pendingCallCount:]
			lastProcessedJobIdx++
			if lastProcessedJobIdx >= len(jobs) {
				break
			}
		}
	}
}
