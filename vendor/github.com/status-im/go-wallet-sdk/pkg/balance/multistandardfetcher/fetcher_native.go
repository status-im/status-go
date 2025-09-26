package multistandardfetcher

import (
	"errors"
	"math/big"
	"strconv"

	"github.com/ethereum/go-ethereum/common"

	"github.com/status-im/go-wallet-sdk/pkg/contracts/multicall3"
	"github.com/status-im/go-wallet-sdk/pkg/multicall"
)

func buildNativeJob(account AccountAddress, multicall3Address common.Address) multicall.Job {
	return multicall.Job{
		Calls: []multicall3.IMulticall3Call{multicall.BuildNativeBalanceCall(account, multicall3Address)},
		CallResultFn: func(result multicall3.IMulticall3Result) (any, error) {
			return multicall.ProcessNativeBalanceResult(result)
		},
	}
}

func processNativeJobResult(account AccountAddress, jobResult multicall.JobResult) (result NativeResult) {
	result = NativeResult{
		Account: account,
	}
	if jobResult.Err != nil {
		result.Err = jobResult.Err
		return
	}
	result.AtBlockNumber = jobResult.BlockNumber
	result.AtBlockHash = jobResult.BlockHash

	if len(jobResult.Results) != 1 {
		result.Err = errors.New("expected 1 call result, got " + strconv.Itoa(len(jobResult.Results)))
		return
	}

	callResult := jobResult.Results[0]

	if callResult.Err != nil {
		result.Err = callResult.Err
		return
	}

	parsedResult, ok := callResult.Value.(*big.Int)
	if !ok || parsedResult == nil {
		result.Err = errors.New("could not parse call result")
		return
	}

	result.Result = parsedResult
	return
}
