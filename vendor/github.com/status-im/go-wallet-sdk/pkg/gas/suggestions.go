package gas

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum"
)

// DefaultConfig returns default configuration
func DefaultConfig(chainClass ChainClass) SuggestionsConfig {
	switch chainClass {
	case ChainClassL1:
		return SuggestionsConfig{
			NetworkCongestionBlocks:           10,
			GasPriceEstimationBlocks:          10,
			LowRewardPercentile:               10,
			MediumRewardPercentile:            45,
			HighRewardPercentile:              90,
			LowBaseFeeMultiplier:              1.025, // 2.5% buffer for base fee
			MediumBaseFeeMultiplier:           1.025,
			HighBaseFeeMultiplier:             1.025,
			LowBaseFeeCongestionMultiplier:    0.0, // No congestion-based adjustment for Low level
			MediumBaseFeeCongestionMultiplier: 10.0,
			HighBaseFeeCongestionMultiplier:   10.0,
		}
	}

	return SuggestionsConfig{
		NetworkCongestionBlocks:           10,
		GasPriceEstimationBlocks:          50,
		LowRewardPercentile:               10,
		MediumRewardPercentile:            45,
		HighRewardPercentile:              90,
		LowBaseFeeMultiplier:              1.025,
		MediumBaseFeeMultiplier:           4.1,
		HighBaseFeeMultiplier:             10.25,
		LowBaseFeeCongestionMultiplier:    0.0, // No congestion-based adjustment at any level
		MediumBaseFeeCongestionMultiplier: 0.0,
		HighBaseFeeCongestionMultiplier:   0.0,
	}
}

func GetTxSuggestions(ctx context.Context, ethClient EthClient, params ChainParameters, config SuggestionsConfig, callMsg *ethereum.CallMsg) (*TxSuggestions, error) {
	switch params.ChainClass {
	case ChainClassL1:
		return getL1Suggestions(ctx, ethClient, params, config, callMsg)
	case ChainClassLineaStack:
		return getLineaTxSuggestions(ctx, ethClient, params, config, callMsg)
	}
	return getL2Suggestions(ctx, ethClient, params, config, callMsg)
}

func EstimateInclusion(ctx context.Context, ethClient EthClient, params ChainParameters, config SuggestionsConfig, fee Fee) (*Inclusion, error) {
	blockCount := uint64(max(config.GasPriceEstimationBlocks, config.NetworkCongestionBlocks))
	rewardPercentiles := []float64{config.MediumRewardPercentile}

	feeHistory, err := ethClient.FeeHistory(ctx, blockCount, nil, rewardPercentiles)
	if err != nil {
		return nil, fmt.Errorf("failed to get fee history: %w", err)
	}

	sortedBaseFees := getSortedBaseFees(feeHistory)
	sortedMediumPriorityFees := getSortedPriorityFees(feeHistory, MediumPriorityFeeIndex)

	inclusion := estimateInclusion(fee, sortedBaseFees, sortedMediumPriorityFees, params.NetworkBlockTime)
	return &inclusion, nil
}
