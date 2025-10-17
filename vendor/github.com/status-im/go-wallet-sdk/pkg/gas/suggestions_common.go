package gas

import (
	"math/big"

	"github.com/ethereum/go-ethereum"
)

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
