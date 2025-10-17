package gas

import (
	"math/big"
)

const (
	priorityFeePercentileHigh   = 30.0
	priorityFeePercentileMedium = 20.0
	priorityFeePercentileLow    = 10.0

	baseFeePercentileSecondBlock = 45.0
	baseFeePercentileThirdBlock  = 35.0
	baseFeePercentileFourthBlock = 30.0
	baseFeePercentileFifthBlock  = 20.0
	baseFeePercentileSixthBlock  = 10.0
)

// estimateBlocksUntilInclusion estimates when a transaction will be included based on its fee
// and current network conditions
// returned value for maxBlocks is -1 if the upper bound is unknown
func estimateBlocksUntilInclusion(
	priorityFee *big.Int,
	maxFeePerGas *big.Int,
	sortedBaseFees []*big.Int,
	sortedMediumPriorityFees []*big.Int,
) (minBlocks int, maxBlocks int) {
	maxBaseFee := new(big.Int).Sub(maxFeePerGas, priorityFee)

	priorityFeeForFirstTwoBlock := new(big.Int)
	priorityFeeForSecondTwoBlocks := new(big.Int)
	priorityFeeForThirdTwoBlocks := new(big.Int)
	if len(sortedMediumPriorityFees) > 0 {
		priorityFeeForFirstTwoBlock = getPercentile(sortedMediumPriorityFees, priorityFeePercentileHigh)
		priorityFeeForSecondTwoBlocks = getPercentile(sortedMediumPriorityFees, priorityFeePercentileMedium)
		priorityFeeForThirdTwoBlocks = getPercentile(sortedMediumPriorityFees, priorityFeePercentileLow)
	}

	// To include the transaction in the block `inclusionInBlock` its base fee has to be in a higher than `baseFeePercentile`
	// and its priority fee has to be higher than the `priorityFee`
	inclusions := []struct {
		inclusionInBlock  int
		baseFeePercentile float64
		priorityFee       *big.Int
	}{
		{2, baseFeePercentileSecondBlock, priorityFeeForFirstTwoBlock},
		{3, baseFeePercentileThirdBlock, priorityFeeForSecondTwoBlocks},
		{4, baseFeePercentileFourthBlock, priorityFeeForSecondTwoBlocks},
		{5, baseFeePercentileFifthBlock, priorityFeeForThirdTwoBlocks},
		{6, baseFeePercentileSixthBlock, priorityFeeForThirdTwoBlocks},
	}

	inclusionIdx := -1
	for idx, p := range inclusions {
		baseFeePercentileIndex := getPercentile(sortedBaseFees, p.baseFeePercentile)
		if maxBaseFee.Cmp(baseFeePercentileIndex) >= 0 && priorityFee.Cmp(p.priorityFee) >= 0 {
			inclusionIdx = idx
			break
		}
	}

	minBlocks = 1
	maxBlocks = inclusions[len(inclusions)-1].inclusionInBlock + 1

	if inclusionIdx < 0 {
		minBlocks = inclusions[len(inclusions)-1].inclusionInBlock
		return
	}

	if inclusionIdx == 0 {
		maxBlocks = inclusions[0].inclusionInBlock
		return
	}

	minBlocks = inclusions[inclusionIdx-1].inclusionInBlock
	maxBlocks = inclusions[inclusionIdx].inclusionInBlock
	return
}
