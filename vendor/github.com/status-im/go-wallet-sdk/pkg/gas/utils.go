package gas

import (
	"math"
	"math/big"
	"slices"

	"github.com/ethereum/go-ethereum"
)

// getPercentile calculates the value at a given percentile from sorted data
func getPercentile(sortedData []*big.Int, percentile float64) *big.Int {
	if len(sortedData) == 0 {
		return big.NewInt(0)
	}

	n := len(sortedData)

	// Handle edge cases
	if percentile <= 0 {
		return new(big.Int).Set(sortedData[0])
	}
	if percentile >= 100 {
		return new(big.Int).Set(sortedData[n-1])
	}

	// Calculate the rank using nearest-rank method
	rank := math.Ceil(percentile / 100.0 * float64(n))

	// Convert to 0-based index
	index := int(rank) - 1
	index = max(index, 0)
	index = min(index, n-1)

	return new(big.Int).Set(sortedData[index])
}

// extractLastPriorityFeesFromHistory extracts  priority fees for a given rewardsPercentileIndex from the last nBlocks of the fee history
func extractLastPriorityFeesFromHistory(feeHistory *ethereum.FeeHistory, rewardsPercentileIndex int, nBlocks int) []*big.Int {
	priorityFees := make([]*big.Int, 0, nBlocks)

	startIdx := max(len(feeHistory.Reward)-nBlocks, 0)
	for _, blockRewards := range feeHistory.Reward[startIdx:] {
		if rewardsPercentileIndex < len(blockRewards) && blockRewards[rewardsPercentileIndex] != nil {
			priorityFees = append(priorityFees, new(big.Int).Set(blockRewards[rewardsPercentileIndex]))
		}
	}

	return priorityFees
}

// getSortedBaseFees sorts base fees from fee history
func getSortedBaseFees(feeHistory *ethereum.FeeHistory) []*big.Int {
	sortedBaseFees := make([]*big.Int, len(feeHistory.BaseFee[:len(feeHistory.BaseFee)-2]))
	copy(sortedBaseFees, feeHistory.BaseFee[:len(feeHistory.BaseFee)-2])
	slices.SortFunc(sortedBaseFees, func(a, b *big.Int) int {
		return a.Cmp(b)
	})
	return sortedBaseFees
}

// getSortedPriorityFees sorts priority fees from fee history
func getSortedPriorityFees(feeHistory *ethereum.FeeHistory, rewardsPercentileIndex int) []*big.Int {
	priorityFees := make([]*big.Int, 0)
	for _, blockRewards := range feeHistory.Reward {
		if rewardsPercentileIndex < len(blockRewards) && blockRewards[rewardsPercentileIndex] != nil {
			priorityFees = append(priorityFees, new(big.Int).Set(blockRewards[rewardsPercentileIndex]))
		}
	}
	return priorityFees
}
