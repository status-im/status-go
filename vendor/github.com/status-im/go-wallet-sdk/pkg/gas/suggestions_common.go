package gas

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum"
)

// Fetch fee history and check result correctness
func getFeeHistory(ctx context.Context, ethClient EthClient, blockCount uint64, lastBlock *big.Int, rewardPercentiles []float64) (*ethereum.FeeHistory, error) {
	feeHistory, err := ethClient.FeeHistory(ctx, blockCount, lastBlock, rewardPercentiles)
	if err != nil {
		return nil, err
	}

	if feeHistory == nil {
		return nil, fmt.Errorf("fee history is nil")
	}

	if len(feeHistory.BaseFee) < int(blockCount)+1 {
		return nil, fmt.Errorf("baseFee length is less than %d", blockCount+1)
	}

	if len(feeHistory.Reward) < int(blockCount) {
		return nil, fmt.Errorf("reward length is less than %d", blockCount)
	}

	for i, reward := range feeHistory.Reward {
		if len(reward) < len(rewardPercentiles) {
			return nil, fmt.Errorf("reward %d length is less than %d", i, len(rewardPercentiles))
		}
	}

	return feeHistory, nil
}

// For any chain except Linea Stack
func fillInclusions(txSuggestions *TxSuggestions, feeHistory *ethereum.FeeHistory, avgBlockTime float64) {
	sortedBaseFees := getSortedBaseFees(feeHistory)
	sortedMediumPriorityFees := getSortedPriorityFees(feeHistory, MediumPriorityFeeIndex)
	txSuggestions.FeeSuggestions.LowInclusion = estimateInclusion(txSuggestions.FeeSuggestions.Low, sortedBaseFees, sortedMediumPriorityFees, avgBlockTime)
	txSuggestions.FeeSuggestions.MediumInclusion = estimateInclusion(txSuggestions.FeeSuggestions.Medium, sortedBaseFees, sortedMediumPriorityFees, avgBlockTime)
	txSuggestions.FeeSuggestions.HighInclusion = estimateInclusion(txSuggestions.FeeSuggestions.High, sortedBaseFees, sortedMediumPriorityFees, avgBlockTime)
}

func estimateInclusion(feeSugestion Fee, sortedBaseFees []*big.Int, sortedMediumPriorityFees []*big.Int, avgBlockTime float64) Inclusion {
	minBlocks, maxBlocks := estimateBlocksUntilInclusion(feeSugestion.MaxPriorityFeePerGas, feeSugestion.MaxFeePerGas, sortedBaseFees, sortedMediumPriorityFees)

	return Inclusion{
		MinBlocksUntilInclusion: minBlocks,
		MaxBlocksUntilInclusion: maxBlocks,
		MinTimeUntilInclusion:   float64(minBlocks) * avgBlockTime,
		MaxTimeUntilInclusion:   float64(maxBlocks) * avgBlockTime,
	}
}
