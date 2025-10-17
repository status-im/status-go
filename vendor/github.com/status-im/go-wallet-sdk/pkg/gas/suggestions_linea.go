package gas

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum"
)

func getLineaTxSuggestions(ctx context.Context, ethClient EthClient, params ChainParameters, config SuggestionsConfig, callMsg *ethereum.CallMsg) (*TxSuggestions, error) {
	var gasPrice *GasPrice
	var gasLimit *big.Int

	if callMsg == nil {
		tmpGasPrice, err := suggestLineaGasPriceForAccount(ctx, ethClient, callMsg.From)
		if err != nil {
			return nil, fmt.Errorf("failed to get linea suggestions: %w", err)
		}
		gasPrice = tmpGasPrice
		gasLimit = big.NewInt(0)
	} else {
		lineaEstimateGasResult, err := estimateLineaTxGas(ctx, ethClient, callMsg)
		if err != nil {
			return nil, fmt.Errorf("failed to estimate linea gas: %w", err)
		}

		tmpGasPrice, err := suggestLineaGasPrice(lineaEstimateGasResult)
		if err != nil {
			return nil, fmt.Errorf("failed to get linea suggestions: %w", err)
		}
		gasPrice = tmpGasPrice
		gasLimit = lineaEstimateGasResult.GasLimit
	}

	twiceBaseFee := big.NewInt(0).Mul(gasPrice.BaseFeePerGas, big.NewInt(2))

	ret := &TxSuggestions{
		GasLimit: gasLimit,
		FeeSuggestions: &FeeSuggestions{
			EstimatedBaseFee:      gasPrice.BaseFeePerGas,
			NetworkCongestion:     0,
			PriorityFeeLowerBound: gasPrice.LowPriorityFeePerGas,
			PriorityFeeUpperBound: gasPrice.HighPriorityFeePerGas,
			Low: Fee{
				MaxPriorityFeePerGas: gasPrice.LowPriorityFeePerGas,
				MaxFeePerGas:         big.NewInt(0).Add(twiceBaseFee, gasPrice.LowPriorityFeePerGas),
			},
			Medium: Fee{
				MaxPriorityFeePerGas: gasPrice.MediumPriorityFeePerGas,
				MaxFeePerGas:         big.NewInt(0).Add(twiceBaseFee, gasPrice.MediumPriorityFeePerGas),
			},
			High: Fee{
				MaxPriorityFeePerGas: gasPrice.HighPriorityFeePerGas,
				MaxFeePerGas:         big.NewInt(0).Add(twiceBaseFee, gasPrice.HighPriorityFeePerGas),
			},
		},
	}

	// Calculate inclusions
	blockCount := uint64(config.NetworkCongestionBlocks)
	rewardPercentiles := []float64{config.MediumRewardPercentile}

	feeHistory, err := ethClient.FeeHistory(ctx, blockCount, nil, rewardPercentiles)
	if err != nil {
		return nil, fmt.Errorf("failed to get fee history: %w", err)
	}

	fillInclusionsLinea(ret, feeHistory, params.NetworkBlockTime)

	return ret, nil
}

func fillInclusionsLinea(txSuggestions *TxSuggestions, feeHistory *ethereum.FeeHistory, avgBlockTime float64) {
	sortedBaseFees := getSortedBaseFees(feeHistory)
	sortedMediumPriorityFees := getSortedPriorityFees(feeHistory, 0)

	txSuggestions.FeeSuggestions.LowInclusion = estimateInclusion(txSuggestions.FeeSuggestions.Low, sortedBaseFees, sortedMediumPriorityFees, avgBlockTime)
	txSuggestions.FeeSuggestions.MediumInclusion = estimateInclusion(txSuggestions.FeeSuggestions.Medium, sortedBaseFees, sortedMediumPriorityFees, avgBlockTime)
	txSuggestions.FeeSuggestions.HighInclusion = estimateInclusion(txSuggestions.FeeSuggestions.High, sortedBaseFees, sortedMediumPriorityFees, avgBlockTime)
}
