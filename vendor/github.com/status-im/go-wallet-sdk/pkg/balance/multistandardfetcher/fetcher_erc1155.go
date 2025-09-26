package multistandardfetcher

import (
	"errors"
	"math/big"
	"strconv"

	"github.com/status-im/go-wallet-sdk/pkg/contracts/multicall3"
	"github.com/status-im/go-wallet-sdk/pkg/multicall"
)

func buildERC1155JobRunner(account AccountAddress, tokens []CollectibleID) multicall.Job {
	job := multicall.Job{
		Calls: make([]multicall3.IMulticall3Call, 0, len(tokens)),
		CallResultFn: func(result multicall3.IMulticall3Result) (any, error) {
			return multicall.ProcessERC1155BalanceResult(result)
		},
	}
	for _, token := range tokens {
		job.Calls = append(job.Calls, multicall.BuildERC1155BalanceCall(account, token.ContractAddress, token.TokenID))
	}
	return job
}

func processERC1155JobResult(account AccountAddress, tokens []CollectibleID, jobResult multicall.JobResult) (result ERC1155Result) {
	result = ERC1155Result{
		Account: account,
		Results: make(map[HashableCollectibleID]*big.Int),
	}
	if jobResult.Err != nil {
		result.Err = jobResult.Err
		return
	}
	result.AtBlockNumber = jobResult.BlockNumber
	result.AtBlockHash = jobResult.BlockHash

	if len(jobResult.Results) != len(tokens) {
		result.Err = errors.New("expected " + strconv.Itoa(len(tokens)) + " call results, got " + strconv.Itoa(len(jobResult.Results)))
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
		result.Results[tokens[i].ToHashableCollectibleID()] = parsedResult
	}

	return
}
