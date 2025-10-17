package gas

import (
	"fmt"
	"math/big"
	"slices"

	"github.com/ethereum/go-ethereum"
)

// calculatePriorityFeeFromHistory calculates priority fee for a given percentile from fee history
func calculatePriorityFeeFromHistory(feeHistory *ethereum.FeeHistory, nBlocks int, rewardsPercentileIndex int) (*big.Int, error) {
	priorityFees := extractLastPriorityFeesFromHistory(feeHistory, rewardsPercentileIndex, nBlocks)
	return calculatePriorityFee(priorityFees)
}

func calculatePriorityFee(priorityFees []*big.Int) (*big.Int, error) {
	if len(priorityFees) == 0 {
		return nil, fmt.Errorf("no priority fees found")
	}

	// Calculate median of the collected priority fees for stability
	sortedFees := make([]*big.Int, len(priorityFees))
	copy(sortedFees, priorityFees)
	slices.SortFunc(sortedFees, func(a, b *big.Int) int {
		return a.Cmp(b)
	})

	n := len(sortedFees)
	medianIndex := n / 2
	medianFee := new(big.Int).Set(sortedFees[medianIndex])
	if n%2 == 0 {
		// If there are an even number of fees, take the average of the two middle values
		medianFee = new(big.Int).Add(medianFee, sortedFees[medianIndex-1])
		medianFee.Div(medianFee, big.NewInt(2))
	}

	if medianFee.Sign() > 0 {
		return medianFee, nil
	}

	// Median fee is 0, return the minimum non-zero fee
	for _, fee := range sortedFees[medianIndex:] {
		if fee.Sign() > 0 {
			return fee, nil
		}
	}

	// All fees are 0, return 0
	return big.NewInt(0), nil
}
