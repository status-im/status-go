package gas

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum"
)

func getL1ChainSuggestions(ctx context.Context, ethClient EthClient, params ChainParameters, config SuggestionsConfig) (*FeeSuggestions, error) {
	txSuggestions, err := getL1TxSuggestions(ctx, ethClient, params, config, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get l1 tx suggestions: %w", err)
	}

	return txSuggestions.FeeSuggestions, nil
}

func getL1TxSuggestions(ctx context.Context, ethClient EthClient, params ChainParameters, config SuggestionsConfig, callMsg *ethereum.CallMsg) (*TxSuggestions, error) {
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

	feeHistory, err := getFeeHistory(ctx, ethClient, blockCount, nil, rewardPercentiles)
	if err != nil {
		return nil, fmt.Errorf("failed to get fee history: %w", err)
	}

	gasPrice, err := suggestGasPrice(feeHistory, config.GasPriceEstimationBlocks)
	if err != nil {
		return nil, fmt.Errorf("failed to suggest gas price: %w", err)
	}

	networkCongestion := calculateNetworkCongestionFromHistory(feeHistory, config.NetworkCongestionBlocks)

	lowBaseFee := adjustL1BaseFee(gasPrice.BaseFeePerGas, networkCongestion, config.LowBaseFeeMultiplier, config.LowBaseFeeCongestionMultiplier)
	mediumBaseFee := adjustL1BaseFee(gasPrice.BaseFeePerGas, networkCongestion, config.MediumBaseFeeMultiplier, config.MediumBaseFeeCongestionMultiplier)
	highBaseFee := adjustL1BaseFee(gasPrice.BaseFeePerGas, networkCongestion, config.HighBaseFeeMultiplier, config.HighBaseFeeCongestionMultiplier)

	ret.FeeSuggestions = &FeeSuggestions{
		EstimatedBaseFee:      gasPrice.BaseFeePerGas,
		NetworkCongestion:     networkCongestion,
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

func adjustL1BaseFee(baseFee *big.Int, congestion float64, baseFeeMultiplier float64, baseFeeCongestionMultiplier float64) *big.Int {
	baseFeeFloat := new(big.Float).SetInt(baseFee)
	baseFeeFloat.Mul(baseFeeFloat, big.NewFloat(baseFeeMultiplier))

	congestionFactor := new(big.Float).SetFloat64(baseFeeCongestionMultiplier * congestion)
	additionalBasedOnCongestion := new(big.Float)
	additionalBasedOnCongestion.Mul(baseFeeFloat, congestionFactor)

	adjustedBaseFeeFloat := new(big.Float)
	adjustedBaseFeeFloat.Add(baseFeeFloat, additionalBasedOnCongestion)

	adjustedBaseFee := new(big.Int)
	adjustedBaseFeeFloat.Int(adjustedBaseFee)
	return adjustedBaseFee
}
