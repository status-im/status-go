package gas

import (
	"fmt"

	"github.com/ethereum/go-ethereum"
)

func suggestGasPrice(feeHistory *ethereum.FeeHistory, nBlocks int) (*GasPrice, error) {
	if feeHistory == nil || len(feeHistory.BaseFee) == 0 {
		return nil, fmt.Errorf("fee history is nil or baseFee is empty")
	}

	estimatedBaseFee := feeHistory.BaseFee[len(feeHistory.BaseFee)-1]

	low, err := calculatePriorityFeeFromHistory(feeHistory, nBlocks, LowPriorityFeeIndex)
	if err != nil {
		return nil, err
	}
	medium, err := calculatePriorityFeeFromHistory(feeHistory, nBlocks, MediumPriorityFeeIndex)
	if err != nil {
		return nil, err
	}
	high, err := calculatePriorityFeeFromHistory(feeHistory, nBlocks, HighPriorityFeeIndex)
	if err != nil {
		return nil, err
	}

	// It shouldn't normally happen, but make sure priority fees increase with level
	if low.Cmp(medium) > 0 {
		medium.Set(low)
	}
	if medium.Cmp(high) > 0 {
		high.Set(medium)
	}

	return &GasPrice{
		BaseFeePerGas:           estimatedBaseFee,
		LowPriorityFeePerGas:    low,
		MediumPriorityFeePerGas: medium,
		HighPriorityFeePerGas:   high,
	}, nil
}
