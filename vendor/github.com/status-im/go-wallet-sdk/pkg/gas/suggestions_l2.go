package gas

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum"
)

func getL2Suggestions(ctx context.Context, ethClient EthClient, params ChainParameters, config SuggestionsConfig, callMsg *ethereum.CallMsg) (*TxSuggestions, error) {
	ret := &TxSuggestions{
		GasLimit: big.NewInt(0),
	}

	if callMsg != nil {
		gasLimit, err := ethClient.EstimateGas(ctx, *callMsg)
		if err != nil {
			return nil, fmt.Errorf("failed to estimate gas: %w", err)
		}
		ret.GasLimit = big.NewInt(0).SetUint64(gasLimit)
	}

	blockCount := uint64(max(config.GasPriceEstimationBlocks, config.NetworkCongestionBlocks))
	rewardPercentiles := []float64{config.LowRewardPercentile, config.MediumRewardPercentile, config.HighRewardPercentile}

	feeHistory, err := ethClient.FeeHistory(ctx, blockCount, nil, rewardPercentiles)
	if err != nil {
		return nil, fmt.Errorf("failed to get fee history: %w", err)
	}

	gasPrice, err := suggestGasPrice(feeHistory, config.GasPriceEstimationBlocks)
	if err != nil {
		return nil, fmt.Errorf("failed to suggest gas price: %w", err)
	}

	lowBaseFee := adjustL2BaseFee(gasPrice.BaseFeePerGas, config.LowBaseFeeMultiplier)
	mediumBaseFee := adjustL2BaseFee(gasPrice.BaseFeePerGas, config.MediumBaseFeeMultiplier)
	highBaseFee := adjustL2BaseFee(gasPrice.BaseFeePerGas, config.HighBaseFeeMultiplier)

	ret.FeeSuggestions = &FeeSuggestions{
		EstimatedBaseFee:      gasPrice.BaseFeePerGas,
		NetworkCongestion:     0,
		PriorityFeeLowerBound: gasPrice.LowPriorityFeePerGas,
		PriorityFeeUpperBound: gasPrice.HighPriorityFeePerGas,
		Low: Fee{
			MaxPriorityFeePerGas: gasPrice.LowPriorityFeePerGas,
			MaxFeePerGas:         big.NewInt(0).Add(lowBaseFee, gasPrice.LowPriorityFeePerGas),
		},
		Medium: Fee{
			MaxPriorityFeePerGas: gasPrice.MediumPriorityFeePerGas,
			MaxFeePerGas:         big.NewInt(0).Add(mediumBaseFee, gasPrice.MediumPriorityFeePerGas),
		},
		High: Fee{
			MaxPriorityFeePerGas: gasPrice.HighPriorityFeePerGas,
			MaxFeePerGas:         big.NewInt(0).Add(highBaseFee, gasPrice.HighPriorityFeePerGas),
		},
	}

	// Calculate inclusions
	fillInclusions(ret, feeHistory, params.NetworkBlockTime)

	return ret, nil
}

func adjustL2BaseFee(baseFee *big.Int, baseFeeMultiplier float64) *big.Int {
	adjustedBaseFeeFloat := new(big.Float).SetInt(baseFee)
	adjustedBaseFeeFloat.Mul(adjustedBaseFeeFloat, big.NewFloat(baseFeeMultiplier))

	adjustedBaseFee := new(big.Int)
	adjustedBaseFeeFloat.Int(adjustedBaseFee)
	return adjustedBaseFee
}
