package gas

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
)

func getLineaChainSuggestions(ctx context.Context, ethClient EthClient, params ChainParameters, config SuggestionsConfig, account common.Address) (*FeeSuggestions, error) {
	gasPrice, err := suggestLineaGasPriceForAccount(ctx, ethClient, account)
	if err != nil {
		return nil, fmt.Errorf("failed to get linea suggestions: %w", err)
	}

	txSuggestions, err := calculateLineaTxSuggestions(ctx, ethClient, params, config, gasPrice, big.NewInt(0))
	if err != nil {
		return nil, fmt.Errorf("failed to calculate linea tx suggestions: %w", err)
	}

	return txSuggestions.FeeSuggestions, nil
}

func getLineaTxSuggestions(ctx context.Context, ethClient EthClient, params ChainParameters, config SuggestionsConfig, callMsg *ethereum.CallMsg) (*TxSuggestions, error) {
	lineaEstimateGasResult, err := estimateLineaTxGas(ctx, ethClient, callMsg)
	if err != nil {
		return nil, fmt.Errorf("failed to estimate linea gas: %w", err)
	}

	gasPrice, err := suggestLineaGasPrice(lineaEstimateGasResult)
	if err != nil {
		return nil, fmt.Errorf("failed to get linea suggestions: %w", err)
	}

	return calculateLineaTxSuggestions(ctx, ethClient, params, config, gasPrice, lineaEstimateGasResult.GasLimit)
}

func calculateLineaTxSuggestions(ctx context.Context, ethClient EthClient, params ChainParameters, config SuggestionsConfig, gasPrice *GasPrice, gasLimit *big.Int) (*TxSuggestions, error) {
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

	feeHistory, err := getFeeHistory(ctx, ethClient, blockCount, nil, rewardPercentiles)
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
