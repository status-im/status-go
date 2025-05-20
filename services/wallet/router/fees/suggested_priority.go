package fees

import (
	"context"
	"errors"
	"math/big"
	"sort"
	"strings"

	"github.com/ethereum/go-ethereum/common/hexutil"
)

const (
	// To get closer to MM values, we don't want to increase the base fee
	baseFeeIncreaseFactor = 1.025 // 2.5% increase

	priorityWeight = 0.7
	gasUsedWeight  = 0.3

	blocksToCheck = 5 // Number of blocks to check for estimating time in case of non-EIP1559 chains
)

func hexStringToBigInt(value string) (*big.Int, error) {
	valueWitoutPrefix := strings.TrimPrefix(value, "0x")
	val, success := new(big.Int).SetString(valueWitoutPrefix, 16)
	if !success {
		return nil, errors.New("failed to convert hex string to big.Int")
	}
	return val, nil
}

func scaleBaseFeePerGas(value string) (*big.Int, error) {
	val, err := hexStringToBigInt(value)
	if err != nil {
		return nil, err
	}
	if baseFeeIncreaseFactor > 0 {
		valueDouble := new(big.Float).SetInt(val)
		valueDouble.Mul(valueDouble, big.NewFloat(baseFeeIncreaseFactor))
		scaledValue := new(big.Int)
		valueDouble.Int(scaledValue)
		return scaledValue, nil
	}
	return val, nil
}

func (f *FeeManager) getNonEIP1559SuggestedFees(ctx context.Context, chainID uint64) (*SuggestedFees, error) {
	backend, err := f.RPCClient.EthClient(chainID)
	if err != nil {
		return nil, err
	}
	gasPrice, err := backend.SuggestGasPrice(ctx)
	if err != nil {
		return nil, err
	}

	estimatedTime := f.TransactionEstimatedTimeV2Legacy(ctx, chainID, gasPrice)

	return &SuggestedFees{
		GasPrice: gasPrice,
		NonEIP1559Fees: &NonEIP1559Fees{
			GasPrice:      (*hexutil.Big)(gasPrice),
			EstimatedTime: estimatedTime,
		},
		EIP1559Enabled: false,
	}, nil
}

// getEIP1559SuggestedFees returns suggested fees for EIP-1559 compatible chains
// source https://github.com/brave/brave-core/blob/master/components/brave_wallet/browser/eth_gas_utils.cc
func getEIP1559SuggestedFees(chainID uint64, feeHistory *FeeHistory) (lowPriorityFee, avgPriorityFee, highPriorityFee, suggestedBaseFee *big.Int, err error) {
	if feeHistory == nil || !feeHistory.isEIP1559Compatible(chainID) {
		return nil, nil, nil, nil, ErrEIP1559IncompaibleChain
	}

	pendingBaseFee := feeHistory.BaseFeePerGas[len(feeHistory.BaseFeePerGas)-1]
	suggestedBaseFee, err = scaleBaseFeePerGas(pendingBaseFee)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	fallbackPriorityFee := big.NewInt(2e9) // 2 Gwei
	lowPriorityFee = new(big.Int).Set(fallbackPriorityFee)
	avgPriorityFee = new(big.Int).Set(fallbackPriorityFee)
	highPriorityFee = new(big.Int).Set(fallbackPriorityFee)

	if len(feeHistory.Reward) == 0 {
		return lowPriorityFee, avgPriorityFee, highPriorityFee, suggestedBaseFee, nil
	}

	priorityFees := make([][]*big.Int, 3)
	for i := 0; i < 3; i++ {
		currentPriorityFees := []*big.Int{}
		invalidData := false

		for _, r := range feeHistory.Reward {
			if len(r) != 3 {
				invalidData = true
				break
			}

			fee, err := hexStringToBigInt(r[i])
			if err != nil {
				invalidData = true
				break
			}
			currentPriorityFees = append(currentPriorityFees, fee)
		}

		if invalidData {
			return nil, nil, nil, nil, ErrInvalidRewardData
		}

		sort.Slice(currentPriorityFees, func(a, b int) bool {
			return currentPriorityFees[a].Cmp(currentPriorityFees[b]) < 0
		})

		percentileIndex := int(float64(len(currentPriorityFees)) * 0.5)
		if i == 0 {
			lowPriorityFee = currentPriorityFees[percentileIndex]
		} else if i == 1 {
			avgPriorityFee = currentPriorityFees[percentileIndex]
		} else {
			highPriorityFee = currentPriorityFees[percentileIndex]
		}

		priorityFees[i] = currentPriorityFees
	}

	// Adjust low priority fee if it's equal to avg
	lowIndex := int(float64(len(priorityFees[0])) * 0.5)
	for lowIndex > 0 && lowPriorityFee == avgPriorityFee {
		lowIndex--
		lowPriorityFee = priorityFees[0][lowIndex]
	}

	// Adjust high priority fee if it's equal to avg
	highIndex := int(float64(len(priorityFees[2])) * 0.5)
	for highIndex < len(priorityFees[2])-1 && highPriorityFee == avgPriorityFee {
		highIndex++
		highPriorityFee = priorityFees[2][highIndex]
	}

	return lowPriorityFee, avgPriorityFee, highPriorityFee, suggestedBaseFee, nil
}

func calculateNetworkCongestion(feeHistory *FeeHistory) float64 {
	if len(feeHistory.BaseFeePerGas) == 0 || len(feeHistory.Reward) == 0 || len(feeHistory.GasUsedRatio) == 0 {
		return 0.0
	}

	var totalBaseFee uint64
	for _, baseFee := range feeHistory.BaseFeePerGas {
		fee, err := hexStringToBigInt(baseFee)
		if err != nil {
			return 0.0
		}
		totalBaseFee = totalBaseFee + fee.Uint64()
	}
	avgBaseFee := float64(totalBaseFee) / float64(len(feeHistory.BaseFeePerGas))

	var totalPriorityFee uint64
	var countPriorityFees int
	for _, rewardSet := range feeHistory.Reward {
		for _, reward := range rewardSet {
			fee, err := hexStringToBigInt(reward)
			if err != nil {
				return 0.0
			}
			totalPriorityFee = totalPriorityFee + fee.Uint64()
			countPriorityFees++
		}
	}
	avgPriorityFee := float64(totalPriorityFee) / float64(countPriorityFees)

	var totalGasUsedRatio float64
	for _, gasUsedRatio := range feeHistory.GasUsedRatio {
		totalGasUsedRatio += gasUsedRatio
	}
	avgGasUsedRatio := totalGasUsedRatio / float64(len(feeHistory.GasUsedRatio))

	priorityFeeRatio := avgPriorityFee / avgBaseFee

	congestionScore := (priorityFeeRatio * priorityWeight) + (avgGasUsedRatio * gasUsedWeight)

	return congestionScore
}
