package multistandardfetcher

import (
	"errors"
	"math/big"
	"strconv"

	"github.com/status-im/go-wallet-sdk/pkg/contracts/multicall3"
	"github.com/status-im/go-wallet-sdk/pkg/multicall"
)

func buildERC721Job(account AccountAddress, contractAddresses []ContractAddress) multicall.Job {
	job := multicall.Job{
		Calls: make([]multicall3.IMulticall3Call, 0, len(contractAddresses)),
		CallResultFn: func(result multicall3.IMulticall3Result) (any, error) {
			return multicall.ProcessERC721BalanceResult(result)
		},
	}
	for _, contractAddress := range contractAddresses {
		job.Calls = append(job.Calls, multicall.BuildERC721BalanceCall(account, contractAddress))
	}
	return job
}

func processERC721JobResult(account AccountAddress, contractAddresses []ContractAddress, jobResult multicall.JobResult) (result ERC721Result) {
	result = ERC721Result{
		Account: account,
		Results: make(map[ContractAddress]*big.Int),
	}
	if jobResult.Err != nil {
		result.Err = jobResult.Err
		return
	}
	result.AtBlockNumber = jobResult.BlockNumber
	result.AtBlockHash = jobResult.BlockHash

	if len(jobResult.Results) != len(contractAddresses) {
		result.Err = errors.New("expected " + strconv.Itoa(len(contractAddresses)) + " call results, got " + strconv.Itoa(len(jobResult.Results)))
		return
	}

	for i, callResult := range jobResult.Results {
		if callResult.Err != nil {
			continue
		}
		parsedResult, ok := callResult.Value.(*big.Int)
		if !ok || parsedResult == nil {
			continue
		}
		result.Results[contractAddresses[i]] = parsedResult
	}

	return
}
